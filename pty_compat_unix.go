//go:build !windows

package main

import (
    "io"
    "os/exec"

    ptylib "github.com/creack/pty"
)

// startPTY starts a PTY-backed command on Unix-like systems and returns a
// bidirectional ReadWriteCloser for I/O, a resize function, and nil wait/kill/close
// (we rely on exec.Cmd for wait & kill).
func startPTY(cmd *exec.Cmd, cols, rows uint16) (io.ReadWriteCloser, func(uint16, uint16) error, func() (int, error), func() error, func(), error) {
    f, err := ptylib.Start(cmd)
    if err != nil {
        return nil, nil, nil, nil, nil, err
    }
    resize := func(c, r uint16) error {
        return ptylib.Setsize(f, &ptylib.Winsize{Rows: r, Cols: c})
    }
    if cols > 0 && rows > 0 {
        _ = resize(cols, rows)
    }
    return f, resize, nil, nil, func(){}, nil
}
