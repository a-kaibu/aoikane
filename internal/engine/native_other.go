package engine

import "errors"

func Open(root string) (Synthesizer, error) {
	return nil, errors.New("native DLL synthesis is unavailable in this build; use aoikane-dll on Windows x64 or --url http://WINDOWS_HOST:8080")
}
