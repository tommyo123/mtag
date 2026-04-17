package tests

import (
	"testing"

	"github.com/tommyo123/mtag/flac"
)

// flacBytesReader is a tiny [io.ReaderAt] over an in-memory buffer,
// used by the FLAC fuzz tests.
type flacBytesReader []byte

func (b flacBytesReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(b)) {
		return 0, errEOFFLAC
	}
	n := copy(p, b[off:])
	if n < len(p) {
		return n, errEOFFLAC
	}
	return n, nil
}

const errEOFFLAC errStr = "EOF"

// FuzzFLACReadBlocks exercises the FLAC block walker with arbitrary
// input. Any panic is a fuzz failure.
func FuzzFLACReadBlocks(f *testing.F) {
	seed := []byte{
		'f', 'L', 'a', 'C',
		0x84, 0x00, 0x00, 0x08,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	f.Add(seed)
	f.Add([]byte("fLaC\x80\x00\x00\x00"))
	f.Fuzz(func(t *testing.T, b []byte) {
		_, _, _ = flac.ReadBlocks(flacBytesReader(b), int64(len(b)))
	})
}

// FuzzFLACDecodeVorbisComment pushes random bytes into the comment
// decoder.
func FuzzFLACDecodeVorbisComment(f *testing.F) {
	vc := &flac.VorbisComment{Vendor: "seed", Fields: []flac.Field{{Name: "TITLE", Value: "x"}}}
	f.Add(flac.EncodeVorbisComment(vc))
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = flac.DecodeVorbisComment(b)
	})
}
