//go:build linux

package tests

import (
	"bytes"
	"errors"
	"syscall"
	"testing"

	"github.com/tommyo123/mtag"
)

func TestEdge_XattrPreservedOnMP3Rewrite(t *testing.T) {
	path := testdataCopy(t, "mp3-id3v24.mp3")
	const name = "user.mtag.test"
	want := []byte("kept")
	if err := syscall.Setxattr(path, name, want, 0); err != nil {
		if errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EPERM) {
			t.Skipf("xattrs unavailable: %v", err)
		}
		t.Fatalf("setxattr: %v", err)
	}

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	f.SetTitle("xattr")
	if err := f.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	size, err := syscall.Getxattr(path, name, nil)
	if err != nil {
		t.Fatalf("getxattr size: %v", err)
	}
	got := make([]byte, size)
	size, err = syscall.Getxattr(path, name, got)
	if err != nil {
		t.Fatalf("getxattr: %v", err)
	}
	got = got[:size]
	if !bytes.Equal(got, want) {
		t.Fatalf("xattr mismatch: got %q want %q", got, want)
	}
}
