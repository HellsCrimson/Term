//go:build !windows

package main

// isElevated always returns false on non-Windows platforms
func isElevated() bool { return false }

