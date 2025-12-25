package main

import (
    "encoding/binary"
    "io"
    "time"
)

var termrecMagic = []byte{'T','E','R','M','R','E','C',1}

// TermrecWriter writes a binary terminal recording stream to w
type TermrecWriter struct {
    w      io.Writer
    start  time.Time
    lastTs time.Time
}

type TermrecHeader struct {
    StartUnixNano int64
    Cols          uint16
    Rows          uint16
    Flags         uint32 // bit0: capture input
}

func NewTermrecWriter(w io.Writer, cols, rows uint16, captureInput bool) (*TermrecWriter, error) {
    // magic
    if _, err := w.Write(termrecMagic); err != nil {
        return nil, err
    }
    // header
    hdr := TermrecHeader{
        StartUnixNano: time.Now().UnixNano(),
        Cols:          cols,
        Rows:          rows,
        Flags:         0,
    }
    if captureInput { hdr.Flags |= 1 }
    // binary little endian
    if err := binary.Write(w, binary.LittleEndian, hdr.StartUnixNano); err != nil { return nil, err }
    if err := binary.Write(w, binary.LittleEndian, hdr.Cols); err != nil { return nil, err }
    if err := binary.Write(w, binary.LittleEndian, hdr.Rows); err != nil { return nil, err }
    if err := binary.Write(w, binary.LittleEndian, hdr.Flags); err != nil { return nil, err }
    now := time.Now()
    return &TermrecWriter{w: w, start: now, lastTs: now}, nil
}

// Event format: varint(delta_ns), 1 byte type ('O','I','R'), varint len, payload

func (tw *TermrecWriter) writeEvent(t byte, payload []byte) error {
    now := time.Now()
    delta := now.Sub(tw.lastTs)
    tw.lastTs = now
    if err := writeUvarint(tw.w, uint64(delta.Nanoseconds())); err != nil { return err }
    if _, err := tw.w.Write([]byte{t}); err != nil { return err }
    if err := writeUvarint(tw.w, uint64(len(payload))); err != nil { return err }
    if len(payload) > 0 {
        if _, err := tw.w.Write(payload); err != nil { return err }
    }
    return nil
}

func (tw *TermrecWriter) WriteOutput(p []byte) error { return tw.writeEvent('O', p) }
func (tw *TermrecWriter) WriteInput(p []byte) error  { return tw.writeEvent('I', p) }
func (tw *TermrecWriter) WriteResize(cols, rows uint16) error {
    var buf [4]byte
    binary.LittleEndian.PutUint16(buf[:2], cols)
    binary.LittleEndian.PutUint16(buf[2:], rows)
    return tw.writeEvent('R', buf[:])
}

func writeUvarint(w io.Writer, x uint64) error {
    var buf [10]byte
    n := binary.PutUvarint(buf[:], x)
    _, err := w.Write(buf[:n])
    return err
}
