package mtag

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/tommyo123/mtag/flac"
)

type sparseRange struct {
	off  int64
	data []byte
}

type sparseReaderAt struct {
	size      int64
	ranges    []sparseRange
	forbidden [][2]int64
}

func (r sparseReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= r.size {
		return 0, io.EOF
	}
	end := off + int64(len(p))
	if end > r.size {
		end = r.size
	}
	for _, rng := range r.forbidden {
		if off < rng[1] && end > rng[0] {
			return 0, io.ErrUnexpectedEOF
		}
	}
	n := int(end - off)
	clear(p[:n])
	for _, rg := range r.ranges {
		rgEnd := rg.off + int64(len(rg.data))
		if off >= rgEnd || end <= rg.off {
			continue
		}
		copyStart := max64(off, rg.off)
		copyEnd := min64(end, rgEnd)
		copy(p[copyStart-off:copyEnd-off], rg.data[copyStart-rg.off:copyEnd-rg.off])
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func TestReadASFMetadataSkipsOversizeObject(t *testing.T) {
	size := int64(30 + 24 + maxASFObjectBytes + 1)
	head := make([]byte, 30)
	copy(head[:16], asfHeaderObjectGUID[:])
	binary.LittleEndian.PutUint64(head[16:24], uint64(size))
	binary.LittleEndian.PutUint32(head[24:28], 1)

	obj := make([]byte, 24)
	copy(obj[:16], asfExtContentGUID[:])
	binary.LittleEndian.PutUint64(obj[16:24], uint64(24+maxASFObjectBytes+1))

	r := sparseReaderAt{
		size: size,
		ranges: []sparseRange{
			{off: 0, data: head},
			{off: 30, data: obj},
		},
		forbidden: [][2]int64{{54, size}},
	}

	view, err := readASFMetadataWithOptions(r, size, false)
	if err != nil {
		t.Fatalf("readASFMetadataWithOptions() error = %v", err)
	}
	if view == nil {
		t.Fatal("view = nil, want empty view")
	}
	if len(view.Fields) != 0 || len(view.Pictures) != 0 {
		t.Fatalf("view = %+v, want empty", view)
	}
}

func TestScanCAFInfoSkipsOversizeChunk(t *testing.T) {
	size := int64(8 + 12 + maxCAFInfoChunkBytes + 1)
	head := []byte{'c', 'a', 'f', 'f', 0, 1, 0, 0}
	chunk := make([]byte, 12)
	copy(chunk[:4], []byte("info"))
	binary.BigEndian.PutUint64(chunk[4:12], uint64(maxCAFInfoChunkBytes+1))

	r := sparseReaderAt{
		size: size,
		ranges: []sparseRange{
			{off: 0, data: head},
			{off: 8, data: chunk},
		},
		forbidden: [][2]int64{{20, size}},
	}

	if got := scanCAFInfo(r, size); got != nil {
		t.Fatalf("scanCAFInfo() = %#v, want nil", got)
	}
}

func TestReadRealMediaMetadataSkipsOversizeObject(t *testing.T) {
	objSize := int64(10 + maxRealMediaObjectBytes + 1)
	size := int64(18) + objSize
	head := make([]byte, 18)
	copy(head[:4], []byte(".RMF"))
	binary.BigEndian.PutUint32(head[4:8], 18)

	obj := make([]byte, 10)
	copy(obj[:4], []byte("CONT"))
	binary.BigEndian.PutUint32(obj[4:8], uint32(objSize))

	r := sparseReaderAt{
		size: size,
		ranges: []sparseRange{
			{off: 0, data: head},
			{off: 18, data: obj},
		},
		forbidden: [][2]int64{{28, size}},
	}

	view, err := readRealMediaMetadata(r, size)
	if err != nil {
		t.Fatalf("readRealMediaMetadata() error = %v", err)
	}
	if view != nil {
		t.Fatalf("view = %#v, want nil", view)
	}
}

func TestParseMatroskaInfoSkipsOversizeElement(t *testing.T) {
	parseMatroskaInfo(sparseReaderAt{size: maxMatroskaElementBytes + 1}, 0, maxMatroskaElementBytes+1, &matroskaView{})
}

func TestDetectFLACSkipPicturesDropsPictureData(t *testing.T) {
	vc := flac.EncodeVorbisComment(&flac.VorbisComment{
		Vendor: "mtag-test",
		Fields: []flac.Field{
			{Name: "TITLE", Value: "Song"},
			{Name: "METADATA_BLOCK_PICTURE", Value: "encoded-picture"},
		},
	})
	pic := flac.EncodePicture(&flac.Picture{
		Type: 3,
		MIME: "image/png",
		Data: []byte("\x89PNG\r\n\x1a\npic"),
	})
	var buf bytes.Buffer
	buf.WriteString("fLaC")
	if err := flac.WriteBlock(&buf, flac.Block{Type: flac.BlockVorbisComment, Body: vc}); err != nil {
		t.Fatal(err)
	}
	if err := flac.WriteBlock(&buf, flac.Block{Type: flac.BlockPicture, IsLast: true, Body: pic}); err != nil {
		t.Fatal(err)
	}

	f := testFileForKind(ContainerFLAC)
	f.src = bytes.NewReader(buf.Bytes())
	f.size = int64(buf.Len())

	if err := f.detectFLAC(openConfig{skipPictures: true}); err != nil {
		t.Fatalf("detectFLAC() error = %v", err)
	}
	if f.flac == nil || f.flac.comment == nil {
		t.Fatal("flac view missing")
	}
	if got := f.flac.comment.Get("TITLE"); got != "Song" {
		t.Fatalf("TITLE = %q, want %q", got, "Song")
	}
	if got := f.flac.comment.Get("METADATA_BLOCK_PICTURE"); got != "" {
		t.Fatalf("METADATA_BLOCK_PICTURE = %q, want empty", got)
	}
	if len(f.flac.pictures) != 0 {
		t.Fatalf("len(pictures) = %d, want 0", len(f.flac.pictures))
	}
}

func TestDecodeOGGCommentWithOptionsSkipsPictureFields(t *testing.T) {
	vc := flac.EncodeVorbisComment(&flac.VorbisComment{
		Vendor: "mtag-test",
		Fields: []flac.Field{
			{Name: "TITLE", Value: "Song"},
			{Name: "METADATA_BLOCK_PICTURE", Value: "encoded-picture"},
		},
	})
	packet := append([]byte{0x03, 'v', 'o', 'r', 'b', 'i', 's'}, vc...)
	view, err := decodeOGGCommentWithOptions([]byte{0x01, 'v', 'o', 'r', 'b', 'i', 's'}, packet, true)
	if err != nil {
		t.Fatalf("decodeOGGCommentWithOptions() error = %v", err)
	}
	if view == nil || view.comment == nil {
		t.Fatal("view missing")
	}
	if got := view.comment.Get("TITLE"); got != "Song" {
		t.Fatalf("TITLE = %q, want %q", got, "Song")
	}
	if got := view.comment.Get("METADATA_BLOCK_PICTURE"); got != "" {
		t.Fatalf("METADATA_BLOCK_PICTURE = %q, want empty", got)
	}
}
