//go:build windows

package tests

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/tommyo123/mtag"
	"github.com/tommyo123/mtag/id3v2"
)

func TestWindowsAtomicRenameWithOpenReader(t *testing.T) {
	v2 := id3v2.NewTag(4)
	v2.SetText(id3v2.FrameTitle, "short")
	path := buildTestFile(t, v2, nil, bytes.Repeat([]byte{0x55}, 2048))

	reader, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.SetLyrics(strings.Repeat("rename verification payload ", 200))
	if err := f.Save(); err != nil {
		t.Fatalf("Save while another reader held the file open = %v", err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if got := g.Lyrics(); !strings.Contains(got, "rename verification payload") {
		t.Fatalf("saved lyrics missing after rename-backed save: %q", got)
	}
}
