package main

import (
	"testing"

	"github.com/a-kaibu/aoikane/internal/engine"
)

func TestBinaryBackendOptions(t *testing.T) {
	f := flags("backend")
	dir := engineFlags(f)
	if backendName == "portable" {
		t.Setenv("AOIKANE_PORTABLE", "")
		f = flags("backend")
		dir = engineFlags(f)
		if *dir != "embedded" || f.Lookup("assets") == nil || f.Lookup("engine-dir") != nil {
			t.Fatal("portable binary must default to embedded data and reject DLL selection")
		}
		if _, err := engine.Open("unused"); err == nil {
			t.Fatal("portable build must exclude the native DLL implementation")
		}
		if err := nativeCommand([]string{"pack"}); err == nil {
			t.Fatal("portable binary must reject DLL packing")
		}
	} else {
		if *dir != "" || f.Lookup("engine-dir") == nil || f.Lookup("assets") != nil || f.Lookup("portable") != nil {
			t.Fatal("DLL binary must expose only native engine selection")
		}
	}
}
