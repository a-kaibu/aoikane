//go:build js && wasm

package synthesis

import (
	"encoding/json"
	"syscall/js"
	"unsafe"

	"github.com/a-kaibu/aoikane/internal/onnx"
)

func browserVocoderParallelism() int {
	v := js.Global().Get("__aoikaneVocoderWorkers")
	if v.Type() == js.TypeNumber {
		return min(6, max(0, v.Int()))
	}
	return 0
}

func browserRender(in Input, profile onnx.OriginalKernelProfile) ([]byte, bool) {
	fn := js.Global().Get("__aoikaneRender")
	if fn.Type() != js.TypeFunction || profile.AVX512Convolutions || profile.AVX512Unary || profile.FMAEmbeddings || profile.FMASmallConvolutions || profile.FMAProjections || profile.Threads != 0 && profile.Threads != 6 {
		return nil, false
	}
	request, err := json.Marshal(in)
	if err != nil {
		return nil, false
	}
	result := make(chan js.Value, 1)
	resolve := js.FuncOf(func(_ js.Value, args []js.Value) any { result <- args[0]; return nil })
	reject := js.FuncOf(func(_ js.Value, _ []js.Value) any { result <- js.Null(); return nil })
	defer resolve.Release()
	defer reject.Release()
	fn.Invoke(string(request)).Call("then", resolve).Call("catch", reject)
	bytes := <-result
	if bytes.IsNull() || bytes.IsUndefined() || !bytes.InstanceOf(js.Global().Get("Uint8Array")) {
		return nil, false
	}
	size := bytes.Get("byteLength").Int()
	if size < 0 || size > 256*1024*1024 || size%2 != 0 {
		return nil, false
	}
	pcm := make([]byte, size)
	js.CopyBytesToGo(pcm, bytes)
	return pcm, true
}

func browserVocode(inputs []modelTensor, profile onnx.OriginalKernelProfile) ([]float32, bool) {
	fn := js.Global().Get("__aoikaneVocode")
	if fn.Type() != js.TypeFunction || profile.AVX512Convolutions || profile.AVX512Unary || profile.FMAEmbeddings || profile.FMASmallConvolutions || profile.FMAProjections || profile.Threads != 0 && profile.Threads != 6 {
		return nil, false
	}
	r := js.Global().Get("Object").New()
	frames := 0
	for _, t := range inputs {
		if len(t.Shape) != 3 {
			return nil, false
		}
		frames = int(t.Shape[2])
		b := js.Global().Get("Uint8Array").New(len(t.Data))
		js.CopyBytesToJS(b, t.Data)
		r.Set(t.Name, b)
	}
	r.Set("frames", frames)
	result := make(chan js.Value, 1)
	resolve := js.FuncOf(func(_ js.Value, args []js.Value) any { result <- args[0]; return nil })
	reject := js.FuncOf(func(_ js.Value, _ []js.Value) any { result <- js.Null(); return nil })
	defer resolve.Release()
	defer reject.Release()
	fn.Invoke(r).Call("then", resolve).Call("catch", reject)
	b := <-result
	if b.IsNull() || b.IsUndefined() || !b.InstanceOf(js.Global().Get("Uint8Array")) || b.Get("byteLength").Int() != frames*240*4 {
		return nil, false
	}
	wave := make([]float32, frames*240)
	js.CopyBytesToGo(unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(wave))), len(wave)*4), b)
	return wave, true
}
