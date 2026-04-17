package mtag

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/tommyo123/mtag/flac"
)

func buildWAVInfoFile(info map[string]string) []byte {
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	buf.Write([]byte{0, 0, 0, 0})
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	buf.Write([]byte{16, 0, 0, 0})
	buf.Write(make([]byte, 16))
	var list bytes.Buffer
	list.WriteString("INFO")
	for k, v := range info {
		list.WriteString(k)
		size := uint32(len(v) + 1)
		var sz [4]byte
		binary.LittleEndian.PutUint32(sz[:], size)
		list.Write(sz[:])
		list.WriteString(v)
		list.WriteByte(0)
		if size%2 == 1 {
			list.WriteByte(0)
		}
	}
	buf.WriteString("LIST")
	var listSize [4]byte
	binary.LittleEndian.PutUint32(listSize[:], uint32(list.Len()))
	buf.Write(listSize[:])
	buf.Write(list.Bytes())
	buf.WriteString("data")
	buf.Write([]byte{0, 0, 0, 0})
	out := buf.Bytes()
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	return out
}

func buildASFFile(t *testing.T, view *asfView) []byte {
	t.Helper()
	child := renderASFContentDescription(view)
	out := make([]byte, 30)
	copy(out[:16], asfHeaderObjectGUID[:])
	binary.LittleEndian.PutUint64(out[16:24], uint64(30+len(child)))
	binary.LittleEndian.PutUint32(out[24:28], 1)
	out[28] = 1
	out[29] = 2
	out = append(out, child...)
	return out
}

func buildRealMediaFile(t *testing.T, view *realMediaView) []byte {
	t.Helper()
	cont, err := renderRealMediaCONT(view)
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 10)
	copy(data[:4], []byte("DATA"))
	binary.BigEndian.PutUint32(data[4:8], uint32(len(data)))
	header := make([]byte, 18)
	copy(header[:4], []byte(".RMF"))
	binary.BigEndian.PutUint32(header[4:8], uint32(len(header)))
	binary.BigEndian.PutUint32(header[14:18], 2)
	out := append(header, cont...)
	out = append(out, data...)
	return out
}

func TestFLACSaveWritableSourceFullRewrite(t *testing.T) {
	src := &memWritable{data: buildFLACCommentFile(t, flac.Field{Name: "TITLE", Value: "Before"})}
	f, err := OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	f.SetTitle("After")
	f.SetCoverArt("image/png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 13, 'I', 'H', 'D', 'R', 0, 0, 0, 1, 0, 0, 0, 1, 8, 0, 0, 0, 0})
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
		t.Fatalf("Title after save = %q", got)
	}
	if len(g.Images()) == 0 {
		t.Fatal("FLAC picture lost after save")
	}
}

func TestWAVSaveWritableSourceFullRewrite(t *testing.T) {
	src := &memWritable{data: buildWAVInfoFile(map[string]string{
		"INAM": "before",
		"IART": "artist",
	})}
	f, err := OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	f.SetTitle("after")
	f.SetAlbum("album")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := OpenBytes(src.data)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.Container() != ContainerWAV {
		t.Fatalf("container = %v, want WAV", g.Container())
	}
	if got := g.Title(); got != "after" {
		t.Fatalf("Title after save = %q", got)
	}
	if tg := g.Tag(TagRIFFInfo); tg == nil || tg.Get("IPRD") != "album" {
		t.Fatalf("RIFF INFO album missing after save")
	}
}

func TestASFSaveWritableSourceRewrite(t *testing.T) {
	src := &memWritable{data: buildASFFile(t, &asfView{
		Fields: []asfField{{Name: "Title", Value: "before"}},
	})}
	f, err := OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	f.SetTitle("after")
	f.SetArtist("artist")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := OpenBytes(src.data)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.Container() != ContainerASF {
		t.Fatalf("container = %v, want ASF", g.Container())
	}
	if got := g.Title(); got != "after" {
		t.Fatalf("Title after save = %q", got)
	}
	if got := g.Artist(); got != "artist" {
		t.Fatalf("Artist after save = %q", got)
	}
}

func TestRealMediaSaveWritableSourceRewrite(t *testing.T) {
	src := &memWritable{data: buildRealMediaFile(t, &realMediaView{
		Fields: []realMediaField{{Name: "Title", Value: "before"}},
	})}
	f, err := OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	f.SetTitle("after")
	f.SetComment("note")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := OpenBytes(src.data)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.Container() != ContainerRealMedia {
		t.Fatalf("container = %v, want RealMedia", g.Container())
	}
	if got := g.Title(); got != "after" {
		t.Fatalf("Title after save = %q", got)
	}
	if got := g.Comment(); got != "note" {
		t.Fatalf("Comment after save = %q", got)
	}
}
