//go:build windows

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// conptyIO wraps the write end (to ConPTY input) and read end (from ConPTY output)
type conptyIO struct {
	in  *os.File // write to child
	out *os.File // read from child
}

func (c *conptyIO) Read(p []byte) (int, error)  { return c.out.Read(p) }
func (c *conptyIO) Write(p []byte) (int, error) { return c.in.Write(p) }
func (c *conptyIO) Close() error {
	var errs []string
	if c.in != nil {
		if err := c.in.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if c.out != nil {
		if err := c.out.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// Win32 interop
var (
	modkernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	procCreatePseudoConsole           = modkernel32.NewProc("CreatePseudoConsole")
	procClosePseudoConsole            = modkernel32.NewProc("ClosePseudoConsole")
	procResizePseudoConsole           = modkernel32.NewProc("ResizePseudoConsole")
	procInitializeProcThreadAttrList  = modkernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute     = modkernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttributeList = modkernel32.NewProc("DeleteProcThreadAttributeList")
	procCreateProcessW                = modkernel32.NewProc("CreateProcessW")
)

const (
	EXTENDED_STARTUPINFO_PRESENT = 0x00080000
	CREATE_UNICODE_ENVIRONMENT   = 0x00000400
	STARTF_USESTDHANDLES         = 0x00000100
	// Attribute constant for Pseudo Console
	PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE = 0x00020016
)

type coord struct {
	X int16
	Y int16
}

func packCoord(c coord) uintptr {
	v := (uint32(uint16(c.Y)) << 16) | uint32(uint16(c.X))
	return uintptr(v)
}

// opaque type for attribute list pointer
type procThreadAttributeList struct{}

// STARTUPINFOEX contains a STARTUPINFO followed by lpAttributeList
type startupInfoEx struct {
	windows.StartupInfo
	lpAttributeList *procThreadAttributeList
}

// processInfo wraps PROCESS_INFORMATION
type processInfo struct {
	Process   windows.Handle
	Thread    windows.Handle
	ProcessId uint32
	ThreadId  uint32
}

func createPseudoConsole(cols, rows uint16, inRead, outWrite windows.Handle) (windows.Handle, error) {
	var size = coord{X: int16(cols), Y: int16(rows)}
	var hpc windows.Handle
	r1, _, e1 := procCreatePseudoConsole.Call(
		packCoord(size),
		uintptr(inRead),
		uintptr(outWrite),
		uintptr(0),
		uintptr(unsafe.Pointer(&hpc)),
	)
	if r1 != 0 { // non-zero HRESULT indicates failure
		if e1 != windows.ERROR_SUCCESS && e1 != nil {
			return 0, e1
		}
		return 0, fmt.Errorf("CreatePseudoConsole failed: HRESULT=0x%x", r1)
	}
	return hpc, nil
}

func closePseudoConsole(hpc windows.Handle) {
	if hpc != 0 {
		procClosePseudoConsole.Call(uintptr(hpc))
	}
}

func resizePseudoConsole(hpc windows.Handle, cols, rows uint16) error {
	var size = coord{X: int16(cols), Y: int16(rows)}
	r1, _, e1 := procResizePseudoConsole.Call(
		uintptr(hpc),
		packCoord(size),
	)
	if r1 != 0 {
		if e1 != windows.ERROR_SUCCESS && e1 != nil {
			return e1
		}
		return fmt.Errorf("ResizePseudoConsole failed: HRESULT=0x%x", r1)
	}
	return nil
}

// buildCommandLine joins command + args for CreateProcessW
func buildCommandLine(cmd *exec.Cmd) *uint16 {
	parts := make([]string, 0, 1+len(cmd.Args))
	if cmd.Path != "" {
		parts = append(parts, quoteIfNeeded(cmd.Path))
	}
	for _, a := range cmd.Args[1:] { // first arg is usually Path
		parts = append(parts, quoteIfNeeded(a))
	}
	s := strings.Join(parts, " ")
	u16, _ := windows.UTF16PtrFromString(s)
	return u16
}

func quoteIfNeeded(s string) string {
	if s == "" {
		return ""
	}
	if strings.IndexByte(s, ' ') >= 0 || strings.IndexByte(s, '\t') >= 0 {
		// naive quoting; good enough for typical shells
		return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
	}
	return s
}

// buildEnvBlock builds a Windows environment block (UTF-16, double NUL terminated)
func buildEnvBlock(env []string) *uint16 {
	// Build UTF-16 with embedded NULs between entries and final double NUL
	var utf []uint16
	for _, e := range env {
		// Convert each entry and strip trailing NUL
		u, _ := windows.UTF16FromString(e)
		if len(u) > 0 && u[len(u)-1] == 0 {
			u = u[:len(u)-1]
		}
		utf = append(utf, u...)
		utf = append(utf, 0)
	}
	utf = append(utf, 0)
	return &utf[0]
}

// startPTY starts a ConPTY-backed command on Windows using native APIs.
// It returns a ReadWriteCloser connected to the PTY, a resize function,
// a wait function (exit code), a kill function, a cleanup function, and error.
func startPTY(cmd *exec.Cmd, cols, rows uint16) (io.ReadWriteCloser, func(uint16, uint16) error, func() (int, error), func() error, func(), error) {
	// Create pipes: host writes to inWrite, reads from outRead
	var inRead, inWrite windows.Handle
	var outRead, outWrite windows.Handle
	if err := windows.CreatePipe(&inRead, &inWrite, nil, 0); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := windows.CreatePipe(&outRead, &outWrite, nil, 0); err != nil {
		windows.CloseHandle(inRead)
		windows.CloseHandle(inWrite)
		return nil, nil, nil, nil, nil, err
	}

	// Create ConPTY using pipe ends
	hpc, err := createPseudoConsole(defaultCols(cols), defaultRows(rows), inRead, outWrite)
	// We can close ends we gave to ConPTY once created
	windows.CloseHandle(inRead)
	windows.CloseHandle(outWrite)
	if err != nil {
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, nil, nil, nil, nil, err
	}

	// Attribute list for PseudoConsole
	var attrListSize uintptr
	r1, _, _ := procInitializeProcThreadAttrList.Call(0, 1, 0, uintptr(unsafe.Pointer(&attrListSize)))
	// should fail with ERROR_INSUFFICIENT_BUFFER; attrListSize now set
	_ = r1
	attrList := make([]byte, attrListSize)
	if r1, _, e1 := procInitializeProcThreadAttrList.Call(
		uintptr(unsafe.Pointer(&attrList[0])), 1, 0, uintptr(unsafe.Pointer(&attrListSize))); r1 == 0 {
		closePseudoConsole(hpc)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		if e1 != nil {
			return nil, nil, nil, nil, nil, e1
		}
		return nil, nil, nil, nil, nil, errors.New("InitializeProcThreadAttributeList failed")
	}
	// Update attribute with PseudoConsole handle
	if r1, _, e1 := procUpdateProcThreadAttribute.Call(
		uintptr(unsafe.Pointer(&attrList[0])), 0,
		uintptr(PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE), uintptr(hpc), unsafe.Sizeof(hpc), 0, 0); r1 == 0 {
		procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&attrList[0])))
		closePseudoConsole(hpc)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		if e1 != nil {
			return nil, nil, nil, nil, nil, e1
		}
		return nil, nil, nil, nil, nil, errors.New("UpdateProcThreadAttribute failed")
	}

	// Prepare STARTUPINFOEX
	var siEx startupInfoEx
	siEx.Cb = uint32(unsafe.Sizeof(siEx))
	siEx.lpAttributeList = (*procThreadAttributeList)(unsafe.Pointer(&attrList[0]))

	// Convert working dir
	var dir *uint16
	if cmd.Dir != "" {
		dir, _ = windows.UTF16PtrFromString(cmd.Dir)
	}
	// Build command line
	cl := buildCommandLine(cmd)
	// Build environment block if provided
	var envBlock *uint16
	if len(cmd.Env) > 0 {
		envBlock = buildEnvBlock(cmd.Env)
	}

	// Create process
	var pi processInfo
	flags := uint32(EXTENDED_STARTUPINFO_PRESENT | CREATE_UNICODE_ENVIRONMENT)
	inherit := uint32(0)
	r1, _, e1 := procCreateProcessW.Call(
		0,
		uintptr(unsafe.Pointer(cl)),
		0, 0,
		uintptr(inherit),
		uintptr(flags),
		uintptr(unsafe.Pointer(envBlock)),
		uintptr(unsafe.Pointer(dir)),
		uintptr(unsafe.Pointer(&siEx.StartupInfo)),
		uintptr(unsafe.Pointer(&pi)),
	)
	// attribute list no longer needed
	procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&attrList[0])))
	if r1 == 0 {
		closePseudoConsole(hpc)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		if e1 != nil {
			return nil, nil, nil, nil, nil, e1
		}
		return nil, nil, nil, nil, nil, errors.New("CreateProcessW failed")
	}
	// We don't need the thread handle; close it
	if pi.Thread != 0 {
		windows.CloseHandle(pi.Thread)
	}

	// Wrap our ends as files
	inFile := os.NewFile(uintptr(inWrite), "conpty-in")
	outFile := os.NewFile(uintptr(outRead), "conpty-out")
	rw := &conptyIO{in: inFile, out: outFile}

	// Define wait/kill/close
	waitFn := func() (int, error) {
		s, err := windows.WaitForSingleObject(pi.Process, windows.INFINITE)
		if err != nil {
			return 0, err
		}
		if s != windows.WAIT_OBJECT_0 {
			return 0, fmt.Errorf("unexpected wait status: %d", s)
		}
		var code uint32
		if err := windows.GetExitCodeProcess(pi.Process, &code); err != nil {
			return 0, err
		}
		return int(code), nil
	}
	killFn := func() error {
		// Best-effort terminate
		_ = windows.TerminateProcess(pi.Process, 1)
		return nil
	}
	closeFn := func() {
		// Close pseudo console and process handle; rw will be closed elsewhere
		closePseudoConsole(hpc)
		if pi.Process != 0 {
			windows.CloseHandle(pi.Process)
		}
	}

	resizeFn := func(c, r uint16) error { return resizePseudoConsole(hpc, c, r) }
	return rw, resizeFn, waitFn, killFn, closeFn, nil
}

func defaultCols(c uint16) uint16 {
	if c == 0 {
		return 80
	}
	return c
}
func defaultRows(r uint16) uint16 {
	if r == 0 {
		return 25
	}
	return r
}
