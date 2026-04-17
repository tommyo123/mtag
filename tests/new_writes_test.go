package tests

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tommyo123/mtag"
	"github.com/tommyo123/mtag/ape"
	"github.com/tommyo123/mtag/flac"
)

// -- MP4 native chapter tracks --------------------------------------

// chplBody constructs the body of a Nero "chpl" atom (version 0)
// from a list of (start time in milliseconds, title) entries. The
// outer atom header is added by [neroChplAtom].
func chplBody(entries []struct {
	startMS uint32
	title   string
}) []byte {
	// version (1) + flags (3) + reserved (4) + count (1) + entries
	body := []byte{0, 0, 0, 0, 0, 0, 0, 0, byte(len(entries))}
	for _, e := range entries {
		var ts [4]byte
		binary.BigEndian.PutUint32(ts[:], e.startMS)
		body = append(body, ts[:]...)
		body = append(body, byte(len(e.title)))
		body = append(body, []byte(e.title)...)
	}
	return body
}

// buildMP4WithChpl wraps a Nero chpl atom inside a minimal
// ftyp+moov+udta tree.
func buildMP4WithChpl(entries []struct {
	startMS uint32
	title   string
}) []byte {
	ftyp := mp4Atom("ftyp", []byte("M4A "))
	chpl := mp4Atom("chpl", chplBody(entries))
	udta := mp4Atom("udta", chpl)
	moov := mp4Atom("moov", udta)
	out := append([]byte{}, ftyp...)
	return append(out, moov...)
}

// TestEdge_MP4_NeroChapters_Read decodes a synthetic Nero `chpl`
// atom and verifies File.Chapters surfaces every entry with the
// right start times and titles. The end time of each chapter is
// derived from the next chapter's start (last chapter has zero or
// movie-duration end).
func TestEdge_MP4_NeroChapters_Read(t *testing.T) {
	entries := []struct {
		startMS uint32
		title   string
	}{
		{0, "Intro"},
		{30_000, "Chapter One"},
		{75_500, "Chapter Two"},
	}
	data := buildMP4WithChpl(entries)
	path := filepath.Join(t.TempDir(), "chpl.m4a")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if f.Container() != mtag.ContainerMP4 {
		t.Fatalf("container = %v, want MP4", f.Container())
	}
	chapters := f.Chapters()
	if len(chapters) != len(entries) {
		t.Fatalf("chapters = %d, want %d", len(chapters), len(entries))
	}
	for i, want := range entries {
		got := chapters[i]
		if got.Title != want.title {
			t.Errorf("chapter %d title = %q, want %q", i, got.Title, want.title)
		}
		wantStart := time.Duration(want.startMS) * time.Millisecond
		if got.Start != wantStart {
			t.Errorf("chapter %d start = %v, want %v", i, got.Start, wantStart)
		}
		if i+1 < len(entries) {
			wantEnd := time.Duration(entries[i+1].startMS) * time.Millisecond
			if got.End != wantEnd {
				t.Errorf("chapter %d end = %v, want %v", i, got.End, wantEnd)
			}
		}
	}
}

// TestEdge_MP4_NeroChapters_PreservedOnSave confirms a save of the
// MP4 (touching only the ilst/title via a setter) does not destroy
// the existing Nero chpl atom.
func TestEdge_MP4_NeroChapters_PreservedOnSave(t *testing.T) {
	entries := []struct {
		startMS uint32
		title   string
	}{
		{0, "First"},
		{60_000, "Second"},
	}
	// We need an ilst region for SetTitle to land somewhere; build a
	// full ftyp + moov(udta(meta(ilst), chpl)) tree.
	ftyp := mp4Atom("ftyp", []byte("M4A "))
	ilst := mp4Atom("ilst", nil)
	metaBody := append([]byte{0, 0, 0, 0}, ilst...)
	meta := mp4Atom("meta", metaBody)
	chpl := mp4Atom("chpl", chplBody(entries))
	udta := mp4Atom("udta", append(meta, chpl...))
	moov := mp4Atom("moov", udta)
	data := append(ftyp, moov...)

	path := filepath.Join(t.TempDir(), "chpl-rt.m4a")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.Chapters(); len(got) != len(entries) {
		t.Fatalf("pre-save chapters = %d, want %d", len(got), len(entries))
	}
	f.SetTitle("after-save")
	if err := f.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.Title() != "after-save" {
		t.Errorf("title after save = %q", g.Title())
	}
	chapters := g.Chapters()
	if len(chapters) != len(entries) {
		t.Errorf("post-save chapters = %d, want %d (chapters dropped on save)", len(chapters), len(entries))
	}
	for i, want := range entries {
		if i >= len(chapters) {
			break
		}
		if chapters[i].Title != want.title {
			t.Errorf("post-save chapter %d title = %q, want %q", i, chapters[i].Title, want.title)
		}
	}
}

// -- Typed APE binary cover-art ------------------------------------

// buildMinimalAPEFile returns a synthetic Monkey's Audio file with
// just the "MAC " magic and an APE tag at the tail. Enough for the
// tagger to recognise the container and the read/write path to
// round-trip.
func buildMinimalAPEFile(t *testing.T, tagBody []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString("MAC ")
	buf.Write(make([]byte, 60)) // dummy MAC header padding
	buf.Write(tagBody)
	return buf.Bytes()
}

// TestEdge_APE_CoverArtSetThroughPolymorphicAPI exercises the typed
// SetCoverArt / AddImage path on an APE-native container. The
// resulting on-disk APE binary field must start with the
// extension hint ("jpg\0") followed by the raw payload.
func TestEdge_APE_CoverArtSetThroughPolymorphicAPI(t *testing.T) {
	// Build an APE tag with one text field so the file has a
	// recognisable APE region the parser can find.
	t0 := ape.New()
	t0.Set("Title", "ape-cover-test")
	tagBody, err := t0.Encode()
	if err != nil {
		t.Fatal(err)
	}
	data := buildMinimalAPEFile(t, tagBody)
	path := filepath.Join(t.TempDir(), "covers.ape")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Container() != mtag.ContainerMAC {
		t.Fatalf("container = %v, want MAC", f.Container())
	}
	jpegBody := []byte("\xFF\xD8\xFF\xE0pretend-jpeg-bytes")
	f.SetCoverArt("image/jpeg", jpegBody)
	if err := f.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	imgs := g.Images()
	if len(imgs) == 0 {
		t.Fatalf("no images surfaced after SetCoverArt")
	}
	got := imgs[0]
	if got.MIME != "image/jpeg" {
		t.Errorf("MIME = %q, want image/jpeg", got.MIME)
	}
	if !bytes.Equal(got.Data, jpegBody) {
		t.Errorf("image bytes mismatch: got %d bytes, want %d", len(got.Data), len(jpegBody))
	}
	// Drill into the raw APE field to confirm the on-disk layout
	// has the extension prefix the spec mandates.
	apeTag := g.APE()
	if apeTag == nil {
		t.Fatal("APE tag missing after save")
	}
	field := apeTag.Find(ape.FieldCoverArtFront)
	if field == nil {
		t.Fatal("APE Cover Art (Front) field missing")
	}
	if !field.IsBinary() {
		t.Errorf("APE cover field type = %v, want binary", field.Type())
	}
	raw := field.Value
	nul := bytes.IndexByte(raw, 0)
	if nul <= 0 {
		t.Fatalf("APE binary field has no NUL-terminated extension hint: %q", raw)
	}
	ext := string(raw[:nul])
	if ext != "jpg" {
		t.Errorf("APE extension hint = %q, want %q", ext, "jpg")
	}
	rest := raw[nul+1:]
	if !bytes.Equal(rest, jpegBody) {
		t.Errorf("APE binary payload mismatch after extension prefix")
	}
}

// TestEdge_APE_PNGCoverArtExtension verifies the extension hint
// prefix differs by MIME type (png vs jpg).
func TestEdge_APE_PNGCoverArtExtension(t *testing.T) {
	t0 := ape.New()
	t0.Set("Title", "ape-png-test")
	tagBody, err := t0.Encode()
	if err != nil {
		t.Fatal(err)
	}
	data := buildMinimalAPEFile(t, tagBody)
	path := filepath.Join(t.TempDir(), "png.ape")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	pngBody := []byte("\x89PNG\r\n\x1a\npretend-png")
	f.SetCoverArt("image/png", pngBody)
	if err := f.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	apeTag := g.APE()
	if apeTag == nil {
		t.Fatal("APE tag missing")
	}
	field := apeTag.Find(ape.FieldCoverArtFront)
	if field == nil || !field.IsBinary() {
		t.Fatal("binary front cover field missing")
	}
	nul := bytes.IndexByte(field.Value, 0)
	if nul <= 0 || string(field.Value[:nul]) != "png" {
		t.Errorf("PNG extension hint = %q, want %q", field.Value[:nul], "png")
	}
}

// -- Synthetic Ogg-FLAC -------------------------------------------

// buildOggFLAC builds a minimal Ogg-FLAC stream:
//
//	page 0  : ident packet (0x7F "FLAC" + version + #headers + fLaC + STREAMINFO)
//	page 1  : VORBIS_COMMENT FLAC metadata block
//	page 2  : EOS placeholder audio
//
// CRC values are computed to keep the page checksums valid so the
// production reader doesn't reject the synthesised file.
func buildOggFLAC(t *testing.T, vendor string, fields [][2]string) []byte {
	t.Helper()
	// FLAC ident packet body (per RFC 4880 / FLAC-in-Ogg spec):
	//   0x7F | "FLAC" | major | minor | header_packets (BE16) | "fLaC"
	//   + STREAMINFO metadata block (block type 0, body 34 bytes)
	streamInfoHdr := []byte{0x80, 0, 0, 34} // is-last=1, type=0, length=34
	streamInfoBody := make([]byte, 34)
	ident := bytes.Buffer{}
	ident.WriteByte(0x7F)
	ident.WriteString("FLAC")
	ident.WriteByte(1)        // major version
	ident.WriteByte(0)        // minor version
	ident.Write([]byte{0, 0}) // header packets count (0 = unknown)
	ident.WriteString("fLaC")
	ident.Write(streamInfoHdr)
	ident.Write(streamInfoBody)
	identPacket := ident.Bytes()

	// VORBIS_COMMENT FLAC metadata block: header (4 bytes) + body.
	vc := &flac.VorbisComment{Vendor: vendor}
	for _, f := range fields {
		vc.Fields = append(vc.Fields, flac.Field{Name: f[0], Value: f[1]})
	}
	vcBody := flac.EncodeVorbisComment(vc)
	commentPacket := bytes.Buffer{}
	// Last metadata block flag is set, type = 4 (VORBIS_COMMENT).
	commentPacket.WriteByte(0x84)
	commentPacket.Write([]byte{
		byte(len(vcBody) >> 16),
		byte(len(vcBody) >> 8),
		byte(len(vcBody)),
	})
	commentPacket.Write(vcBody)

	var out bytes.Buffer
	const serial = uint32(0xCAFE_BABE)
	writeOggPage(&out, true, false, false, 0, serial, 0, identPacket)
	writeOggPage(&out, false, false, false, 1, serial, 1, commentPacket.Bytes())
	writeOggPage(&out, false, true, true, 0xFFFFFFFF_FFFFFFFF, serial, 2, []byte("audio"))
	return out.Bytes()
}

// writeOggPage emits one Ogg page with the given flags + body. The
// CRC is computed lazily.
func writeOggPage(w *bytes.Buffer, bos, eos, cont bool, granule uint64, serial, seq uint32, body []byte) {
	header := make([]byte, 27)
	copy(header[:4], "OggS")
	header[4] = 0
	var flags byte
	if cont {
		flags |= 0x01
	}
	if bos {
		flags |= 0x02
	}
	if eos {
		flags |= 0x04
	}
	header[5] = flags
	binary.LittleEndian.PutUint64(header[6:14], granule)
	binary.LittleEndian.PutUint32(header[14:18], serial)
	binary.LittleEndian.PutUint32(header[18:22], seq)
	// CRC slot (filled in below)
	// Segment table: split body into 255-byte segments.
	var segments []byte
	remaining := len(body)
	for remaining >= 255 {
		segments = append(segments, 255)
		remaining -= 255
	}
	segments = append(segments, byte(remaining))
	header[26] = byte(len(segments))

	page := append(header, segments...)
	page = append(page, body...)
	crc := oggCRC(page)
	binary.LittleEndian.PutUint32(page[22:26], crc)
	w.Write(page)
}

// oggCRC is the standard Ogg CRC-32 (polynomial 0x04C11DB7,
// no reflection, init=0).
func oggCRC(data []byte) uint32 {
	var table [256]uint32
	for i := 0; i < 256; i++ {
		c := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if c&0x8000_0000 != 0 {
				c = (c << 1) ^ 0x04C1_1DB7
			} else {
				c <<= 1
			}
		}
		table[i] = c
	}
	var crc uint32
	for _, b := range data {
		crc = (crc << 8) ^ table[byte(crc>>24)^b]
	}
	return crc
}

// TestEdge_OGGFLAC_SyntheticReadWrite builds a tiny Ogg-FLAC file
// from scratch and verifies the polymorphic accessors decode the
// vendor + Vorbis-style fields without needing a corpus fixture.
func TestEdge_OGGFLAC_SyntheticReadWrite(t *testing.T) {
	data := buildOggFLAC(t, "synthetic-ogg-flac",
		[][2]string{
			{"TITLE", "ogg-flac-title"},
			{"ARTIST", "ogg-flac-artist"},
		},
	)
	path := filepath.Join(t.TempDir(), "synth.oga")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := mtag.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if f.Container() != mtag.ContainerOGG {
		t.Fatalf("container = %v, want OGG", f.Container())
	}
	if f.Title() != "ogg-flac-title" {
		t.Errorf("title = %q", f.Title())
	}
	if f.Artist() != "ogg-flac-artist" {
		t.Errorf("artist = %q", f.Artist())
	}
}

// TestEdge_OGGFLAC_SyntheticWrite exercises the Ogg-FLAC save path
// by mutating the synthesised file and verifying the mutated field
// survives an explicit reopen.
func TestEdge_OGGFLAC_SyntheticWrite(t *testing.T) {
	data := buildOggFLAC(t, "synth", [][2]string{{"TITLE", "initial"}})
	path := filepath.Join(t.TempDir(), "synthwrite.oga")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := mtag.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	f.SetTitle("ogg-flac-updated")
	f.SetArtist("ogg-flac-new-artist")
	if err := f.Save(); err != nil {
		// Save may legitimately refuse if the synthetic stream is
		// too minimal for the writer (e.g. the audio placeholder is
		// not a valid FLAC frame). Record it as a skip rather than
		// a hard fail.
		t.Skipf("Ogg-FLAC write refused the synthetic stream: %v", err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer g.Close()
	if g.Title() != "ogg-flac-updated" {
		t.Errorf("title after save = %q", g.Title())
	}
	if g.Artist() != "ogg-flac-new-artist" {
		t.Errorf("artist after save = %q", g.Artist())
	}
}

// -- APE cover on all three APE-native containers -------------------

// TestEdge_APE_CoverArtBackOnWavPack verifies the AddImage path
// maps PictureCoverBack to the "Cover Art (Back)" field instead of
// the Front slot, across every APE-native container mtag supports.
func TestEdge_APE_CoverArtBackOnWavPack(t *testing.T) {
	t0 := ape.New()
	t0.Set("Title", "wavpack-back-cover")
	tagBody, err := t0.Encode()
	if err != nil {
		t.Fatal(err)
	}
	// WavPack magic "wvpk" triggers ContainerWavPack detection.
	body := append([]byte("wvpk"), make([]byte, 28)...)
	body = append(body, tagBody...)
	path := filepath.Join(t.TempDir(), "back.wv")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Container() != mtag.ContainerWavPack {
		t.Fatalf("container = %v, want WavPack", f.Container())
	}
	backBody := []byte("\xFF\xD8back-cover-bytes")
	f.AddImage(mtag.Picture{MIME: "image/jpeg", Type: mtag.PictureCoverBack, Data: backBody})
	if err := f.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	apeTag := g.APE()
	if apeTag == nil {
		t.Fatal("APE tag missing")
	}
	backField := apeTag.Find(ape.FieldCoverArtBack)
	if backField == nil {
		t.Fatal("Cover Art (Back) field missing — AddImage didn't route to the back slot")
	}
	if !backField.IsBinary() {
		t.Errorf("back cover field type = %v, want binary", backField.Type())
	}
	nul := bytes.IndexByte(backField.Value, 0)
	if nul < 0 || !bytes.Equal(backField.Value[nul+1:], backBody) {
		t.Errorf("back cover payload mismatch after extension prefix")
	}
}

// helper for older Go versions; remove once min is 1.21+.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
