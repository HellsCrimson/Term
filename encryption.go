package main

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "io"

    "golang.org/x/crypto/argon2"
)

type Argon2Params struct {
    Time    uint32
    Memory  uint32
    Threads uint8
    KeyLen  uint32
}

var defaultArgon2 = Argon2Params{Time: 3, Memory: 64 * 1024, Threads: 2, KeyLen: 32}

func deriveKeyArgon2(passphrase, salt []byte, p Argon2Params) []byte {
    return argon2.IDKey(passphrase, salt, p.Time, p.Memory, p.Threads, p.KeyLen)
}

func randBytes(n int) ([]byte, error) {
    b := make([]byte, n)
    if _, err := io.ReadFull(rand.Reader, b); err != nil {
        return nil, err
    }
    return b, nil
}

// EncryptKeyGCM encrypts a key using AES-GCM with the provided master key, returning (ciphertext, nonce)
func EncryptKeyGCM(masterKey, plain []byte) ([]byte, []byte, error) {
    block, err := aes.NewCipher(masterKey)
    if err != nil {
        return nil, nil, err
    }
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return nil, nil, err
    }
    nonce, err := randBytes(aead.NonceSize())
    if err != nil {
        return nil, nil, err
    }
    ct := aead.Seal(nil, nonce, plain, nil)
    return ct, nonce, nil
}

// ChunkedAEADWriter wraps an io.Writer and writes data as length+nonce+ciphertext chunks using AES-GCM
type ChunkedAEADWriter struct {
    w     io.Writer
    aead  cipher.AEAD
    nonce []byte
    ctr   uint64
}

func NewChunkedAEADWriter(w io.Writer, key []byte) (*ChunkedAEADWriter, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    // 12-byte base nonce (first 4 bytes random prefix, last 8 bytes used as counter)
    nonce := make([]byte, aead.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce[:4]); err != nil {
        return nil, err
    }
    return &ChunkedAEADWriter{w: w, aead: aead, nonce: nonce, ctr: 0}, nil
}

func (cw *ChunkedAEADWriter) nextNonce() []byte {
    n := make([]byte, len(cw.nonce))
    copy(n, cw.nonce)
    // encode ctr as big endian into last 8 bytes
    c := cw.ctr
    for i := 0; i < 8; i++ {
        n[len(n)-1-i] = byte(c)
        c >>= 8
    }
    cw.ctr++
    return n
}

func (cw *ChunkedAEADWriter) Write(p []byte) (int, error) {
    // chunk up to 64KB
    const maxChunk = 64 * 1024
    written := 0
    for len(p) > 0 {
        chunk := p
        if len(chunk) > maxChunk {
            chunk = p[:maxChunk]
        }
        nonce := cw.nextNonce()
        ct := cw.aead.Seal(nil, nonce, chunk, nil)
        // write: 4-byte big-endian length of ciphertext, then nonce, then ciphertext
        var hdr [4]byte
        l := len(ct)
        hdr[0] = byte(l >> 24)
        hdr[1] = byte(l >> 16)
        hdr[2] = byte(l >> 8)
        hdr[3] = byte(l)
        if _, err := cw.w.Write(hdr[:]); err != nil {
            return written, err
        }
        if _, err := cw.w.Write(nonce); err != nil {
            return written, err
        }
        if _, err := cw.w.Write(ct); err != nil {
            return written, err
        }
        written += len(chunk)
        p = p[len(chunk):]
    }
    return written, nil
}

func b64(data []byte) string { return base64.StdEncoding.EncodeToString(data) }

func decodeB64(s string) ([]byte, error) {
    b, err := base64.StdEncoding.DecodeString(s)
    if err != nil {
        return nil, fmt.Errorf("invalid base64: %w", err)
    }
    return b, nil
}

// ChunkedAEADReader is the counterpart of ChunkedAEADWriter.
// It expects a stream of chunks: [u32 ct_len][nonce][ciphertext]
// and returns the concatenated plaintext.
type ChunkedAEADReader struct {
    r     io.Reader
    aead  cipher.AEAD
    buf   []byte
    off   int
}

func NewChunkedAEADReader(r io.Reader, key []byte) (*ChunkedAEADReader, error) {
    block, err := aes.NewCipher(key)
    if err != nil { return nil, err }
    aead, err := cipher.NewGCM(block)
    if err != nil { return nil, err }
    return &ChunkedAEADReader{r: r, aead: aead}, nil
}

func readFull(r io.Reader, buf []byte) error {
    _, err := io.ReadFull(r, buf)
    return err
}

func (cr *ChunkedAEADReader) Read(p []byte) (int, error) {
    // Serve from buffer if available
    if cr.off < len(cr.buf) {
        n := copy(p, cr.buf[cr.off:])
        cr.off += n
        return n, nil
    }
    // Load next chunk
    var hdr [4]byte
    if err := readFull(cr.r, hdr[:]); err != nil {
        return 0, err
    }
    l := int(hdr[0])<<24 | int(hdr[1])<<16 | int(hdr[2])<<8 | int(hdr[3])
    nonce := make([]byte, cr.aead.NonceSize())
    if err := readFull(cr.r, nonce); err != nil { return 0, err }
    ct := make([]byte, l)
    if err := readFull(cr.r, ct); err != nil { return 0, err }
    pt, err := cr.aead.Open(nil, nonce, ct, nil)
    if err != nil { return 0, err }
    cr.buf = pt
    cr.off = 0
    // Serve from new buffer
    n := copy(p, cr.buf)
    cr.off = n
    return n, nil
}
