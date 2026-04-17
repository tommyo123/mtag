package mtag

import (
	"bytes"
	"errors"
	"testing"

	"github.com/tommyo123/mtag/flac"
	"github.com/tommyo123/mtag/id3v2"
)

type memWritable struct{ data []byte }

func (m *memWritable) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(m.data)) {
		return 0, errors.New("EOF")
	}
	n := copy(p, m.data[off:])
	if n < len(p) {
		return n, errors.New("EOF")
	}
	return n, nil
}

func (m *memWritable) WriteAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("EOF")
	}
	end := int(off) + len(p)
	if end > len(m.data) {
		grow := make([]byte, end)
		copy(grow, m.data)
		m.data = grow
	}
	copy(m.data[off:], p)
	return len(p), nil
}

func (m *memWritable) Truncate(size int64) error {
	if size < 0 {
		return errors.New("EOF")
	}
	switch {
	case int(size) < len(m.data):
		m.data = m.data[:size]
	case int(size) > len(m.data):
		grow := make([]byte, size)
		copy(grow, m.data)
		m.data = grow
	}
	return nil
}

func buildFLACCommentFile(t *testing.T, fields ...flac.Field) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.Write(flac.Magic[:])
	comment := &flac.VorbisComment{
		Vendor: "mtag-test",
		Fields: fields,
	}
	if err := flac.WriteBlock(&buf, flac.Block{
		Type:   flac.BlockVorbisComment,
		IsLast: true,
		Body:   flac.EncodeVorbisComment(comment),
	}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func buildSpeexCommentFile(t *testing.T, fields ...flac.Field) []byte {
	t.Helper()
	comment := &flac.VorbisComment{
		Vendor: "mtag-test",
		Fields: fields,
	}
	var out bytes.Buffer
	w := oggPageWriter{dst: &out}
	if err := emitOGGMetaPages(&w, []byte("Speex   synthetic-id-header"), 0, true); err != nil {
		t.Fatal(err)
	}
	if err := emitOGGMetaPages(&w, flac.EncodeVorbisComment(comment), 0, false); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestID3v2AlbumArtistAlias(t *testing.T) {
	tag := id3v2.NewTag(4)
	tag.SetText(id3v2.FrameTitle, "Song")
	tag.Set(&id3v2.UserTextFrame{
		Description: "ALBUM ARTIST",
		Values:      []string{"Alias Band"},
	})
	raw, err := tag.Encode(0)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, []byte("audio")...)

	f, err := OpenBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := f.AlbumArtist(); got != "Alias Band" {
		t.Fatalf("AlbumArtist = %q, want Alias Band", got)
	}
}

func TestVorbisAlbumArtistAliasAndNormalise(t *testing.T) {
	src := &memWritable{data: buildFLACCommentFile(t, flac.Field{Name: "ALBUM ARTIST", Value: "Alias Band"})}
	f, err := OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	if got := f.AlbumArtist(); got != "Alias Band" {
		t.Fatalf("AlbumArtist = %q, want Alias Band", got)
	}
	f.SetAlbumArtist("Canonical Band")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := OpenBytes(src.data)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if got := g.AlbumArtist(); got != "Canonical Band" {
		t.Fatalf("AlbumArtist after save = %q, want Canonical Band", got)
	}
	if got := g.flac.comment.Get("ALBUM ARTIST"); got != "" {
		t.Fatalf("legacy alias survived normalisation: %q", got)
	}
	if got := g.flac.comment.Get("ALBUMARTIST"); got != "Canonical Band" {
		t.Fatalf("canonical field = %q", got)
	}
}

func TestVorbisDiscNumberPair(t *testing.T) {
	raw := buildFLACCommentFile(t, flac.Field{Name: "DISCNUMBER", Value: "1/3"})
	f, err := OpenBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := f.Disc(); got != 1 {
		t.Fatalf("Disc = %d, want 1", got)
	}
	if got := f.DiscTotal(); got != 3 {
		t.Fatalf("DiscTotal = %d, want 3", got)
	}
}

func TestOGGSpeexReadWrite(t *testing.T) {
	src := &memWritable{data: buildSpeexCommentFile(t,
		flac.Field{Name: "TITLE", Value: "Before"},
		flac.Field{Name: "ARTIST", Value: "Artist"},
	)}
	f, err := OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	if f.Container() != ContainerOGG {
		t.Fatalf("container = %v, want OGG", f.Container())
	}
	if got := f.Title(); got != "Before" {
		t.Fatalf("Title = %q, want Before", got)
	}
	f.SetTitle("After")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := OpenBytes(src.data)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if got := g.Title(); got != "After" {
		t.Fatalf("Title after save = %q, want After", got)
	}
	if got := g.Artist(); got != "Artist" {
		t.Fatalf("Artist after save = %q, want Artist", got)
	}
}
