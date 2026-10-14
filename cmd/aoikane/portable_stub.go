//go:build !((windows || darwin || linux) && (amd64 || arm64))

package main

import (
	"errors"
	"flag"

	"github.com/a-kaibu/aoikane/internal/engine"
)

func openEngine(string) (engine.Synthesizer, func(), error) {
	return nil, nil, errors.New("the portable engine is not available on this platform")
}

const backendName = "portable"
const backendUsage = "\nPortable synthesis is unavailable on this platform.\n"

func engineFlags(f *flag.FlagSet) *string {
	return f.String("assets", "embedded", "dictionary/model directory")
}
func nativeCommand([]string) error { return errors.New("pack and bundle require the DLL binary") }
