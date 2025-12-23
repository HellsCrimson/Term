//go:build windows

package main

import (
    "unsafe"

    "golang.org/x/sys/windows"
)

// isElevated returns true if the current process has an elevated token
func isElevated() bool {
    var token windows.Token
    err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
    if err != nil {
        return false
    }
    defer token.Close()

    type tokenElevation struct {
        TokenIsElevated uint32
    }
    var te tokenElevation
    var outLen uint32
    err = windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(unsafe.Pointer(&te)), uint32(unsafe.Sizeof(te)), &outLen)
    if err != nil {
        return false
    }
    return te.TokenIsElevated != 0
}
