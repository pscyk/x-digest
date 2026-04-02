//go:build debug

package main

import "fmt"

// assert panics with msg when condition is false. Active only in debug builds.
// Build with: go build -tags debug
func assert(condition bool, msg string) {
	if !condition {
		panic(fmt.Sprintf("assertion failed: %s", msg))
	}
}
