package tests

import (
	"testing"

	"github.com/tommyo123/mtag/mp4"
)

type mp4BytesReader []byte

func (b mp4BytesReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(b)) {
		return 0, errStr("EOF")
	}
	n := copy(p, b[off:])
	if n < len(p) {
		return n, errStr("EOF")
	}
	return n, nil
}

// FuzzMP4ReadItems feeds arbitrary bytes to the MP4 ilst decoder.
// Malformed files are allowed to return errors, but must never
// panic.
func FuzzMP4ReadItems(f *testing.F) {
	f.Add([]byte{0, 0, 0, 8, 'f', 't', 'y', 'p'})
	f.Add([]byte{0, 0, 0, 0, 'f', 't', 'y', 'p', 'M', '4', 'A', ' '})
	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = mp4.ReadItems(mp4BytesReader(b), int64(len(b)))
	})
}
