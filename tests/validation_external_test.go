package tests

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tommyo123/mtag"
)

func TestOpenRejectsUnsupportedBytes(t *testing.T) {
	_, err := mtag.OpenBytes([]byte{0x00, 0x01, 0x02, 0x03, 0xA5, 0x5A, 0x10, 0x11})
	if !errors.Is(err, mtag.ErrUnsupportedFormat) {
		t.Fatalf("OpenBytes() error = %v, want %v", err, mtag.ErrUnsupportedFormat)
	}
}

func TestOpenRejectsUnsupportedFileWithoutChangingBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unknown.bin")
	want := []byte("not-an-audio-container\x00\x01\x02\x03")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := mtag.Open(path); !errors.Is(err, mtag.ErrUnsupportedFormat) {
		t.Fatalf("Open() error = %v, want %v", err, mtag.ErrUnsupportedFormat)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("file bytes changed on failed open: got %q want %q", got, want)
	}
}

func TestEmptyFileCanBeTagged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.mp3")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatalf("Open(empty) error = %v", err)
	}
	f.SetTitle("Empty file title")
	if err := f.Save(); err != nil {
		f.Close()
		t.Fatalf("Save() error = %v", err)
	}
	f.Close()

	g, err := mtag.Open(path, mtag.WithReadOnly())
	if err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	defer g.Close()
	if got := g.Title(); got != "Empty file title" {
		t.Fatalf("Title() = %q, want %q", got, "Empty file title")
	}
}

func TestZeroValuePublicMutatorsDoNotPanic(t *testing.T) {
	calls := []struct {
		name string
		fn   func(*mtag.File) error
	}{
		{"SetTitle", func(f *mtag.File) error { f.SetTitle("title"); return nil }},
		{"SetAlbumArtist", func(f *mtag.File) error { f.SetAlbumArtist("artist"); return nil }},
		{"SetYear", func(f *mtag.File) error { f.SetYear(2026); return nil }},
		{"SetTrack", func(f *mtag.File) error { f.SetTrack(2, 10); return nil }},
		{"SetDisc", func(f *mtag.File) error { f.SetDisc(1, 2); return nil }},
		{"SetGenre", func(f *mtag.File) error { f.SetGenre("Rock"); return nil }},
		{"SetComment", func(f *mtag.File) error { f.SetComment("comment"); return nil }},
		{"SetLyrics", func(f *mtag.File) error { f.SetLyrics("lyrics"); return nil }},
		{"SetCompilation", func(f *mtag.File) error { f.SetCompilation(true); return nil }},
		{"SetMusicBrainzID", func(f *mtag.File) error {
			f.SetMusicBrainzID(mtag.MusicBrainzRecordingID, "id")
			return nil
		}},
		{"SetCustomValues", func(f *mtag.File) error { f.SetCustomValues("example", "value"); return nil }},
		{"SetReplayGainTrack", func(f *mtag.File) error {
			f.SetReplayGainTrack(mtag.ReplayGain{Gain: -6.5, Peak: 0.95})
			return nil
		}},
		{"AddImage", func(f *mtag.File) error {
			f.AddImage(mtag.Picture{MIME: "image/jpeg", Type: mtag.PictureCoverFront, Data: []byte{0xFF, 0xD8, 0xFF}})
			return nil
		}},
		{"SetCoverArt", func(f *mtag.File) error { f.SetCoverArt("image/jpeg", []byte{0xFF, 0xD8, 0xFF}); return nil }},
		{"RemoveImages", func(f *mtag.File) error { f.RemoveImages(); return nil }},
		{"RemoveField", func(f *mtag.File) error { return f.RemoveField(mtag.FieldTitle) }},
		{"RemoveTag", func(f *mtag.File) error { return f.RemoveTag(mtag.TagID3v2) }},
	}

	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			var f mtag.File
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			err := tc.fn(&f)
			if tc.name == "RemoveTag" {
				if !errors.Is(err, mtag.ErrNoTag) {
					t.Fatalf("RemoveTag() error = %v, want %v", err, mtag.ErrNoTag)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s error = %v", tc.name, err)
			}
		})
	}
}

func TestZeroValueCueSheetDoesNotPanic(t *testing.T) {
	var f mtag.File
	if _, ok := f.CueSheet(); ok {
		t.Fatal("CueSheet() = ok on zero File")
	}
}

func TestErrsReturnsCopy(t *testing.T) {
	var f mtag.File
	f.SetLyrics("lyrics")
	f.SetCustomValues("example", "value")

	errs := f.Errs()
	if len(errs) != 2 {
		t.Fatalf("len(Errs()) = %d, want 2", len(errs))
	}
	errs[0] = nil

	if f.Err() == nil {
		t.Fatal("Err() changed after caller mutated Errs() result")
	}
	if len(f.Errs()) != 2 {
		t.Fatalf("len(Errs()) after mutation = %d, want 2", len(f.Errs()))
	}
}

func TestZeroValueAudioPropertiesDoesNotPanic(t *testing.T) {
	var f mtag.File
	props := f.AudioProperties()
	if props != (mtag.AudioProperties{}) {
		t.Fatalf("AudioProperties() on zero File = %+v, want zero value", props)
	}
}
