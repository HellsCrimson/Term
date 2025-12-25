package main

import (
    "encoding/binary"
    "fmt"
    "io"
)

type TermrecReader struct {
    r     io.Reader
}

type TermrecHeaderRead struct {
    StartUnixNano int64
    Cols          uint16
    Rows          uint16
    Flags         uint32
}

func NewTermrecReader(r io.Reader) (*TermrecReader, error) {
    // verify magic
    magic := make([]byte, len(termrecMagic))
    if _, err := io.ReadFull(r, magic); err != nil { return nil, err }
    for i := range magic {
        if magic[i] != termrecMagic[i] { return nil, fmt.Errorf("invalid magic") }
    }
    return &TermrecReader{r: r}, nil
}

func (tr *TermrecReader) ReadHeader() (*TermrecHeaderRead, error) {
    var h TermrecHeaderRead
    if err := binary.Read(tr.r, binary.LittleEndian, &h.StartUnixNano); err != nil { return nil, err }
    if err := binary.Read(tr.r, binary.LittleEndian, &h.Cols); err != nil { return nil, err }
    if err := binary.Read(tr.r, binary.LittleEndian, &h.Rows); err != nil { return nil, err }
    if err := binary.Read(tr.r, binary.LittleEndian, &h.Flags); err != nil { return nil, err }
    return &h, nil
}

// Reads next event from the stream. Returns (deltaNs, type, payload, error)
func (tr *TermrecReader) ReadEvent(buf []byte) (uint64, byte, []byte, error) {
    delta, err := readUvarint(tr.r)
    if err != nil { return 0, 0, nil, err }
    tb := make([]byte, 1)
    if _, err := io.ReadFull(tr.r, tb); err != nil { return 0, 0, nil, err }
    ln, err := readUvarint(tr.r)
    if err != nil { return 0, 0, nil, err }
    if int(ln) > cap(buf) {
        buf = make([]byte, ln)
    } else {
        buf = buf[:ln]
    }
    if _, err := io.ReadFull(tr.r, buf); err != nil { return 0, 0, nil, err }
    return delta, tb[0], buf, nil
}

func readUvarint(r io.Reader) (uint64, error) {
    var x uint64
    var s uint
    for i := 0; i < 10; i++ {
        var b [1]byte
        if _, err := r.Read(b[:]); err != nil { return 0, err }
        if b[0] < 0x80 {
            if i == 9 && b[0] > 1 { return 0, fmt.Errorf("varint overflow") }
            return x | uint64(b[0])<<s, nil
        }
        x |= uint64(b[0]&0x7f) << s
        s += 7
    }
    return 0, fmt.Errorf("varint too long")
}

