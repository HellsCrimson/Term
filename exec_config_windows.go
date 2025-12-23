//go:build windows

package main

import (
    "os/exec"
    "syscall"
)

// setCmdNoWindow configures the command to not create/show a console window.
func setCmdNoWindow(cmd *exec.Cmd) {
    // CreationFlags 0x08000000 is CREATE_NO_WINDOW
    // HideWindow hints the OS to hide any console window.
    cmd.SysProcAttr = &syscall.SysProcAttr{
        HideWindow:    true,
        CreationFlags: 0x08000000,
    }
}

