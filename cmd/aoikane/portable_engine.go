//go:build (windows || darwin || linux) && (amd64 || arm64)

package main

import (
	"flag"
	"fmt"
	"github.com/a-kaibu/aoikane/internal/onnx"
	"os"

	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/portable"
)

// openEngine uses the reconstructed frontend and models. Embedded data is
// the default; --assets selects an external directory.
// exactMath keeps the bit-exact reconstruction of the original kernels.
// The default reorders floating-point accumulation for speed, which can
// change individual PCM samples by about one least significant bit.
var exactMath = os.Getenv("AOIKANE_EXACT") != ""

func openEngine(dir string) (engine.Synthesizer, func(), error) {
	onnx.SetFastMath(!exactMath)
	var e *portable.Engine
	var err error
	if dir == "embedded" {
		e, err = portable.OpenEmbedded()
	} else {
		e, err = portable.Open(portable.Options{AssetsDir: dir})
	}
	if err != nil {
		return nil, nil, err
	}
	return e, func() { e.Close() }, nil
}

const backendName = "portable"
const backendUsage = ""

func engineFlags(f *flag.FlagSet) *string {
	dir := os.Getenv("AOIKANE_PORTABLE")
	if dir == "" {
		dir = "embedded"
	}
	value := f.String("assets", dir, "dictionary/model directory (default: embedded data)")
	f.StringVar(value, "portable", dir, "compatibility alias for --assets")
	f.BoolVar(&exactMath, "exact", exactMath, "bit-exact original kernels (slower; default allows ±1 LSB sample differences)")
	return value
}
func nativeCommand(args []string) error { return fmt.Errorf("%s requires the DLL binary", args[0]) }
