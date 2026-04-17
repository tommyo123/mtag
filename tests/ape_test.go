package tests

import (
	"bytes"
	"testing"

	"github.com/tommyo123/mtag/ape"
)

func TestAPE_RoundTripBasic(t *testing.T) {
	in := ape.New()
	in.Set(ape.FieldTitle, "Heroes")
	in.Set(ape.FieldArtist, "David Bowie")
	in.Set(ape.FieldAlbum, "Heroes")
	in.Set(ape.FieldYear, "1977")
	in.Set(ape.FieldTrack, "3/10")
	in.SetBinary(ape.FieldCoverArtFront, []byte("jpg\x00\xff\xd8\xff\xe0"))

	raw, err := in.Encode()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ape.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != ape.CurrentVersion {
		t.Errorf("version = %d, want %d", got.Version, ape.CurrentVersion)
	}
	if got.Get(ape.FieldTitle) != "Heroes" {
		t.Errorf("title = %q", got.Get(ape.FieldTitle))
	}
	if got.Get(ape.FieldArtist) != "David Bowie" {
		t.Errorf("artist = %q", got.Get(ape.FieldArtist))
	}
	if got.Get(ape.FieldTrack) != "3/10" {
		t.Errorf("track = %q", got.Get(ape.FieldTrack))
	}
	cover := got.Find(ape.FieldCoverArtFront)
	if cover == nil || !cover.IsBinary() {
		t.Fatalf("cover art missing or wrong type: %+v", cover)
	}
	if !bytes.Equal(cover.Value, []byte("jpg\x00\xff\xd8\xff\xe0")) {
		t.Errorf("cover bytes mismatch: %x", cover.Value)
	}
}

func TestAPE_Remove(t *testing.T) {
	tag := ape.New()
	tag.Set(ape.FieldTitle, "A")
	tag.Set(ape.FieldArtist, "B")
	if n := tag.Remove(ape.FieldTitle); n != 1 {
		t.Errorf("Remove(Title) = %d, want 1", n)
	}
	if tag.Get(ape.FieldTitle) != "" {
		t.Errorf("title not removed")
	}
	if tag.Get(ape.FieldArtist) != "B" {
		t.Errorf("artist lost")
	}
}

type apeBytesReader []byte

func (b apeBytesReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(b)) {
		return 0, errStr("EOF")
	}
	n := copy(p, b[off:])
	if n < len(p) {
		return n, errStr("EOF")
	}
	return n, nil
}

func TestAPE_ReadFromRegion(t *testing.T) {
	tag := ape.New()
	tag.Set(ape.FieldArtist, "X")
	raw, _ := tag.Encode()
	file := append(bytes.Repeat([]byte{0xAA}, 100), raw...)
	got, offset, err := ape.Read(apeBytesReader(file), int64(len(file)))
	if err != nil {
		t.Fatal(err)
	}
	if offset != 100 {
		t.Errorf("offset = %d, want 100", offset)
	}
	if got.Get(ape.FieldArtist) != "X" {
		t.Errorf("artist = %q", got.Get(ape.FieldArtist))
	}
}

func TestAPE_CaseInsensitiveLookup(t *testing.T) {
	// Use the public Set API so we don't need the unexported
	// "UTF-8 text" flag; the resulting field keeps its declared
	// case and the Get lookup is still case-insensitive.
	tag := ape.New()
	tag.Set("TiTlE", "casey")
	if tag.Get("title") != "casey" || tag.Get("TITLE") != "casey" {
		t.Errorf("case-insensitive lookup failed")
	}
}

// FuzzAPEDecode makes sure malformed APE regions never panic.
func FuzzAPEDecode(f *testing.F) {
	tag := ape.New()
	tag.Set(ape.FieldTitle, "seed")
	raw, _ := tag.Encode()
	f.Add(raw)
	f.Add(make([]byte, ape.FooterSize))
	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = ape.Decode(b)
	})
}

// FuzzAPERead exercises the trailing-region scanner.
func FuzzAPERead(f *testing.F) {
	tag := ape.New()
	tag.Set(ape.FieldArtist, "seed")
	raw, _ := tag.Encode()
	prefix := make([]byte, 64)
	f.Add(append(prefix, raw...))
	f.Add(make([]byte, ape.FooterSize))
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) < ape.FooterSize {
			return
		}
		_, _, _ = ape.Read(apeBytesReader(b), int64(len(b)))
	})
}
