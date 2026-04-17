package ape

import (
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

type footerOnlyReader struct {
	end    int64
	footer [FooterSize]byte
}

func (r footerOnlyReader) ReadAt(p []byte, off int64) (int, error) {
	footerAt := r.end - FooterSize
	if off < footerAt || off >= r.end {
		return 0, io.EOF
	}
	n := copy(p, r.footer[off-footerAt:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func TestRegionRejectsOversizeTag(t *testing.T) {
	end := int64(MaxTagRegionSize + FooterSize + 1)
	var footer [FooterSize]byte
	copy(footer[:8], Magic[:])
	binary.LittleEndian.PutUint32(footer[8:12], CurrentVersion)
	binary.LittleEndian.PutUint32(footer[12:16], uint32(MaxTagRegionSize+1))
	binary.LittleEndian.PutUint32(footer[20:24], flagContainsFooter)

	_, _, err := Region(footerOnlyReader{end: end, footer: footer}, end)
	if !errors.Is(err, ErrTagTooLarge) {
		t.Fatalf("Region() error = %v, want %v", err, ErrTagTooLarge)
	}
}
