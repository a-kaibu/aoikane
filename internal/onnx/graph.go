package onnx

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

// Value owns a contiguous tensor. Supported storage kinds are FLOAT (1),
// INT32 (6), INT64 (7), BOOL (9), and DOUBLE (11). Scalars have an empty Shape.
type Value struct {
	Shape []int64
	Kind  int
	F     []float32
	D     []float64
	I     []int64
	B     []bool
}
type graphAttr struct {
	i      int64
	ints   []int64
	f      float32
	text   string
	tensor *Value
	graph  *Graph
}
type graphNode struct {
	name, op        string
	inputs, outputs []string
	attrs           map[string]graphAttr
}

// Graph executes requested dependency paths, rejecting any operation on those
// paths that has not yet been migrated. It is not a complete ONNX interpreter.
type Graph struct {
	nodes    []graphNode
	initial  map[string]Value
	producer map[string]int
	outputs  []string
}

func OpenGraph(path string) (*Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	model, err := fields(data)
	if err != nil {
		return nil, err
	}
	gd, err := oneBytes(model, 7)
	if err != nil {
		return nil, err
	}
	return parseGraph(gd)
}

func parseGraph(data []byte) (*Graph, error) {
	fs, err := fields(data)
	if err != nil {
		return nil, err
	}
	g := &Graph{initial: make(map[string]Value), producer: make(map[string]int)}
	declared, err := bytesFields(fs, 12)
	if err != nil {
		return nil, err
	}
	for _, d := range declared {
		vf, err := fields(d)
		if err != nil {
			return nil, err
		}
		name, err := oneBytes(vf, 1)
		if err != nil {
			return nil, err
		}
		g.outputs = append(g.outputs, string(name))
	}
	tensors, err := bytesFields(fs, 5)
	if err != nil {
		return nil, err
	}
	for _, td := range tensors {
		tf, err := fields(td)
		if err != nil {
			return nil, err
		}
		name, err := oneBytes(tf, 8)
		if err != nil {
			return nil, err
		}
		v, err := tensorValue(tf)
		if err != nil {
			return nil, fmt.Errorf("initializer %s: %w", name, err)
		}
		if _, ok := g.initial[string(name)]; ok {
			return nil, fmt.Errorf("duplicate initializer")
		}
		g.initial[string(name)] = v
	}
	nodes, err := bytesFields(fs, 1)
	if err != nil {
		return nil, err
	}
	for _, nd := range nodes {
		nf, err := fields(nd)
		if err != nil {
			return nil, err
		}
		op, err := oneBytes(nf, 4)
		if err != nil {
			return nil, err
		}
		n := graphNode{op: string(op), attrs: make(map[string]graphAttr)}
		for _, f := range nf {
			if f.number == 3 {
				n.name = string(f.data)
			}
			if f.number == 7 && len(f.data) > 0 && string(f.data) != "ai.onnx" {
				return nil, fmt.Errorf("custom node domain unsupported")
			}
		}
		in, err := bytesFields(nf, 1)
		if err != nil {
			return nil, err
		}
		out, err := bytesFields(nf, 2)
		if err != nil {
			return nil, err
		}
		for _, s := range in {
			n.inputs = append(n.inputs, string(s))
		}
		for _, s := range out {
			n.outputs = append(n.outputs, string(s))
		}
		attrs, err := bytesFields(nf, 5)
		if err != nil {
			return nil, err
		}
		for _, ad := range attrs {
			af, err := fields(ad)
			if err != nil {
				return nil, err
			}
			name, err := oneBytes(af, 1)
			if err != nil {
				return nil, err
			}
			a := graphAttr{}
			for _, f := range af {
				switch f.number {
				case 2:
					if f.wire != 5 {
						return nil, fmt.Errorf("invalid FLOAT attribute")
					}
					a.f = math.Float32frombits(binary.LittleEndian.Uint32(f.data))
				case 3:
					a.i = int64(f.value)
				case 4:
					a.text = string(f.data)
				case 5:
					tf, err := fields(f.data)
					if err != nil {
						return nil, err
					}
					v, err := tensorValue(tf)
					if err != nil {
						return nil, err
					}
					a.tensor = &v
				case 6:
					a.graph, err = parseGraph(f.data)
					if err != nil {
						return nil, err
					}
				}
			}
			a.ints, err = intValues(af, 8)
			if err != nil {
				return nil, err
			}
			if _, ok := n.attrs[string(name)]; ok {
				return nil, fmt.Errorf("duplicate node attribute")
			}
			n.attrs[string(name)] = a
		}
		for _, name := range n.outputs {
			if _, ok := g.producer[name]; ok {
				return nil, fmt.Errorf("duplicate graph producer")
			}
			if _, ok := g.initial[name]; ok {
				return nil, fmt.Errorf("producer overwrites initializer")
			}
			g.producer[name] = len(g.nodes)
		}
		g.nodes = append(g.nodes, n)
	}
	return g, nil
}

func tensorValue(fs []field) (Value, error) {
	v := Value{}
	for _, f := range fs {
		if f.number == 2 {
			if f.wire != 0 {
				return v, fmt.Errorf("invalid tensor type")
			}
			v.Kind = int(f.value)
		}
		if f.number == 13 || (f.number == 14 && f.value != 0) {
			return v, fmt.Errorf("external tensor unsupported")
		}
	}
	dims, err := intValues(fs, 1)
	if err != nil {
		return v, err
	}
	v.Shape = dims
	n, err := valueCount(dims)
	if err != nil {
		return v, err
	}
	raw, err := oneBytes(fs, 9)
	if err != nil {
		return v, err
	}
	width := 0
	switch v.Kind {
	case 1, 6:
		width = 4
	case 7, 11:
		width = 8
	case 9:
		width = 1
	default:
		return v, fmt.Errorf("tensor kind %d unsupported", v.Kind)
	}
	if n > int(^uint(0)>>1)/width || len(raw) != n*width {
		return v, fmt.Errorf("tensor data length differs")
	}
	switch v.Kind {
	case 1:
		v.F = make([]float32, n)
		for i := range v.F {
			v.F[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		}
	case 11:
		v.D = make([]float64, n)
		for i := range v.D {
			v.D[i] = math.Float64frombits(binary.LittleEndian.Uint64(raw[i*8:]))
		}
	case 6, 7:
		v.I = make([]int64, n)
		for i := range v.I {
			if v.Kind == 6 {
				v.I[i] = int64(int32(binary.LittleEndian.Uint32(raw[i*4:])))
			} else {
				v.I[i] = int64(binary.LittleEndian.Uint64(raw[i*8:]))
			}
		}
	case 9:
		v.B = make([]bool, n)
		for i := range v.B {
			v.B[i] = raw[i] != 0
		}
	}
	return v, nil
}

func valueCount(shape []int64) (int, error) {
	n := 1
	for _, d := range shape {
		if d < 0 || d > int64(int(^uint(0)>>1)) {
			return 0, fmt.Errorf("invalid tensor shape")
		}
		if d == 0 {
			n = 0
			continue
		}
		if n > int(^uint(0)>>1)/int(d) {
			return 0, fmt.Errorf("tensor shape overflows")
		}
		n *= int(d)
	}
	return n, nil
}

func (v Value) validate() error {
	n, err := valueCount(v.Shape)
	if err != nil {
		return err
	}
	switch v.Kind {
	case 1:
		if len(v.F) == n && len(v.I) == 0 && len(v.B) == 0 && len(v.D) == 0 {
			return nil
		}
	case 6, 7:
		if len(v.I) == n && len(v.F) == 0 && len(v.B) == 0 && len(v.D) == 0 {
			return nil
		}
	case 9:
		if len(v.B) == n && len(v.F) == 0 && len(v.I) == 0 && len(v.D) == 0 {
			return nil
		}
	case 11:
		if len(v.D) == n && len(v.F) == 0 && len(v.I) == 0 && len(v.B) == 0 {
			return nil
		}
	}
	return fmt.Errorf("tensor storage differs from shape/kind")
}

func (g *Graph) Run(inputs map[string]Value, outputs []string) (map[string]Value, error) {
	return g.runGraph(inputs, outputs, false)
}

// RunOriginalKernels uses reconstructed MKL/oneDNN Conv, linear, Exp, Log, Sin and Tanh.
// Other operations currently retain the ONNX backend's numeric order; this is
// an in-progress original-engine reconstruction, not an exactness guarantee.
func (g *Graph) RunOriginalKernels(inputs map[string]Value, outputs []string) (map[string]Value, error) {
	return g.runGraph(inputs, outputs, true)
}

// RunOriginalKernelsWithThreads selects a numerical thread partition without
// changing shared graph state or the number of Go execution workers.
// One-thread and six-thread FeatNet/vocoder model paths are reconstructed.
func (g *Graph) RunOriginalKernelsWithThreads(inputs map[string]Value, outputs []string, threads int) (map[string]Value, error) {
	if threads != 1 && threads != 6 {
		return nil, fmt.Errorf("original thread profile %d is not reconstructed", threads)
	}
	return g.runGraph(inputs, outputs, true, threads)
}

// OriginalKernelProfile selects reconstructed numerical orders independently
// of the host CPU and Go worker count. AVX512Convolutions covers the audited
// one-thread AMD profile; other CPU-specific kernels remain under reconstruction.
type OriginalKernelProfile struct {
	Threads            int
	AVX512Convolutions bool
	// AVX512Unary selects the reconstructed Intel VML Exp/Log/Sin/Tanh orders.
	AVX512Unary bool
	// FMAEmbeddings selects the reconstructed one-thread Intel embedding order.
	// Remaining linear projections retain their existing profile.
	FMAEmbeddings bool
	// FMASmallConvolutions selects audited Intel im2col FMA orders:
	// serial small reductions and blocked large reductions.
	// The field name is retained for compatibility; oneDNN filters are separate.
	FMASmallConvolutions bool
	// FMAProjections selects the reconstructed Intel serial and AVX512
	// single-output projection paths.
	FMAProjections bool
}

func (g *Graph) RunOriginalKernelsWithProfile(inputs map[string]Value, outputs []string, profile OriginalKernelProfile) (map[string]Value, error) {
	if profile.Threads != 1 && profile.Threads != 6 || (profile.AVX512Convolutions || profile.FMAEmbeddings || profile.FMASmallConvolutions || profile.FMAProjections) && profile.Threads != 1 {
		return nil, fmt.Errorf("original numerical profile is not reconstructed")
	}
	isa := 0
	if profile.AVX512Convolutions {
		isa = 1
	}
	unary, embedding := 0, 0
	if profile.AVX512Unary {
		unary = 1
	}
	if profile.FMAEmbeddings {
		embedding = 1
	}
	pointwise := 0
	if profile.FMASmallConvolutions {
		pointwise = 1
	}
	projection := 0
	if profile.FMAProjections {
		projection = 1
	}
	return g.runGraph(inputs, outputs, true, profile.Threads, isa, unary, embedding, pointwise, projection)
}

func originalAVX512Unary(profile []int) bool {
	return len(profile) > 2 && profile[2] == 1
}
func originalFMAEmbeddings(profile []int) bool {
	return len(profile) > 3 && profile[3] == 1
}

func originalFMASmallConv(profile []int) bool {
	return len(profile) > 4 && profile[4] == 1
}

func originalFMAProjections(profile []int) bool {
	return len(profile) > 5 && profile[5] == 1
}

func originalAVX512Convolutions(profile []int) bool {
	return len(profile) > 1 && profile[1] == 1
}

func originalThreadCount(threads []int) int {
	if len(threads) == 0 {
		return 6
	}
	return threads[0]
}

func (g *Graph) runGraph(inputs map[string]Value, outputs []string, original bool, threads ...int) (map[string]Value, error) {
	values, err := g.runValues(inputs, outputs, nil, original, threads...)
	if err != nil {
		return nil, err
	}
	for k, v := range values {
		values[k] = cloneValue(v)
	}
	return values, nil
}

func (g *Graph) runValues(inputs map[string]Value, outputs []string, parent func(string) (Value, error), original bool, threads ...int) (map[string]Value, error) {
	if len(outputs) == 0 {
		return nil, fmt.Errorf("requested outputs are empty")
	}
	values := make(map[string]Value, len(g.initial)+len(inputs))
	for k, v := range g.initial {
		values[k] = v
	}
	for k, v := range inputs {
		if _, ok := values[k]; ok {
			return nil, fmt.Errorf("input overrides initializer %s", k)
		}
		if _, ok := g.producer[k]; ok {
			return nil, fmt.Errorf("input overrides computed value %s", k)
		}
		if err := v.validate(); err != nil {
			return nil, fmt.Errorf("input %s: %w", k, err)
		}
		values[k] = v
	}
	state := make([]uint8, len(g.nodes))
	var resolve func(string) (Value, error)
	resolve = func(name string) (Value, error) {
		if v, ok := values[name]; ok {
			return v, nil
		}
		index, ok := g.producer[name]
		if !ok {
			if parent != nil {
				return parent(name)
			}
			return Value{}, fmt.Errorf("missing graph value %s", name)
		}
		if state[index] == 1 {
			return Value{}, fmt.Errorf("cyclic graph")
		}
		state[index] = 1
		n := g.nodes[index]
		var in []Value
		for _, key := range n.inputs {
			if key == "" {
				in = append(in, Value{})
				continue
			}
			v, err := resolve(key)
			if err != nil {
				return Value{}, err
			}
			in = append(in, v)
		}
		var out []Value
		var err error
		if n.op == "If" {
			out, err = executeIf(n, in, resolve, original, threads...)
		} else {
			out, err = executeNode(n, in, original, threads...)
		}
		if err != nil {
			return Value{}, fmt.Errorf("%s (%s): %w", n.name, n.op, err)
		}
		if len(out) != len(n.outputs) {
			return Value{}, fmt.Errorf("node output count differs")
		}
		for i, v := range out {
			if err := v.validate(); err != nil {
				return Value{}, err
			}
			values[n.outputs[i]] = v
		}
		state[index] = 2
		return values[name], nil
	}
	result := make(map[string]Value, len(outputs))
	for _, key := range outputs {
		v, err := resolve(key)
		if err != nil {
			return nil, err
		}
		result[key] = v
	}
	return result, nil
}

func executeIf(n graphNode, in []Value, resolve func(string) (Value, error), original bool, threads ...int) ([]Value, error) {
	if len(in) != 1 || in[0].Kind != 9 || len(in[0].B) != 1 {
		return nil, fmt.Errorf("If requires single-element BOOL condition")
	}
	for name := range n.attrs {
		if name != "then_branch" && name != "else_branch" {
			return nil, fmt.Errorf("unsupported If attribute")
		}
	}
	if n.attrs["then_branch"].graph == nil || n.attrs["else_branch"].graph == nil {
		return nil, fmt.Errorf("If requires both graphs")
	}
	branch := n.attrs["else_branch"].graph
	if in[0].B[0] {
		branch = n.attrs["then_branch"].graph
	}
	values, err := branch.runValues(nil, branch.outputs, resolve, original, threads...)
	if err != nil {
		return nil, err
	}
	out := make([]Value, len(branch.outputs))
	for i, key := range branch.outputs {
		out[i] = values[key]
	}
	return out, nil
}

func cloneValue(v Value) Value {
	v.Shape = append([]int64(nil), v.Shape...)
	v.F = append([]float32(nil), v.F...)
	v.D = append([]float64(nil), v.D...)
	v.I = append([]int64(nil), v.I...)
	v.B = append([]bool(nil), v.B...)
	return v
}
