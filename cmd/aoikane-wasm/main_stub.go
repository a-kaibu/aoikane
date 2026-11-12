//go:build !(js && wasm)

// Command aoikane-wasm is the browser build of the portable synthesizer.
// Build it with GOOS=js GOARCH=wasm; native builds only provide this notice.
package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Printf("aoikane-wasm builds for js/wasm (current %s/%s)\n", runtime.GOOS, runtime.GOARCH)
}
