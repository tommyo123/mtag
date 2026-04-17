package tests

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tommyo123/mtag"
)

func TestWithMaxFileSize_OpenBytes(t *testing.T) {
	_, err := mtag.OpenBytes(make([]byte, 16), mtag.WithMaxFileSize(8))
	if !errors.Is(err, mtag.ErrFileTooLarge) {
		t.Fatalf("OpenBytes error = %v, want ErrFileTooLarge", err)
	}
}

func TestWithMaxFileSize_OpenSource(t *testing.T) {
	r := bytes.NewReader(make([]byte, 16))
	_, err := mtag.OpenSource(r, 16, mtag.WithMaxFileSize(8))
	if !errors.Is(err, mtag.ErrFileTooLarge) {
		t.Fatalf("OpenSource error = %v, want ErrFileTooLarge", err)
	}
}

func TestWithMaxFileSize_OpenWritableSource(t *testing.T) {
	src := &writableBuffer{data: make([]byte, 16)}
	_, err := mtag.OpenWritableSource(src, int64(len(src.data)), mtag.WithMaxFileSize(8))
	if !errors.Is(err, mtag.ErrFileTooLarge) {
		t.Fatalf("OpenWritableSource error = %v, want ErrFileTooLarge", err)
	}
}

func TestWithMaxFileSize_OpenPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.bin")
	if err := os.WriteFile(path, make([]byte, 16), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := mtag.Open(path, mtag.WithMaxFileSize(8))
	if !errors.Is(err, mtag.ErrFileTooLarge) {
		t.Fatalf("Open error = %v, want ErrFileTooLarge", err)
	}
}
