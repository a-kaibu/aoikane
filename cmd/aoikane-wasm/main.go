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

	"github.com/a-kaibu/aoikane/internal/assetfs"
	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/onnx"
	"github.com/a-kaibu/aoikane/internal/portable"
)

var version = "devel"

var (
	engineMu sync.Mutex
	synthMu  sync.Mutex
	eng      *portable.Engine
	kept     []js.Func
)

func main() {
	api := js.Global().Get("Object").New()
	api.Set("version", version)
	api.Set("registerAsset", keep(js.FuncOf(registerAsset)))
	api.Set("warmup", keep(js.FuncOf(warmup)))
	api.Set("voices", keep(js.FuncOf(voices)))
	api.Set("synthesize", keep(js.FuncOf(synthesize)))
	js.Global().Set("aoikane", api)
	if started := js.Global().Get("__aoikaneStarted"); started.Type() == js.TypeFunction {
		started.Invoke()
	}
	select {}
}

func keep(f js.Func) js.Func {
	kept = append(kept, f)
	return f
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

func warmup(_ js.Value, _ []js.Value) any {
	return newPromise(func() (any, error) {
		if _, err := ensureEngine(); err != nil {
			return nil, err
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
