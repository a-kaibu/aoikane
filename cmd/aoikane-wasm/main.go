//go:build js && wasm

// Command aoikane-wasm exposes the portable synthesizer to a web page. The
// page fetches the model assets and registers them with assetfs before
// calling warmup or synthesize.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall/js"

	"runtime"
	"runtime/debug"
	"unsafe"

	"github.com/a-kaibu/aoikane/internal/assetfs"
	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/onnx"
	"github.com/a-kaibu/aoikane/internal/portable"
	"github.com/a-kaibu/aoikane/internal/synthesis"
)

var version = "devel"

var renderer *synthesis.Engine
var rendererMu sync.Mutex
var warmupMu sync.Mutex
var preparedVoices = map[string]bool{}

var (
	engineMu  sync.Mutex
	synthMu   sync.Mutex
	eng       *portable.Engine
	kept      []js.Func
	vocoder   *onnx.Graph
	vocoderMu sync.Mutex
)

func main() {
	api := js.Global().Get("Object").New()
	api.Set("version", version)
	api.Set("registerAsset", keep(js.FuncOf(registerAsset)))
	api.Set("warmup", keep(js.FuncOf(warmup)))
	api.Set("voices", keep(js.FuncOf(voices)))
	api.Set("synthesize", keep(js.FuncOf(synthesize)))
	api.Set("warmupVocoder", keep(js.FuncOf(warmupVocoder)))
	api.Set("vocode", keep(js.FuncOf(vocode)))
	api.Set("warmupRenderer", keep(js.FuncOf(warmupRenderer)))
	api.Set("render", keep(js.FuncOf(render)))
	js.Global().Set("aoikane", api)
	if started := js.Global().Get("__aoikaneStarted"); started.Type() == js.TypeFunction {
		started.Invoke()
	}
	select {}
}

func warmupVocoder(_ js.Value, _ []js.Value) any {
	return newPromise(func() (any, error) {
		vocoderMu.Lock()
		defer vocoderMu.Unlock()
		if vocoder == nil {
			onnx.SetFastMath(true)
			g, err := onnx.OpenGraph("assets/synth/vocoder.onnx")
			if err != nil {
				return nil, err
			}
			vocoder = g
			assetfs.Remove("assets/synth/vocoder.onnx")
			debug.SetGCPercent(200)
			debug.SetMemoryLimit(256 << 20)
		}
		return js.Null(), nil
	})
}

func vocode(_ js.Value, args []js.Value) any {
	if len(args) != 1 || args[0].Type() != js.TypeObject || args[0].IsNull() {
		return rejected(errors.New("vocode expects a tensor request"))
	}
	r := args[0]
	return newPromise(func() (any, error) {
		vocoderMu.Lock()
		defer vocoderMu.Unlock()
		if vocoder == nil {
			return nil, errors.New("vocoder is not initialized")
		}
		frames := r.Get("frames")
		if frames.Type() != js.TypeNumber || frames.Int() < 1 || frames.Int() > 110 {
			return nil, errors.New("invalid vocoder frame count")
		}
		n := frames.Int()
		if float64(n) != frames.Float() {
			return nil, errors.New("vocoder frame count must be an integer")
		}
		inputs := map[string]onnx.Value{}
		for _, p := range []struct {
			name     string
			channels int
		}{{"f0", 1}, {"bap", 5}, {"spc", 60}} {
			b := r.Get(p.name)
			if !b.InstanceOf(js.Global().Get("Uint8Array")) || b.Get("byteLength").Int() != p.channels*n*4 {
				return nil, fmt.Errorf("invalid %s tensor", p.name)
			}
			f := make([]float32, p.channels*n)
			js.CopyBytesToGo(unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(f))), len(f)*4), b)
			inputs[p.name] = onnx.Value{Kind: 1, Shape: []int64{1, int64(p.channels), int64(n)}, F: f}
		}
		out, err := vocoder.RunOriginalKernelsWithProfile(inputs, []string{"wave"}, onnx.OriginalKernelProfile{Threads: 6, Reordered: true})
		if err != nil {
			return nil, err
		}
		wave := out["wave"].F
		if len(wave) != n*240 {
			return nil, errors.New("vocoder output size differs")
		}
		bytes := js.Global().Get("Uint8Array").New(len(wave) * 4)
		js.CopyBytesToJS(bytes, unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(wave))), len(wave)*4))
		return bytes, nil
	})
}

func keep(f js.Func) js.Func {
	kept = append(kept, f)
	return f
}

func warmupRenderer(_ js.Value, _ []js.Value) any {
	return newPromise(func() (any, error) {
		rendererMu.Lock()
		defer rendererMu.Unlock()
		if renderer == nil {
			onnx.SetFastMath(true)
			e, err := synthesis.OpenWithThreads("assets/synth", 6)
			if err != nil {
				return nil, err
			}
			renderer = e
			for _, name := range []string{"assets/synth/featnet.onnx", "assets/synth/vocoder.onnx", "assets/synth/compressor.json"} {
				assetfs.Remove(name)
			}
			debug.SetGCPercent(200)
			debug.SetMemoryLimit(512 << 20)
		}
		return js.Null(), nil
	})
}

func render(_ js.Value, args []js.Value) any {
	if len(args) != 1 || args[0].Type() != js.TypeString {
		return rejected(errors.New("render expects a JSON input"))
	}
	request := args[0].String()
	return newPromise(func() (any, error) {
		rendererMu.Lock()
		e := renderer
		rendererMu.Unlock()
		if e == nil {
			return nil, errors.New("renderer is not initialized")
		}
		var input synthesis.Input
		if err := json.Unmarshal([]byte(request), &input); err != nil {
			return nil, err
		}
		pcm, err := e.Render(context.Background(), input)
		if err != nil {
			return nil, err
		}
		out := js.Global().Get("Uint8Array").New(len(pcm))
		js.CopyBytesToJS(out, pcm)
		return out, nil
	})
}

// registerAsset stores one fetched file. It returns false for malformed calls.
func registerAsset(_ js.Value, args []js.Value) any {
	if len(args) != 2 || args[0].Type() != js.TypeString || args[1].Type() != js.TypeObject {
		return false
	}
	size := args[1].Get("byteLength")
	if size.Type() != js.TypeNumber {
		return false
	}
	n := size.Int()
	if n < 0 {
		return false
	}
	b := make([]byte, n)
	if js.CopyBytesToGo(b, args[1]) != n {
		return false
	}
	assetfs.Put(args[0].String(), b)
	return true
}

func warmup(_ js.Value, args []js.Value) any {
	voice := "aoi"
	if len(args) > 0 && args[0].Type() == js.TypeString && args[0].String() != "" {
		voice = args[0].String()
	}
	switch strings.ToLower(voice) {
	case "aoi", "葵":
		voice = portable.VoiceAoi
	case "akane", "茜":
		voice = portable.VoiceAkane
	}
	return newPromise(func() (any, error) {
		warmupMu.Lock()
		defer warmupMu.Unlock()
		e, err := ensureEngine()
		if err != nil {
			return nil, err
		}
		if !preparedVoices[voice] {
			// Include lazy voice-dictionary setup in model loading, not synthesis.
			if _, err := e.ReadingVoice("", voice); err != nil {
				return nil, err
			}
			preparedVoices[voice] = true
			// Model tensors own their parsed storage. Keep dictionaries, but drop
			// the original neural-model blobs and collect parsing temporaries once.
			for _, name := range []string{"assets/synth/featnet.onnx", "assets/synth/vocoder.onnx", "assets/synth/compressor.json"} {
				assetfs.Remove(name)
			}
			runtime.GC()
		}
		return js.Null(), nil
	})
}

func voices(_ js.Value, _ []js.Value) any {
	list := js.Global().Get("Array").New(2)
	for i, v := range []engine.Voice{
		{ID: portable.VoiceAoi, Name: "aoi", SampleRate: 48000},
		{ID: portable.VoiceAkane, Name: "akane", SampleRate: 48000},
	} {
		item := js.Global().Get("Object").New()
		item.Set("id", v.ID)
		item.Set("name", v.Name)
		item.Set("sampleRate", v.SampleRate)
		list.SetIndex(i, item)
	}
	return list
}

func synthesize(_ js.Value, args []js.Value) any {
	if len(args) != 1 || args[0].Type() != js.TypeString {
		return rejected(errors.New("synthesize expects one JSON request string"))
	}
	request := args[0].String()
	return newPromise(func() (any, error) {
		var r engine.Request
		if err := json.Unmarshal([]byte(request), &r); err != nil {
			return nil, fmt.Errorf("invalid request: %w", err)
		}
		r.Format = "wav"
		switch strings.ToLower(r.Voice) {
		case "", "aoi", "葵":
			r.Voice = portable.VoiceAoi
		case "akane", "茜":
			r.Voice = portable.VoiceAkane
		}
		if err := r.Validate(); err != nil {
			return nil, err
		}
		textEngine, err := ensureEngine()
		if err != nil {
			return nil, err
		}
		synthMu.Lock()
		defer synthMu.Unlock()
		wav, cues, err := textEngine.SynthesizeTimed(context.Background(), r)
		if err != nil {
			return nil, err
		}
		result := js.Global().Get("Object").New()
		audio := js.Global().Get("Uint8Array").New(len(wav))
		js.CopyBytesToJS(audio, wav)
		result.Set("audio", audio)
		list := js.Global().Get("Array").New(len(cues))
		for i, cue := range cues {
			item := js.Global().Get("Object").New()
			item.Set("text", cue.Text)
			item.Set("start", cue.Start.Seconds())
			item.Set("end", cue.End.Seconds())
			list.SetIndex(i, item)
		}
		result.Set("cues", list)
		return result, nil
	})
}

func ensureEngine() (*portable.Engine, error) {
	engineMu.Lock()
	defer engineMu.Unlock()
	if eng != nil {
		return eng, nil
	}
	onnx.SetFastMath(true)
	e, err := portable.Open(portable.Options{AssetsDir: "assets"})
	if err != nil {
		return nil, fmt.Errorf("model load failed: %w", err)
	}
	eng = e
	return e, nil
}

func newPromise(fn func() (any, error)) js.Value {
	var callback js.Func
	callback = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 2 {
			callback.Release()
			return nil
		}
		resolve, reject := args[0], args[1]
		go func() {
			defer callback.Release()
			defer func() {
				if recovered := recover(); recovered != nil {
					reject.Invoke(js.Global().Get("Error").New(fmt.Sprint(recovered)))
				}
			}()
			value, err := fn()
			if err != nil {
				reject.Invoke(js.Global().Get("Error").New(err.Error()))
				return
			}
			resolve.Invoke(value)
		}()
		return nil
	})
	return js.Global().Get("Promise").New(callback)
}

func rejected(err error) js.Value {
	var callback js.Func
	callback = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 1 {
			args[1].Invoke(js.Global().Get("Error").New(err.Error()))
		}
		callback.Release()
		return nil
	})
	return js.Global().Get("Promise").New(callback)
}
