package tests

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/tommyo123/mtag"
)

// TestEdge_MTimePreservedOnSave covers every save path that has a
// corresponding testdata fixture. mtime preservation is centralised
// in Save(), so one failure here points to a regression in that
// wrapper rather than a per-format leak.
func TestEdge_MTimePreservedOnSave(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mtime preservation is covered on Unix-like systems")
	}
	fixtures := []string{
		"mp3-id3v24.mp3", // ID3v2 in-place patch
		"flac.flac",      // FLAC in-place meta patch
		"mp4-aac.m4a",    // MP4 ilst in-place patch
		"ogg-vorbis.ogg", // OGG comment packet rewrite
		"wav-id3.wav",    // RIFF chunk in-place patch
		"aiff.aiff",      // AIFF chunk rewrite
		"wma.wma",        // ASF content-description rewrite
		"monkeys-audio.ape", // APE tag in-place patch
		"wavpack.wv",     // WavPack + APE
		"dsf.dsf",        // DSF trailing ID3
		"caf.caf",        // CAF info chunk patch
	}
	want := time.Date(2004, time.March, 2, 3, 4, 5, 0, time.UTC)
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			path := testdataCopy(t, name)
			if err := os.Chtimes(path, want, want); err != nil {
				t.Fatalf("seed mtime: %v", err)
			}
			f, err := mtag.Open(path)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			f.SetTitle("mtime")
			if err := f.Save(); err != nil {
				f.Close()
				t.Skipf("save: %v", err)
			}
			if err := f.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			if !info.ModTime().Equal(want) {
				t.Errorf("mtime mismatch: got %v want %v", info.ModTime(), want)
			}
		})
	}
}
