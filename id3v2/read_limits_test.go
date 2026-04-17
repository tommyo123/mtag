package id3v2

import (
	"bytes"
	"errors"
	"testing"
)

func TestReadBoundedRejectsOversizeDeclaredBody(t *testing.T) {
	var hdr [HeaderSize]byte
	copy(hdr[:3], []byte("ID3"))
	hdr[3] = 4
	if err := EncodeSynchsafe(hdr[6:10], 1<<21); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadBounded(bytes.NewReader(hdr[:]), 0, int64(len(hdr))); !errors.Is(err, ErrTagTooLarge) {
		t.Fatalf("ReadBounded() error = %v, want %v", err, ErrTagTooLarge)
	}
}

func TestReadRejectsHugeDeclaredBody(t *testing.T) {
	var hdr [HeaderSize]byte
	copy(hdr[:3], []byte("ID3"))
	hdr[3] = 4
	if err := EncodeSynchsafe(hdr[6:10], MaxTagBodySize+1); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(bytes.NewReader(hdr[:]), 0); !errors.Is(err, ErrTagTooLarge) {
		t.Fatalf("Read() error = %v, want %v", err, ErrTagTooLarge)
	}
}
