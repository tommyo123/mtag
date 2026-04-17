package tests

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/tommyo123/mtag/flac"
	"github.com/tommyo123/mtag/id3v2"
)

// testdataDir returns the absolute path to the tests/testdata directory.
// It works from any working directory by resolving relative to the
// source file location.
func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test source file location")
	}
	return filepath.Join(filepath.Dir(file), "testdata")
}

// testdataPath resolves a filename inside tests/testdata/.
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(testdataDir(t), name)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("testdata fixture %q not found: %v", name, err)
	}
	return path
}

// testdataCopy copies a testdata fixture into t.TempDir() so destructive
// tests never modify the original. Returns the path to the copy.
func testdataCopy(t *testing.T, name string) string {
	t.Helper()
	src := testdataPath(t, name)
	dir := t.TempDir()
	dst := filepath.Join(dir, filepath.Base(src))
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copy %s: %v", name, err)
	}
	return dst
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// errStr implements error for inline test stubs.
type errStr string

func (e errStr) Error() string { return string(e) }

// writableBuffer is a simple in-memory WritableSource for tests that
// need to exercise OpenWritableSource without touching disk.
type writableBuffer struct {
	mu   sync.Mutex
	data []byte
}

func (b *writableBuffer) ReadAt(p []byte, off int64) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n := copy(p, b.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (b *writableBuffer) WriteAt(p []byte, off int64) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	end := int(off) + len(p)
	if end > len(b.data) {
		b.data = append(b.data, make([]byte, end-len(b.data))...)
	}
	copy(b.data[off:], p)
	return len(p), nil
}

func (b *writableBuffer) Truncate(size int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if int(size) < len(b.data) {
		b.data = b.data[:size]
	}
	return nil
}

// mp4Atom builds a single MP4 atom: 4-byte BE size + 4-byte name + body.
func mp4Atom(name string, body []byte) []byte {
	size := uint32(8 + len(body))
	var hdr [8]byte
	binary.BigEndian.PutUint32(hdr[0:4], size)
	copy(hdr[4:8], name)
	out := make([]byte, 0, int(size))
	out = append(out, hdr[:]...)
	out = append(out, body...)
	return out
}

// buildMinimalMP4 wraps ilst body + optional chpl into ftyp+moov+udta+meta.
func buildMinimalMP4(ilstBody []byte, chplBody []byte) []byte {
	ftyp := mp4Atom("ftyp", []byte("M4A "))
	ilst := mp4Atom("ilst", ilstBody)
	metaBody := append([]byte{0, 0, 0, 0}, ilst...)
	meta := mp4Atom("meta", metaBody)
	var udtaBody []byte
	udtaBody = append(udtaBody, meta...)
	if chplBody != nil {
		udtaBody = append(udtaBody, mp4Atom("chpl", chplBody)...)
	}
	udta := mp4Atom("udta", udtaBody)
	moov := mp4Atom("moov", udta)
	out := append([]byte{}, ftyp...)
	return append(out, moov...)
}

// buildMDTAMP4 builds a minimal MP4 with a single mdta key-value pair.
func buildMDTAMP4(key, value string) []byte {
	// Encode a minimal mdta-style meta atom: keys + ilst with one item.
	keyAtom := mp4Atom("mdta", []byte(key))
	keysBody := make([]byte, 4) // version + flags
	keysBody = append(keysBody, keyAtom...)
	keys := mp4Atom("keys", keysBody)
	// ilst item: index 1 (big-endian uint32) wrapping a data atom.
	var idx [4]byte
	binary.BigEndian.PutUint32(idx[:], 1)
	dataBody := make([]byte, 8) // type=1 (UTF-8) + locale=0
	dataBody[3] = 1
	dataBody = append(dataBody, []byte(value)...)
	data := mp4Atom("data", dataBody)
	item := mp4Atom(string(idx[:]), data)
	ilst := mp4Atom("ilst", item)
	metaBody := make([]byte, 4) // version + flags
	metaBody = append(metaBody, mp4Atom("hdlr", mdtaHandler())...)
	metaBody = append(metaBody, keys...)
	metaBody = append(metaBody, ilst...)
	meta := mp4Atom("meta", metaBody)
	ftyp := mp4Atom("ftyp", []byte("M4A "))
	moov := mp4Atom("moov", mp4Atom("udta", meta))
	out := append([]byte{}, ftyp...)
	return append(out, moov...)
}

func mdtaHandler() []byte {
	body := make([]byte, 24)
	copy(body[8:12], "mdta")
	return body
}

// buildTwoFrameMP3 creates an MP3-like file with an ID3v2 tag (with
// the given padding) plus two fake MPEG sync frames. Used by tests
// that need a "valid enough" MP3 to exercise the write path.
func buildTwoFrameMP3(padding int) []byte {
	tag := id3v2.NewTag(4)
	tag.SetText(id3v2.FrameTitle, "Test")
	body, err := tag.Encode(padding)
	if err != nil {
		panic(err)
	}
	frame := make([]byte, 417)
	frame[0], frame[1] = 0xFF, 0xFB
	body = append(body, frame...)
	body = append(body, frame...)
	return body
}

// buildMinimalFLAC creates a tiny but valid FLAC in memory.
func buildMinimalFLAC(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.Write(flac.Magic[:])
	info := make([]byte, 34)
	binary.BigEndian.PutUint16(info[0:2], 4096)
	binary.BigEndian.PutUint16(info[2:4], 4096)
	info[10] = byte(44100 >> 12)
	info[11] = byte((44100 >> 4) & 0xFF)
	info[12] = byte((44100&0xF)<<4) | 0x03
	info[13] = 0xF0
	if err := flac.WriteBlock(&buf, flac.Block{
		Type: flac.BlockStreamInfo, IsLast: true, Body: info,
	}); err != nil {
		t.Fatal(err)
	}
	buf.Write(make([]byte, 1024)) // pad
	return buf.Bytes()
}

// buildFLACWithBlocks creates a FLAC with the given metadata blocks.
func buildFLACWithBlocks(t *testing.T, blocks ...flac.Block) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.Write(flac.Magic[:])
	for i := range blocks {
		blocks[i].IsLast = i == len(blocks)-1
		if err := flac.WriteBlock(&buf, blocks[i]); err != nil {
			t.Fatal(err)
		}
	}
	return buf.Bytes()
}

// buildW64WithSummary creates a minimal Wave64 file. The fields map
// is currently unused (the fixture just needs valid structure).
func buildW64WithSummary(fields map[string]string) []byte {
	// Wave64 uses 128-bit GUIDs as chunk IDs. Build the absolute minimum.
	var buf bytes.Buffer
	// RIFF GUID
	riffGUID := []byte{0x72, 0x69, 0x66, 0x66, 0x2E, 0x91, 0xCF, 0x11, 0xA5, 0xD6, 0x28, 0xDB, 0x04, 0xC1, 0x00, 0x00}
	buf.Write(riffGUID)
	// File size placeholder (8 bytes LE, will patch)
	sizePos := buf.Len()
	buf.Write(make([]byte, 8))
	// WAVE GUID
	waveGUID := []byte{0x77, 0x61, 0x76, 0x65, 0xF3, 0xAC, 0xD3, 0x11, 0x8C, 0xD1, 0x00, 0xC0, 0x4F, 0x8E, 0xDB, 0x8A}
	buf.Write(waveGUID)
	// fmt chunk (minimal PCM)
	fmtGUID := []byte{0x66, 0x6D, 0x74, 0x20, 0xF3, 0xAC, 0xD3, 0x11, 0x8C, 0xD1, 0x00, 0xC0, 0x4F, 0x8E, 0xDB, 0x8A}
	buf.Write(fmtGUID)
	fmtSize := uint64(24 + 16) // 16 bytes PCM fmt + 24 header
	binary.LittleEndian.PutUint64(buf.Bytes()[buf.Len():buf.Len()], fmtSize)
	var fmtSz [8]byte
	binary.LittleEndian.PutUint64(fmtSz[:], fmtSize)
	buf.Write(fmtSz[:])
	// PCM format: 1 channel, 44100 Hz, 16-bit
	var fmt [16]byte
	binary.LittleEndian.PutUint16(fmt[0:2], 1)     // PCM
	binary.LittleEndian.PutUint16(fmt[2:4], 1)     // mono
	binary.LittleEndian.PutUint32(fmt[4:8], 44100) // sample rate
	binary.LittleEndian.PutUint32(fmt[8:12], 88200)
	binary.LittleEndian.PutUint16(fmt[12:14], 2)
	binary.LittleEndian.PutUint16(fmt[14:16], 16)
	buf.Write(fmt[:])
	// data chunk (1 second silence)
	dataGUID := []byte{0x64, 0x61, 0x74, 0x61, 0xF3, 0xAC, 0xD3, 0x11, 0x8C, 0xD1, 0x00, 0xC0, 0x4F, 0x8E, 0xDB, 0x8A}
	buf.Write(dataGUID)
	dataLen := 44100 * 2
	var dataSz [8]byte
	binary.LittleEndian.PutUint64(dataSz[:], uint64(24+dataLen))
	buf.Write(dataSz[:])
	buf.Write(make([]byte, dataLen))
	// Patch file size
	data := buf.Bytes()
	binary.LittleEndian.PutUint64(data[sizePos:sizePos+8], uint64(len(data)))
	return data
}
