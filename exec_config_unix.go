//go:build !windows

package main

import "os/exec"

// setCmdNoWindow is a no-op on non-Windows platforms.
func setCmdNoWindow(cmd *exec.Cmd) {}

