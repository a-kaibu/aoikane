package onnx

import (
	"sync"
	"unsafe"
	"weak"
)

// Large FLOAT intermediates are recycled between graph nodes. Reuse only
// avoids fresh allocations (and their page faults); every buffer is cleared
// before it is handed out again, so values are unaffected.
const (
	pooledMinFloats = 4096
	pooledMaxBytes  = 384 << 20
)

type bufferPool[T any] struct {
	mu    sync.Mutex
	free  map[int][][]T
	bytes int
	// live records buffers handed out and not yet returned. Weak pointers
	// let the collector reclaim a buffer that is never returned; a later
	// allocation at the same address is then not mistaken for it.
	live map[uintptr]liveBuffer[T]
}

type liveBuffer[T any] struct {
	ptr weak.Pointer[T]
	n   int
}

var (
	floats = bufferPool[float32]{free: make(map[int][][]float32), live: make(map[uintptr]liveBuffer[float32])}
	ints   = bufferPool[int64]{free: make(map[int][][]int64), live: make(map[uintptr]liveBuffer[int64])}
)

// lookup returns the live record of b, if b is exactly a handed-out buffer.
func (p *bufferPool[T]) lookup(b []T) (uintptr, bool) {
	ptr := unsafe.SliceData(b)
	key := uintptr(unsafe.Pointer(ptr))
	r, ok := p.live[key]
	return key, ok && r.n == len(b) && r.ptr.Value() == ptr
}

// getFloats returns n zeroed floats, reusing a released buffer when possible.
func getFloats(n int) []float32 { return floats.get(n, true) }

// getFloatsUncleared may return stale contents; callers overwrite every element.
func getFloatsUncleared(n int) []float32 { return floats.get(n, false) }

func putFloats(b []float32) { floats.put(b) }

func getInts(n int) []int64 { return ints.get(n, true) }

func (p *bufferPool[T]) get(n int, zero bool) []T {
	if n < pooledMinFloats {
		return make([]T, n)
	}
	p.mu.Lock()
	var b []T
	if list := p.free[n]; len(list) > 0 {
		b = list[len(list)-1]
		p.free[n] = list[:len(list)-1]
		p.bytes -= int(unsafe.Sizeof(b[0])) * n
	}
	if b == nil {
		p.mu.Unlock()
		b = make([]T, n)
		p.mu.Lock()
	} else if zero {
		clear(b)
	}
	if len(p.live) >= 4096 {
		for key, r := range p.live {
			if r.ptr.Value() == nil {
				delete(p.live, key)
			}
		}
	}
	ptr := unsafe.SliceData(b)
	p.live[uintptr(unsafe.Pointer(ptr))] = liveBuffer[T]{weak.Make(ptr), n}
	p.mu.Unlock()
	return b
}

// pooled reports whether b is exactly a buffer currently handed out by get,
// so ownership can be tracked by the graph executor.
func (p *bufferPool[T]) pooled(b []T) bool {
	if len(b) < pooledMinFloats {
		return false
	}
	p.mu.Lock()
	_, ok := p.lookup(b)
	p.mu.Unlock()
	return ok
}

// put returns a buffer from get once nothing references it.
func (p *bufferPool[T]) put(b []T) {
	if len(b) == 0 {
		return
	}
	size := int(unsafe.Sizeof(b[0])) * len(b)
	p.mu.Lock()
	defer p.mu.Unlock()
	key, ok := p.lookup(b)
	if !ok {
		return
	}
	delete(p.live, key)
	if p.bytes+size > pooledMaxBytes {
		return
	}
	p.free[len(b)] = append(p.free[len(b)], b)
	p.bytes += size
}

// bufferLife tracks which pooled buffers the values of one graph run hold.
// A buffer is released when the last value referring to any part of it has
// no remaining consumers.
type bufferLife struct {
	uses    map[string]int
	holders map[unsafe.Pointer]int
	owned   map[unsafe.Pointer]ownedBuffer
	byName  map[string][]unsafe.Pointer
}

type ownedBuffer struct {
	bytes   uintptr
	release func()
}

func newBufferLife(uses map[string]int) *bufferLife {
	return &bufferLife{uses: uses, holders: make(map[unsafe.Pointer]int), owned: make(map[unsafe.Pointer]ownedBuffer), byName: make(map[string][]unsafe.Pointer)}
}

func (l *bufferLife) owner(p unsafe.Pointer, bytes uintptr) unsafe.Pointer {
	start, end := uintptr(p), uintptr(p)+bytes
	for q, b := range l.owned {
		if start >= uintptr(q) && end <= uintptr(q)+b.bytes {
			return q
		}
	}
	return nil
}

func (l *bufferLife) hold(name string, p unsafe.Pointer, bytes uintptr, isPooled bool, release func()) {
	q := l.owner(p, bytes)
	if q == nil {
		if !isPooled {
			return
		}
		q = p
		l.owned[q] = ownedBuffer{bytes, release}
	}
	l.holders[q]++
	l.byName[name] = append(l.byName[name], q)
}

// produced registers a node output. Pooled outputs become owned buffers;
// outputs that alias an owned buffer keep it alive.
func (l *bufferLife) produced(name string, v Value) {
	if f := v.F; len(f) > 0 {
		l.hold(name, unsafe.Pointer(unsafe.SliceData(f)), uintptr(4*len(f)), floats.pooled(f), func() { putFloats(f) })
	}
	if i := v.I; len(i) > 0 {
		l.hold(name, unsafe.Pointer(unsafe.SliceData(i)), uintptr(8*len(i)), ints.pooled(i), func() { ints.put(i) })
	}
}

// consumed is called once per input reference after a node has run.
func (l *bufferLife) consumed(name string) {
	n, ok := l.uses[name]
	if !ok {
		return
	}
	n--
	l.uses[name] = n
	if n > 0 {
		return
	}
	for _, q := range l.byName[name] {
		l.holders[q]--
		if l.holders[q] == 0 {
			b := l.owned[q]
			delete(l.owned, q)
			delete(l.holders, q)
			b.release()
		}
	}
	delete(l.byName, name)
}

// release returns every buffer still owned at the end of a run.
func (l *bufferLife) release() {
	for q, b := range l.owned {
		delete(l.owned, q)
		b.release()
	}
}

// graphUses counts node-input references of computed values. Values read by
// subgraphs or requested as outputs are never released.
func (g *Graph) graphUses(outputs []string) map[string]int {
	g.usesOnce.Do(func() {
		g.uses = make(map[string]int)
		for _, n := range g.nodes {
			for _, in := range n.inputs {
				if _, ok := g.producer[in]; ok {
					g.uses[in]++
				}
			}
		}
		g.pinned = make(map[string]bool)
		var visit func(*Graph)
		visit = func(sub *Graph) {
			for _, n := range sub.nodes {
				for _, in := range n.inputs {
					g.pinned[in] = true
				}
				for _, a := range n.attrs {
					if a.graph != nil {
						visit(a.graph)
					}
				}
			}
			for _, out := range sub.outputs {
				g.pinned[out] = true
			}
		}
		for _, n := range g.nodes {
			for _, a := range n.attrs {
				if a.graph != nil {
					visit(a.graph)
				}
			}
		}
		for _, out := range g.outputs {
			g.pinned[out] = true
		}
	})
	uses := make(map[string]int, len(g.uses))
	for k, v := range g.uses {
		if !g.pinned[k] {
			uses[k] = v
		}
	}
	for _, out := range outputs {
		delete(uses, out)
	}
	return uses
}
