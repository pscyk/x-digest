//go:build !debug

package main

// assert is a no-op in release builds.
func assert(_ bool, _ string) {}
