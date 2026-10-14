//go:build (windows || darwin || linux) && (amd64 || arm64)

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/portable"
)

// openEngine uses the reconstructed frontend and models. Embedded data is
// the default; --assets selects an external directory.
func openEngine(dir string) (engine.Synthesizer, func(), error) {
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
const backendUsage = "\nPure-Go backend: embedded dictionaries/models by default, no original DLL.\nUse --assets DIR to load external dictionaries/models.\n"

func engineFlags(f *flag.FlagSet) *string {
	dir := os.Getenv("AOIKANE_PORTABLE")
	if dir == "" {
		dir = "embedded"
	}
	value := f.String("assets", dir, "dictionary/model directory (default: embedded data)")
	f.StringVar(value, "portable", dir, "compatibility alias for --assets")
	return value
}
func nativeCommand(args []string) error { return fmt.Errorf("%s requires the DLL binary", args[0]) }
