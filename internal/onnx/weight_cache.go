package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"
)

// A weight cache holds the FLOAT initializers of one model file that are not
// 4-byte aligned in it, copied to aligned offsets. Mapping the cache lets
// those weights alias read-only memory as well, so start-up copies nothing.
// Entries follow the parse order of the model's tensors.

const weightCacheMagic = "AOIKF32\x01"

type weightCache struct {
	path    string
	sizes   []int
	offsets []int
	blob    []byte
	next    int
	failed  bool
	collect [][]byte
}

// openWeightCache returns the cache for model path, or one that records
// the weights to write when the cache is missing. It returns nil when no
// cache can be used.
func openWeightCache(model string) *weightCache {
	if !littleEndianHost || !canMapWeights || os.Getenv("AOIKANE_NO_CACHE") != "" {
		return nil
	}
	abs, err := filepath.Abs(model)
	if err != nil {
		return nil
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return nil
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("weights\x00%s\x00%d\x00%d", abs, info.Size(), info.ModTime().UnixNano())))
	ws := &weightCache{path: filepath.Join(base, "aoikane", "parsed", hex.EncodeToString(h[:16])+".f32")}
	if b, _, err := readModel(ws.path); err == nil && ws.load(b) {
		ws.blob = b
	}
	return ws
}

func (ws *weightCache) load(b []byte) bool {
	if len(b) < 16 || string(b[:8]) != weightCacheMagic {
		return false
	}
	count := binary.LittleEndian.Uint64(b[8:])
	if count > uint64(len(b))/8 {
		return false
	}
	at := align64(16 + 8*int(count))
	for i := 0; i < int(count); i++ {
		n := int(binary.LittleEndian.Uint64(b[16+8*i:]))
		if n <= 0 || at+4*n > len(b) {
			return false
		}
		ws.sizes = append(ws.sizes, n)
		ws.offsets = append(ws.offsets, at)
		at = align64(at + 4*n)
	}
	return uintptr(unsafe.Pointer(&b[0]))%64 == 0
}

func align64(n int) int { return (n + 63) &^ 63 }

// take returns the cached copy of the next unaligned tensor, or nil after
// recording raw for the cache that will be written.
func (ws *weightCache) take(raw []byte, n int) []float32 {
	if ws == nil || n < 1024 {
		return nil
	}
	if ws.blob == nil {
		ws.collect = append(ws.collect, raw)
		return nil
	}
	k := ws.next
	ws.next++
	if ws.failed || k >= len(ws.sizes) || ws.sizes[k] != n {
		ws.failed = true
		return nil
	}
	return unsafe.Slice((*float32)(unsafe.Pointer(&ws.blob[ws.offsets[k]])), n)
}

// finish writes a missing cache after a successful parse.
func (ws *weightCache) finish() {
	if ws == nil || ws.blob != nil || len(ws.collect) == 0 {
		return
	}
	size := align64(16 + 8*len(ws.collect))
	for _, raw := range ws.collect {
		size = align64(size + len(raw))
	}
	b := make([]byte, size)
	copy(b, weightCacheMagic)
	binary.LittleEndian.PutUint64(b[8:], uint64(len(ws.collect)))
	at := align64(16 + 8*len(ws.collect))
	for i, raw := range ws.collect {
		binary.LittleEndian.PutUint64(b[16+8*i:], uint64(len(raw)/4))
		copy(b[at:], raw)
		at = align64(at + len(raw))
	}
	dir := filepath.Dir(ws.path)
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(b)
	cerr := tmp.Close()
	if werr != nil || cerr != nil || os.Rename(tmp.Name(), ws.path) != nil {
		os.Remove(tmp.Name())
	}
}
