//go:build js && wasm

package ai

import "syscall/js"

// nowMs returns the current time in milliseconds using the browser's
// performance.now() — high-resolution monotonic clock available in WASM.
func nowMs() float64 {
	return js.Global().Get("performance").Call("now").Float()
}