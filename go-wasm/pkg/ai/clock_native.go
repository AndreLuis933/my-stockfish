//go:build !(js && wasm)

package ai

import "time"

// nowMs returns the current time in milliseconds using Go's time package.
// Used when running natively (go test, CLI tools) — outside the browser.
func nowMs() float64 {
	return float64(time.Now().UnixMilli())
}