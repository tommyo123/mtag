package tests

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/tommyo123/mtag"
)

func buildCAFInfoFixture(entries [][2]string) []byte {
	var buf bytes.Buffer
	buf.WriteString("caff")
	buf.Write([]byte{0, 1})
	buf.Write([]byte{0, 0})

	var payload bytes.Buffer
	var count [4]byte
	binary.BigEndian.PutUint32(count[:], uint32(len(entries)))
	payload.Write(count[:])
	for _, kv := range entries {
		payload.WriteString(kv[0])
		payload.WriteByte(0)
		payload.WriteString(kv[1])
		payload.WriteByte(0)
	}

	buf.WriteString("info")
	var sizeBuf [8]byte
	binary.BigEndian.PutUint64(sizeBuf[:], uint64(payload.Len()))
	buf.Write(sizeBuf[:])
	buf.Write(payload.Bytes())
	return buf.Bytes()
}

func TestCAFSaveWritableSourceFullRewrite(t *testing.T) {
	src := &writableBuffer{data: buildCAFInfoFixture([][2]string{
		{"title", "initial"},
		{"artist", "artist"},
	})}
	f, err := mtag.OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	f.SetTitle("updated")
	f.SetAlbum("added-album")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	g, err := mtag.OpenBytes(src.data)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.Container() != mtag.ContainerCAF {
		t.Fatalf("container = %v, want CAF", g.Container())
	}
	if got := g.Title(); got != "updated" {
		t.Fatalf("Title() = %q", got)
	}
	if got := g.Album(); got != "added-album" {
		t.Fatalf("Album() = %q", got)
	}
}

func TestCAFRemoveTagDropsInfoChunk(t *testing.T) {
	src := &writableBuffer{data: buildCAFInfoFixture([][2]string{
		{"title", "initial"},
		{"artist", "artist"},
	})}
	f, err := mtag.OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.RemoveTag(mtag.TagCAF); err != nil {
		t.Fatal(err)
	}
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	g, err := mtag.OpenBytes(src.data)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.Container() != mtag.ContainerCAF {
		t.Fatalf("container = %v, want CAF", g.Container())
	}
	if g.Tag(mtag.TagCAF) != nil {
		t.Fatal("TagCAF survived RemoveTag roundtrip")
	}
	if g.Title() != "" || g.Artist() != "" {
		t.Fatalf("CAF fields survived RemoveTag: title=%q artist=%q", g.Title(), g.Artist())
	}
}
