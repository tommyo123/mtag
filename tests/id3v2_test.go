package tests

import (
	"bytes"
	"compress/zlib"
	"testing"

	"github.com/tommyo123/mtag/id3v2"
)

// id3v2BytesReader adapts a byte slice to io.ReaderAt for round-
// trip tests that build a tag in memory, serialise it, and parse
// it back.
type id3v2BytesReader []byte

func (b id3v2BytesReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(b)) {
		return 0, errStr("EOF")
	}
	n := copy(p, b[off:])
	if n < len(p) {
		return n, errStr("EOF")
	}
	return n, nil
}

// v2Decode is a tiny helper that serialises and re-parses a tag.
func v2Decode(t *testing.T, tag *id3v2.Tag) *id3v2.Tag {
	t.Helper()
	raw, err := tag.Encode(0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := id3v2.Read(id3v2BytesReader(raw), 0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestID3v2_SynchsafeRoundTrip(t *testing.T) {
	cases := []uint32{0, 1, 127, 128, 0xFFFF, 0xFFFFF, 0xFFFFFFF}
	for _, v := range cases {
		var b [4]byte
		if err := id3v2.EncodeSynchsafe(b[:], v); err != nil {
			t.Fatalf("encode %d: %v", v, err)
		}
		if got := id3v2.DecodeSynchsafe(b[:]); got != v {
			t.Errorf("synchsafe round-trip: got %d want %d", got, v)
		}
	}
}

func TestID3v2_UnsynchronisationRoundTrip(t *testing.T) {
	src := []byte{0xFF, 0xE0, 'x', 0xFF, 0x00, 0xFF, 0xFF, 0xE0}
	un := id3v2.Unsynchronise(src)
	re := id3v2.Resynchronise(un)
	if !bytes.Equal(src, re) {
		t.Errorf("round-trip mismatch\n src: %v\n un:  %v\n re:  %v", src, un, re)
	}
}

// TestID3v2_UTF16BOM verifies mtag's UTF-16 writer emits the
// little-endian BOM (0xFF 0xFE) that real-world decoders expect,
// and that strict BOM-aware decoders round-trip our output.
func TestID3v2_UTF16BOM(t *testing.T) {
	s := "fjäril"
	b, err := id3v2.EncodeString(id3v2.EncUTF16, s, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 2 {
		t.Fatalf("too short: %v", b)
	}
	// Spec allows either endian; mtag emits LE for maximum
	// compatibility with legacy players.
	if b[0] != 0xFF || b[1] != 0xFE {
		t.Errorf("UTF-16 output should start with LE BOM 0xFF 0xFE, got %x %x", b[0], b[1])
	}
	// The BE-aware decoder must also accept our LE BOM
	// (DecodeString(UTF16) is BOM-aware).
	got, err := id3v2.DecodeString(id3v2.EncUTF16, b)
	if err != nil {
		t.Fatal(err)
	}
	if got != s {
		t.Errorf("round-trip: got %q, want %q", got, s)
	}
	// A manually-built BE-BOM buffer must also round-trip (the
	// decoder ignores which endianness the BOM selected).
	beBOM := []byte{0xFE, 0xFF}
	beBOM = append(beBOM, 0x00, 'h', 0x00, 'i')
	got, err = id3v2.DecodeString(id3v2.EncUTF16, beBOM)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hi" {
		t.Errorf("BE BOM round-trip: got %q, want %q", got, "hi")
	}
}

func TestID3v2_TextEncodingRoundTrip(t *testing.T) {
	values := []string{"ASCII only", "Kafé – 日本語", ""}
	encs := []id3v2.Encoding{id3v2.EncISO8859, id3v2.EncUTF8, id3v2.EncUTF16, id3v2.EncUTF16BE}
	for _, e := range encs {
		for _, s := range values {
			b, err := id3v2.EncodeString(e, s, false)
			if err != nil {
				t.Fatalf("encode %v %q: %v", e, s, err)
			}
			got, err := id3v2.DecodeString(e, b)
			if err != nil {
				t.Fatalf("decode %v %q: %v", e, s, err)
			}
			if e == id3v2.EncISO8859 {
				if s == "ASCII only" && got != s {
					t.Errorf("latin1 ascii: got %q want %q", got, s)
				}
				continue
			}
			if got != s {
				t.Errorf("enc %v: got %q want %q", e, got, s)
			}
		}
	}
}

// Generic placeholders used by the round-trip tests — see the
// equivalent block in id3v1_test.go for the rationale (the tests
// validate byte-level round-tripping; the specific text content is
// irrelevant beyond being recognisable in failure messages).
const (
	v2Title   = "sample title"
	v2Artist  = "sample artist"
	v2Album   = "sample album"
	v2Comment = "sample comment"
	v2Lang    = "eng"
	// minimalJPEG is just enough JFIF header for callers that
	// inspect the picture body to confirm the bytes round-tripped.
	v2PictureMIME = "image/jpeg"
)

var minimalJPEG = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}

func TestID3v2_TagRoundTripV24(t *testing.T) {
	tag := id3v2.NewTag(4)
	tag.Set(&id3v2.TextFrame{FrameID: id3v2.FrameTitle, Values: []string{v2Title}})
	tag.Set(&id3v2.TextFrame{FrameID: id3v2.FrameArtist, Values: []string{v2Artist}})
	tag.Set(&id3v2.TextFrame{FrameID: id3v2.FrameAlbum, Values: []string{v2Album}})
	tag.Set(&id3v2.CommentFrame{FrameID: id3v2.FrameComment, Language: v2Lang, Text: v2Comment})
	tag.Set(&id3v2.PictureFrame{
		MIME:        v2PictureMIME,
		PictureType: byte(3), // PictureCoverFront per APIC spec
		Description: "cover",
		Data:        minimalJPEG,
	})

	parsed := v2Decode(t, tag)
	if got := parsed.Text(id3v2.FrameTitle); got != v2Title {
		t.Errorf("title = %q, want %q", got, v2Title)
	}
	if got := parsed.Text(id3v2.FrameArtist); got != v2Artist {
		t.Errorf("artist = %q, want %q", got, v2Artist)
	}
	c, ok := parsed.Find(id3v2.FrameComment).(*id3v2.CommentFrame)
	if !ok || c.Text != v2Comment || c.Language != v2Lang {
		t.Errorf("comment mismatch: %+v", c)
	}
	pic, ok := parsed.Find(id3v2.FramePicture).(*id3v2.PictureFrame)
	if !ok || pic.MIME != v2PictureMIME || pic.PictureType != 3 {
		t.Errorf("picture mismatch: %+v", pic)
	}
}

func TestID3v2_PictureFrameUnsynchronisedDLIRoundTrip(t *testing.T) {
	tag := id3v2.NewTag(4)
	pic := &id3v2.PictureFrame{
		MIME:        "image/jpeg",
		PictureType: 3,
		Data:        []byte{0xFF, 0xE0, 0x00, 0xFF, 0xD8, 0xFF, 0x00, 0xE1},
	}
	pic.FrameFlags = id3v2.FrameFlags{
		Version: 4,
		Raw:     [2]byte{0x00, 0x03}, // per-frame unsync + DLI
	}
	tag.Set(pic)

	got := v2Decode(t, tag).Find(id3v2.FramePicture)
	apic, ok := got.(*id3v2.PictureFrame)
	if !ok {
		t.Fatalf("APIC re-read as %T, want *id3v2.PictureFrame", got)
	}
	if apic.MIME != pic.MIME || apic.PictureType != pic.PictureType || !bytes.Equal(apic.Data, pic.Data) {
		t.Fatalf("APIC round-trip mismatch: got %+v want %+v", apic, pic)
	}
}

func TestID3v2_V24TagUnsynchronisationFlagDoesNotDoubleEscapeFrames(t *testing.T) {
	tag := id3v2.NewTag(4)
	tag.Unsynchronised = true
	pic := &id3v2.PictureFrame{
		MIME:        "image/jpeg",
		PictureType: 3,
		Data:        []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0xFF, 0xDB},
	}
	pic.FrameFlags = id3v2.FrameFlags{
		Version: 4,
		Raw:     [2]byte{0x00, 0x03}, // per-frame unsync + DLI
	}
	tag.Set(pic)

	got := v2Decode(t, tag).Find(id3v2.FramePicture)
	apic, ok := got.(*id3v2.PictureFrame)
	if !ok {
		t.Fatalf("APIC re-read as %T, want *id3v2.PictureFrame", got)
	}
	if !bytes.Equal(apic.Data, pic.Data) {
		t.Fatalf("APIC bytes changed across v2.4 tag unsync round-trip")
	}
}

// TestID3v2_VersionConversion verifies that re-encoding a tag with
// a different major version translates the frame IDs as the spec
// dictates: TYER (v2.3) ↔ TDRC (v2.4), and the older identifier
// disappears from the output.
func TestID3v2_VersionConversion(t *testing.T) {
	const sampleYear = "1991"
	tag := id3v2.NewTag(3)
	tag.Set(&id3v2.TextFrame{FrameID: id3v2.FrameTitle, Values: []string{v2Title}})
	tag.Set(&id3v2.TextFrame{FrameID: id3v2.FrameYear, Values: []string{sampleYear}})

	tag.Version = 4
	parsed := v2Decode(t, tag)
	if parsed.Version != 4 {
		t.Errorf("version = %d", parsed.Version)
	}
	if got := parsed.Text(id3v2.FrameRecordingTime); got != sampleYear {
		t.Errorf("TDRC = %q, want %q", got, sampleYear)
	}
	if parsed.Text(id3v2.FrameYear) != "" {
		t.Errorf("unexpected TYER in v2.4 output")
	}
}

// TestID3v2_DowngradeToV22 re-encodes a tag in the v2.2 dialect and
// verifies the canonical 4-byte frame IDs were translated to their
// 3-byte v2.2 equivalents in the on-disk output.
func TestID3v2_DowngradeToV22(t *testing.T) {
	tag := id3v2.NewTag(4)
	tag.Set(&id3v2.TextFrame{FrameID: id3v2.FrameTitle, Values: []string{v2Title}})
	tag.Set(&id3v2.TextFrame{FrameID: id3v2.FrameArtist, Values: []string{v2Artist}})
	tag.Version = 2
	raw, err := tag.Encode(0)
	if err != nil {
		t.Fatal(err)
	}
	for _, expectID := range []string{"TT2", "TP1"} {
		if !bytes.Contains(raw, []byte(expectID)) {
			t.Errorf("expected %s frame in v2.2 output; raw=%q", expectID, raw)
		}
	}
}

func TestID3v2_UFIDRoundTrip(t *testing.T) {
	in := &id3v2.UniqueFileIDFrame{
		Owner:      "http://musicbrainz.org",
		Identifier: []byte("aabbccdd-eeff-0011-2233-445566778899"),
	}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FrameUFID).(*id3v2.UniqueFileIDFrame)
	if got.Owner != in.Owner || string(got.Identifier) != string(in.Identifier) {
		t.Errorf("UFID round-trip: %+v vs %+v", got, in)
	}
}

func TestID3v2_GEOBRoundTrip(t *testing.T) {
	in := &id3v2.GeneralObjectFrame{
		MIME:        "application/octet-stream",
		Filename:    "data.bin",
		Description: "binary blob",
		Data:        []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0xFF},
	}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FrameGEOB).(*id3v2.GeneralObjectFrame)
	if got.MIME != in.MIME || got.Filename != in.Filename ||
		got.Description != in.Description || string(got.Data) != string(in.Data) {
		t.Errorf("GEOB round-trip: %+v vs %+v", got, in)
	}
}

func TestID3v2_MCDIRoundTrip(t *testing.T) {
	in := &id3v2.MusicCDIDFrame{TOC: []byte{0x01, 0x02, 0x03, 0x04, 0xFF, 0x00, 0xAA}}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FrameMCDI).(*id3v2.MusicCDIDFrame)
	if string(got.TOC) != string(in.TOC) {
		t.Errorf("MCDI TOC mismatch")
	}
}

func TestID3v2_SYLTRoundTrip(t *testing.T) {
	in := &id3v2.SyncedLyricsFrame{
		Language:    "eng",
		Format:      id3v2.SylTimestampMillis,
		ContentType: id3v2.SylContentLyrics,
		Description: "verse",
		Entries: []id3v2.SylEntry{
			{Text: "Hello", Timestamp: 0},
			{Text: "world", Timestamp: 1500},
			{Text: "again", Timestamp: 3000},
		},
	}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FrameSYLT).(*id3v2.SyncedLyricsFrame)
	if got.Language != in.Language || got.Format != in.Format ||
		got.ContentType != in.ContentType || got.Description != in.Description {
		t.Errorf("SYLT header mismatch: %+v vs %+v", got, in)
	}
	if len(got.Entries) != len(in.Entries) {
		t.Fatalf("entry count: %d vs %d", len(got.Entries), len(in.Entries))
	}
	for i, e := range in.Entries {
		if got.Entries[i] != e {
			t.Errorf("SYLT entry %d: got %+v want %+v", i, got.Entries[i], e)
		}
	}
}

func TestID3v2_RVA2RoundTrip(t *testing.T) {
	in := &id3v2.RVA2Frame{
		Identification: "track",
		Channels: []id3v2.RVA2Adjustment{
			{Channel: id3v2.RVA2MasterVol, Adjustment: -3072, PeakBits: 16, Peak: []byte{0x80, 0x00}},
			{Channel: id3v2.RVA2FrontLeft, Adjustment: -2048, PeakBits: 8, Peak: []byte{0xFF}},
		},
	}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FrameRVA2).(*id3v2.RVA2Frame)
	if got.Identification != in.Identification {
		t.Errorf("identification: %q vs %q", got.Identification, in.Identification)
	}
	if len(got.Channels) != len(in.Channels) {
		t.Fatalf("channel count: %d vs %d", len(got.Channels), len(in.Channels))
	}
	for i, ch := range in.Channels {
		gc := got.Channels[i]
		if gc.Channel != ch.Channel || gc.Adjustment != ch.Adjustment ||
			gc.PeakBits != ch.PeakBits || string(gc.Peak) != string(ch.Peak) {
			t.Errorf("RVA2 channel %d: got %+v want %+v", i, gc, ch)
		}
	}
	if in.Channels[0].AdjustmentDB() != -6.0 {
		t.Errorf("master dB = %v, want -6.0", in.Channels[0].AdjustmentDB())
	}
}

func TestID3v2_AudioTextRoundTrip(t *testing.T) {
	in := &id3v2.AudioTextFrame{
		MIME:      "audio/ogg",
		Scrambled: true,
		Text:      "sample title",
		Audio:     []byte{0x10, 0x20, 0x30, 0x40},
	}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FrameATXT).(*id3v2.AudioTextFrame)
	if got.MIME != in.MIME || got.Text != in.Text || !got.Scrambled || !bytes.Equal(got.Audio, in.Audio) {
		t.Fatalf("ATXT mismatch: got %+v want %+v", got, in)
	}
}

func TestID3v2_EncryptedFrameRoundTrip(t *testing.T) {
	flags := id3v2.FrameFlags{Version: 4}
	flags.Raw[1] = 0x45 // grouped + encrypted + data-length-indicator
	tag := id3v2.NewTag(4)
	tag.Set(&id3v2.EncryptionRegistrationFrame{
		Owner:    "https://example.test/encr",
		MethodID: 7,
		Data:     []byte{1, 2, 3},
	})
	tag.Frames = append(tag.Frames, &id3v2.EncryptedFrame{
		FrameID:                id3v2.FramePrivate,
		MethodID:               7,
		GroupSymbol:            9,
		HasGroupSymbol:         true,
		DataLengthIndicator:    2,
		HasDataLengthIndicator: true,
		Data:                   []byte{0xAA, 0xBB},
		FrameFlags:             flags,
	})
	got := v2Decode(t, tag).Find(id3v2.FramePrivate).(*id3v2.EncryptedFrame)
	if got.MethodID != 7 || got.GroupSymbol != 9 || !got.HasDataLengthIndicator || got.DataLengthIndicator != 2 {
		t.Fatalf("encrypted frame header data lost: %+v", got)
	}
	if !bytes.Equal(got.Data, []byte{0xAA, 0xBB}) {
		t.Fatalf("encrypted payload mismatch: %x", got.Data)
	}
	if got.Registration == nil || got.Registration.Owner != "https://example.test/encr" {
		t.Fatalf("ENCR registration not linked: %+v", got.Registration)
	}
}

func TestID3v2_ExtendedHeaderV23CRCRoundTrip(t *testing.T) {
	tag := id3v2.NewTag(3)
	tag.ExtendedHeader = id3v2.ExtendedHeader{
		Present:    true,
		CRCPresent: true,
	}
	tag.SetText(id3v2.FrameTitle, "first title")
	first := v2Decode(t, tag)
	if !first.ExtendedHeader.Present || !first.ExtendedHeader.CRCPresent || first.ExtendedHeader.CRC == 0 {
		t.Fatalf("v2.3 extended header lost: %+v", first.ExtendedHeader)
	}
	firstCRC := first.ExtendedHeader.CRC
	first.SetText(id3v2.FrameTitle, "second title that changes the crc")
	second := v2Decode(t, first)
	if !second.ExtendedHeader.CRCPresent {
		t.Fatalf("v2.3 extended header CRC flag dropped")
	}
	if second.ExtendedHeader.CRC == firstCRC {
		t.Fatalf("v2.3 extended-header CRC was not regenerated")
	}
}

func TestID3v2_ExtendedHeaderV24RestrictionsRoundTrip(t *testing.T) {
	tag := id3v2.NewTag(4)
	tag.ExtendedHeader = id3v2.ExtendedHeader{
		Present:             true,
		Update:              true,
		CRCPresent:          true,
		RestrictionsPresent: true,
		Restrictions:        0xB5,
	}
	tag.SetText(id3v2.FrameTitle, "restricted")
	got := v2Decode(t, tag)
	if !got.ExtendedHeader.Present || !got.ExtendedHeader.Update || !got.ExtendedHeader.CRCPresent {
		t.Fatalf("v2.4 extended header flags lost: %+v", got.ExtendedHeader)
	}
	if !got.ExtendedHeader.RestrictionsPresent || got.ExtendedHeader.Restrictions != 0xB5 {
		t.Fatalf("v2.4 restrictions lost: %+v", got.ExtendedHeader)
	}
}

func TestID3v2_PlayCountRoundTrip(t *testing.T) {
	cases := []uint64{0, 1, 255, 1<<24 - 1, 1<<32 - 1, 1 << 40}
	for _, n := range cases {
		tag := id3v2.NewTag(4)
		tag.Set(&id3v2.PlayCountFrame{Count: n})
		got := v2Decode(t, tag).Find(id3v2.FramePlayCount).(*id3v2.PlayCountFrame)
		if got.Count != n {
			t.Errorf("PCNT round-trip %d → %d", n, got.Count)
		}
	}
}

func TestID3v2_PopularimeterRoundTrip(t *testing.T) {
	in := &id3v2.PopularimeterFrame{Email: "user@example.com", Rating: 200, Count: 42}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FramePopularimeter).(*id3v2.PopularimeterFrame)
	if got.Email != in.Email || got.Rating != in.Rating || got.Count != in.Count {
		t.Errorf("POPM round-trip mismatch: %+v vs %+v", got, in)
	}
}

func TestID3v2_ChapterRoundTrip(t *testing.T) {
	in := &id3v2.ChapterFrame{
		ElementID:   "ch01",
		StartTimeMs: 0,
		EndTimeMs:   60_000,
		StartOffset: 0xFFFFFFFF,
		EndOffset:   0xFFFFFFFF,
		SubFrames: []id3v2.Frame{
			&id3v2.TextFrame{FrameID: id3v2.FrameTitle, Values: []string{"Intro"}},
		},
	}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FrameChapter).(*id3v2.ChapterFrame)
	if got.ElementID != "ch01" || got.StartTimeMs != 0 || got.EndTimeMs != 60_000 {
		t.Errorf("CHAP header round-trip: %+v", got)
	}
	if len(got.SubFrames) != 1 {
		t.Fatalf("expected 1 sub-frame, got %d", len(got.SubFrames))
	}
	if tf, ok := got.SubFrames[0].(*id3v2.TextFrame); !ok || tf.Values[0] != "Intro" {
		t.Errorf("sub-frame mismatch: %+v", got.SubFrames[0])
	}
}

func TestID3v2_TOCRoundTrip(t *testing.T) {
	in := &id3v2.TOCFrame{
		ElementID: "toc",
		TopLevel:  true,
		Ordered:   true,
		ChildIDs:  []string{"ch01", "ch02", "ch03"},
		SubFrames: []id3v2.Frame{
			&id3v2.TextFrame{FrameID: id3v2.FrameTitle, Values: []string{"Episode"}},
		},
	}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FrameTOC).(*id3v2.TOCFrame)
	if !got.TopLevel || !got.Ordered {
		t.Errorf("CTOC flags lost: %+v", got)
	}
	if len(got.ChildIDs) != 3 || got.ChildIDs[1] != "ch02" {
		t.Errorf("CTOC children: %v", got.ChildIDs)
	}
	if len(got.SubFrames) != 1 {
		t.Errorf("CTOC sub-frames: %v", got.SubFrames)
	}
}

// Bit positions reproduced verbatim from id3v2.4.0-structure §4.1
// so the wire-format constants are searchable / self-documenting
// inside the test rather than appearing as raw 0x.. literals.
const (
	v24FlagCompressed   = 0x08 // §4.1.2.k — frame format flag "k"
	v24FlagDLI          = 0x01 // §4.1.2.p — frame format flag "p"
	v24TagMajorRev      = 0x04 // §3.1     — major version byte for v2.4
	v24TagRevision      = 0x00 // §3.1     — revision byte
	v24TagFlagsNoExtras = 0x00 // §3.1     — header flags: no unsync, ext
	//                                       header, experimental, footer
)

// TestID3v2_CompressedFrameRoundTrip manually assembles a zlib-
// compressed TIT2 frame and verifies the parser inflates it back
// transparently. Every literal in the byte-level construction is
// derived from id3v2.4.0-structure; the named constants above
// document which subsection each value comes from so a future
// reader doesn't need to consult the spec to decode the test.
func TestID3v2_CompressedFrameRoundTrip(t *testing.T) {
	const want = "Compressed"
	// TIT2 body = encoding byte (ISO-8859-1) + ASCII title.
	titleBody := append([]byte{byte(id3v2.EncISO8859)}, []byte(want)...)

	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	zw.Write(titleBody)
	zw.Close()
	compressed := buf.Bytes()

	// Compressed v2.4 frames carry a 4-byte synchsafe data-length
	// indicator (the body's *uncompressed* size) before the
	// compressed payload — see §4.1.2.p.
	var dli [4]byte
	if err := id3v2.EncodeSynchsafe(dli[:], uint32(len(titleBody))); err != nil {
		t.Fatal(err)
	}
	frameBody := append(dli[:], compressed...)

	// Frame header: 4-byte ID + 4-byte synchsafe size + 2-byte
	// flags (status byte = 0, format byte = compressed | DLI).
	var frame bytes.Buffer
	frame.WriteString(id3v2.FrameTitle)
	var sz [4]byte
	if err := id3v2.EncodeSynchsafe(sz[:], uint32(len(frameBody))); err != nil {
		t.Fatal(err)
	}
	frame.Write(sz[:])
	frame.Write([]byte{0x00, v24FlagCompressed | v24FlagDLI})
	frame.Write(frameBody)

	// Tag header: "ID3" magic + major + revision + flags + size.
	var tag bytes.Buffer
	tag.WriteString("ID3")
	tag.Write([]byte{v24TagMajorRev, v24TagRevision, v24TagFlagsNoExtras})
	if err := id3v2.EncodeSynchsafe(sz[:], uint32(frame.Len())); err != nil {
		t.Fatal(err)
	}
	tag.Write(sz[:])
	tag.Write(frame.Bytes())

	parsed, err := id3v2.Read(id3v2BytesReader(tag.Bytes()), 0)
	if err != nil {
		t.Fatalf("read compressed tag: %v", err)
	}
	if got := parsed.Text(id3v2.FrameTitle); got != want {
		t.Errorf("compressed title = %q, want %q", got, want)
	}
}

// TestID3v2_CompressedFrameReEncode verifies that a frame read with
// its Compressed flag set comes out of Encode still compressed (not
// silently inflated). The serialise path zlib-compresses the body
// and emits the 4-byte length indicator per version convention.
func TestID3v2_CompressedFrameReEncode(t *testing.T) {
	// Start from the inflated body the typed parser produces.
	const titleText = "CompressedReEncode"
	tag := id3v2.NewTag(4)
	tf := &id3v2.TextFrame{FrameID: id3v2.FrameTitle, Values: []string{titleText}}
	tf.FrameFlags = id3v2.FrameFlags{
		Version: 4,
		Raw:     [2]byte{0x00, v24FlagCompressed | v24FlagDLI},
	}
	tag.Set(tf)

	encoded, err := tag.Encode(0)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// Re-read through the normal parser and verify the text still
	// round-trips — this exercises both re-compression and DLI
	// prefix decoding.
	parsed, err := id3v2.Read(id3v2BytesReader(encoded), 0)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if got := parsed.Text(id3v2.FrameTitle); got != titleText {
		t.Errorf("title round-trip = %q, want %q", got, titleText)
	}
	// Confirm the on-disk frame still carries the compressed flag
	// byte so downstream tooling can tell the body is zlib'd.
	got := parsed.Find(id3v2.FrameTitle)
	if got == nil {
		t.Fatal("TIT2 missing after re-encode")
	}
	if !got.Flags().Compressed() {
		t.Errorf("Compressed flag dropped on re-encode")
	}
}

func TestID3v2_LINKRoundTrip(t *testing.T) {
	in := &id3v2.LinkedInfoFrame{
		FrameIdentifier: "TIT2",
		URL:             "http://example.com/tags",
		IDData:          []byte("arbitrary"),
	}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FrameLINK).(*id3v2.LinkedInfoFrame)
	if got.FrameIdentifier != in.FrameIdentifier || got.URL != in.URL ||
		string(got.IDData) != string(in.IDData) {
		t.Errorf("LINK round-trip: %+v vs %+v", got, in)
	}
}

func TestID3v2_COMRRoundTrip(t *testing.T) {
	in := &id3v2.CommercialFrame{
		Prices:       "USD9.99/EUR9.00",
		ValidUntil:   "20301231",
		ContactURL:   "http://shop.example",
		ReceivedAs:   2,
		NameOfSeller: "ACME Records",
		Description:  "High-quality download",
	}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FrameCOMR).(*id3v2.CommercialFrame)
	if got.Prices != in.Prices || got.ValidUntil != in.ValidUntil ||
		got.ContactURL != in.ContactURL || got.ReceivedAs != in.ReceivedAs ||
		got.NameOfSeller != in.NameOfSeller || got.Description != in.Description {
		t.Errorf("COMR round-trip: %+v vs %+v", got, in)
	}
}

func TestID3v2_POSSRoundTrip(t *testing.T) {
	in := &id3v2.PositionSyncFrame{
		Format:   id3v2.SylTimestampMillis,
		Position: 90_000,
	}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FramePOSS).(*id3v2.PositionSyncFrame)
	if got.Format != in.Format || got.Position != in.Position {
		t.Errorf("POSS round-trip: %+v vs %+v", got, in)
	}
}

func TestID3v2_GRIDRoundTrip(t *testing.T) {
	in := &id3v2.GroupRegistrationFrame{
		Owner:       "test@example",
		GroupSymbol: 0xC0,
		GroupData:   []byte{1, 2, 3, 4},
	}
	tag := id3v2.NewTag(4)
	tag.Set(in)
	got := v2Decode(t, tag).Find(id3v2.FrameGRID).(*id3v2.GroupRegistrationFrame)
	if got.Owner != in.Owner || got.GroupSymbol != in.GroupSymbol ||
		string(got.GroupData) != string(in.GroupData) {
		t.Errorf("GRID round-trip: %+v vs %+v", got, in)
	}
}

// FuzzID3v2Read throws random bytes at the v2 parser. The parser
// is allowed to return any error but must never panic.
func FuzzID3v2Read(f *testing.F) {
	tag := id3v2.NewTag(4)
	tag.SetText(id3v2.FrameTitle, "seed")
	tag.SetText(id3v2.FrameArtist, "fuzzer")
	if raw, err := tag.Encode(64); err == nil {
		f.Add(raw)
	}
	f.Add([]byte("ID3\x04\x00\x00\x00\x00\x00\x00"))
	f.Add([]byte("ID3\x03\x00\x00\x7f\x7f\x7f\x7f"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < id3v2.HeaderSize {
			return
		}
		_, _ = id3v2.Read(id3v2BytesReader(data), 0)
	})
}
