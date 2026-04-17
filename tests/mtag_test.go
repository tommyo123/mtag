package tests

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tommyo123/mtag"
	"github.com/tommyo123/mtag/id3v1"
	"github.com/tommyo123/mtag/id3v2"
)

// buildTestFile constructs an MP3-like file on disk consisting of a
// serialised v2 tag, a blob of fake audio, and an optional v1 footer.
// The audio deliberately contains 0xFF bytes so that accidental
// rewrite bugs show up.
func buildTestFile(t *testing.T, v2 *id3v2.Tag, v1 *id3v1.Tag, audio []byte) string {
	t.Helper()
	var buf bytes.Buffer
	if v2 != nil {
		raw, err := v2.Encode(256)
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(raw)
	}
	buf.Write(audio)
	if v1 != nil {
		enc := v1.Encode()
		buf.Write(enc[:])
	}
	path := filepath.Join(t.TempDir(), "track.mp3")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOpenReadsBothFormats(t *testing.T) {
	v2 := id3v2.NewTag(3)
	v2.SetText(id3v2.FrameTitle, "Suffragette City")
	v2.SetText(id3v2.FrameArtist, "David Bowie")
	v2.SetText(id3v2.FrameAlbum, "Ziggy Stardust")
	v2.SetText(id3v2.FrameYear, "1972")

	v1 := &id3v1.Tag{
		Title:  "Different Title",
		Artist: "Legacy Artist",
		Album:  "Legacy Album",
		Year:   "1972",
		Genre:  17,
	}
	path := buildTestFile(t, v2, v1, bytes.Repeat([]byte{0xFF, 0x00}, 1024))

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if got, want := f.Formats(), mtag.FormatID3v23|mtag.FormatID3v1; got != want {
		t.Errorf("formats = %v, want %v", got, want)
	}
	if f.Title() != "Suffragette City" {
		t.Errorf("title = %q", f.Title())
	}
	if f.Artist() != "David Bowie" {
		t.Errorf("artist = %q", f.Artist())
	}
	if f.Year() != 1972 {
		t.Errorf("year = %d", f.Year())
	}
}

func TestSaveSyncsV1FromV2(t *testing.T) {
	v2 := id3v2.NewTag(4)
	v2.SetText(id3v2.FrameTitle, "Heroes")
	v2.SetText(id3v2.FrameArtist, "Bowie")

	v1 := &id3v1.Tag{Genre: 255}
	path := buildTestFile(t, v2, v1, []byte("AUDIO-PAYLOAD"))

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.SetTitle("Heroes (Remastered)")
	f.SetArtist("David Bowie")
	f.SetYear(1977)
	f.SetTrack(2, 10)
	f.SetGenre("Rock")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.V1() == nil {
		t.Fatal("v1 lost after save")
	}
	if g.V1().Title != "Heroes (Remastered)" {
		t.Errorf("v1 title not synced: %q", g.V1().Title)
	}
	if g.V1().Year != "1977" {
		t.Errorf("v1 year not synced: %q", g.V1().Year)
	}
	if g.V1().GenreName() != "Rock" {
		t.Errorf("v1 genre not synced: %q", g.V1().GenreName())
	}
	if g.V2().Text(id3v2.FrameTitle) != "Heroes (Remastered)" {
		t.Errorf("v2 title = %q", g.V2().Text(id3v2.FrameTitle))
	}
}

func TestInPlaceWritePreservesAudio(t *testing.T) {
	v2 := id3v2.NewTag(3)
	v2.SetText(id3v2.FrameTitle, "TitleA")
	audio := bytes.Repeat([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 2048)
	path := buildTestFile(t, v2, nil, audio)

	f, _ := mtag.Open(path)
	f.SetTitle("TitleB")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	raw, _ := os.ReadFile(path)
	if !bytes.Contains(raw, audio) {
		t.Errorf("audio payload corrupted by in-place write")
	}
}

func TestGrowRewritesFile(t *testing.T) {
	v2 := id3v2.NewTag(4)
	v2.SetText(id3v2.FrameTitle, "x")
	audio := bytes.Repeat([]byte{0x12, 0x34}, 4096)
	path := buildTestFile(t, v2, nil, audio)

	f, _ := mtag.Open(path)
	big := bytes.Repeat([]byte{0xAB, 0xCD, 0xEF}, 4096)
	f.SetCoverArt("image/jpeg", big)
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if imgs := g.Images(); len(imgs) != 1 || !bytes.Equal(imgs[0].Data, big) {
		t.Errorf("cover art lost or altered after rewrite")
	}
	raw, _ := os.ReadFile(path)
	if !bytes.Contains(raw, audio) {
		t.Errorf("audio corrupted by rewrite")
	}
}

func TestSaveWithStripsV1(t *testing.T) {
	v2 := id3v2.NewTag(4)
	v2.SetText(id3v2.FrameTitle, "Ashes to Ashes")
	v1 := &id3v1.Tag{Title: "Ashes to Ashes", Genre: 17}
	path := buildTestFile(t, v2, v1, []byte("audio"))

	f, _ := mtag.Open(path)
	if err := f.SaveWith(mtag.FormatID3v24); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.V1() != nil {
		t.Errorf("v1 footer not stripped")
	}
	if g.V2() == nil {
		t.Errorf("v2 tag missing after SaveWith")
	}
}

func TestSaveWithRejectsMultipleV2Versions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.mp3")
	os.WriteFile(path, []byte("audio"), 0o644)
	f, _ := mtag.Open(path)
	defer f.Close()
	err := f.SaveWith(mtag.FormatID3v23 | mtag.FormatID3v24)
	if err == nil {
		t.Fatal("expected error for multiple v2 versions")
	}
}

func TestPictureRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fresh.mp3")
	os.WriteFile(path, []byte("AUDIO-DATA"), 0o644)

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	cover := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 1, 2, 3, 4}
	f.SetTitle("Fame")
	f.SetCoverArt("image/png", cover)
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, _ := mtag.Open(path)
	defer g.Close()
	imgs := g.Images()
	if len(imgs) != 1 {
		t.Fatalf("expected 1 image, got %d", len(imgs))
	}
	if imgs[0].MIME != "image/png" {
		t.Errorf("mime = %q", imgs[0].MIME)
	}
	if !bytes.Equal(imgs[0].Data, cover) {
		t.Errorf("cover data mismatch")
	}
	if imgs[0].Type != mtag.PictureCoverFront {
		t.Errorf("type = %v", imgs[0].Type)
	}
}

// TestSetterOnTagLessFileCreatesV2 checks that mutating a file with no
// tags persists the new values.
func TestSetterOnTagLessFileCreatesV2(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bare.mp3")
	os.WriteFile(path, []byte("RAW-AUDIO-BYTES"), 0o644)

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Formats() != 0 {
		t.Fatalf("fresh file should report no tags, got %s", f.Formats())
	}
	f.SetTitle("First Save")
	f.SetArtist("An Artist")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if !g.Formats().HasAny(mtag.FormatID3v2Any) {
		t.Errorf("expected an ID3v2 tag to have been created, got %s", g.Formats())
	}
	if g.Title() != "First Save" {
		t.Errorf("title = %q", g.Title())
	}
	if g.Artist() != "An Artist" {
		t.Errorf("artist = %q", g.Artist())
	}
}

// TestCorruptV2HeaderDoesNotLeakIntoAudio constructs a file whose
// first ten bytes look like a valid ID3v2 header but whose body is
// garbage. After editing and saving, the corrupt bytes must not
// survive anywhere in the resulting file.
func TestCorruptV2HeaderDoesNotLeakIntoAudio(t *testing.T) {
	// Valid-looking v2.3 header claiming a 64-byte body…
	header := []byte{
		'I', 'D', '3', 3, 0, 0, // magic + version + flags
		0, 0, 0, 64, // synchsafe size 64
	}
	// …whose body is neither a valid frame nor padding. Include a
	// distinctive signature we can later grep for.
	corruptBody := bytes.Repeat([]byte{0xBA, 0xAD}, 32)
	audio := bytes.Repeat([]byte{0x12, 0x34}, 2048)

	var buf bytes.Buffer
	buf.Write(header)
	buf.Write(corruptBody)
	buf.Write(audio)

	path := filepath.Join(t.TempDir(), "corrupt.mp3")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatalf("Open on corrupt v2 must still succeed (v1 recovery); got %v", err)
	}
	// The corrupt v2 did not parse, so no v2 should be visible.
	if f.V2() != nil {
		t.Errorf("corrupt tag should not be exposed")
	}
	// First setter creates a fresh v2; old corrupt bytes must be
	// replaced, not preserved.
	f.SetTitle("Clean")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	raw, _ := os.ReadFile(path)
	if bytes.Contains(raw, corruptBody) {
		t.Errorf("corrupt v2 body survived the rewrite")
	}
	if !bytes.Contains(raw, audio) {
		t.Errorf("audio payload was lost during corrupt-tag rewrite")
	}
	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.Title() != "Clean" {
		t.Errorf("title = %q", g.Title())
	}
}

// TestV1SyncTrackOverflow verifies that a track number too large for
// ID3v1 clears the footer's track byte instead of leaving a stale
// value behind.
func TestV1SyncTrackOverflow(t *testing.T) {
	v1 := &id3v1.Tag{Track: 5, Genre: 255}
	v2 := id3v2.NewTag(4)
	v2.SetText(id3v2.FrameTitle, "A")
	path := buildTestFile(t, v2, v1, []byte("audio"))

	f, _ := mtag.Open(path)
	f.SetTrack(999, 1000) // out of byte range
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, _ := mtag.Open(path)
	defer g.Close()
	if g.V1().Track != 0 {
		t.Errorf("stale v1 track: got %d, want 0", g.V1().Track)
	}
	if g.V1().HasTrack() {
		t.Errorf("HasTrack should be false after overflow")
	}
	// When track is absent, the v1 comment must use the full 30-byte
	// field rather than the 28-byte v1.1 layout.
	if g.V1().Track != 0 {
		t.Errorf("expected v1.0 layout, got v1.1 with track %d", g.V1().Track)
	}
}

// TestTagSetNilIsSafe covers the id3v2.Tag.Set nil contract.
func TestTagSetNilIsSafe(t *testing.T) {
	tag := id3v2.NewTag(4)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Set(nil) panicked: %v", r)
		}
	}()
	tag.Set(nil)
	if len(tag.Frames) != 0 {
		t.Errorf("Set(nil) added a frame: %+v", tag.Frames)
	}
}

func TestReplayGainFallsBackToRVA2(t *testing.T) {
	v2 := id3v2.NewTag(4)
	v2.SetText(id3v2.FrameTitle, "ReplayGain")
	v2.Set(&id3v2.RVA2Frame{
		Identification: "replaygain_track",
		Channels: []id3v2.RVA2Adjustment{{
			Channel:    id3v2.RVA2MasterVol,
			Adjustment: -3840, // -7.5 dB
			PeakBits:   32,
			Peak:       []byte{0x80, 0x00, 0x00, 0x00},
		}},
	})
	path := buildTestFile(t, v2, nil, []byte("audio"))

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rg, ok := f.ReplayGainTrack()
	if !ok {
		t.Fatal("ReplayGainTrack did not detect RVA2")
	}
	if math.Abs(rg.Gain-(-7.5)) > 0.01 {
		t.Fatalf("gain = %.4f, want -7.5", rg.Gain)
	}
	if math.Abs(rg.Peak-0.5) > 0.01 {
		t.Fatalf("peak = %.4f, want ~0.5", rg.Peak)
	}
}

func TestReplayGainWriteAddsRVA2(t *testing.T) {
	v2 := id3v2.NewTag(4)
	v2.SetText(id3v2.FrameTitle, "ReplayGain")
	path := buildTestFile(t, v2, nil, []byte("audio"))

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.SetReplayGainTrack(mtag.ReplayGain{Gain: -6.25, Peak: 0.75})
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	rg, ok := g.ReplayGainTrack()
	if !ok {
		t.Fatal("ReplayGainTrack missing after save")
	}
	if math.Abs(rg.Gain-(-6.25)) > 0.01 {
		t.Fatalf("gain = %.4f, want -6.25", rg.Gain)
	}
	if math.Abs(rg.Peak-0.75) > 0.01 {
		t.Fatalf("peak = %.4f, want ~0.75", rg.Peak)
	}
	if _, ok := g.V2().Find(id3v2.FrameRVA2).(*id3v2.RVA2Frame); !ok {
		t.Fatal("RVA2 frame not written")
	}
}

func TestSeekFrameFindsSecondaryTag(t *testing.T) {
	primary := id3v2.NewTag(4)
	primary.SetText(id3v2.FrameTitle, "Primary")
	secondary := id3v2.NewTag(4)
	secondary.ExtendedHeader = id3v2.ExtendedHeader{Present: true, Update: true}
	secondary.SetText(id3v2.FrameAlbum, "Supplement")

	gap := []byte("AUDIO-GAP")
	secondaryRaw, err := secondary.Encode(0)
	if err != nil {
		t.Fatal(err)
	}
	primary.Set(&id3v2.SeekFrame{Offset: uint32(len(gap))})
	primaryRaw, err := primary.Encode(0)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.Write(primaryRaw)
	buf.Write(gap)
	buf.Write(secondaryRaw)

	path := filepath.Join(t.TempDir(), "seek.mp3")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if f.Title() != "Primary" {
		t.Fatalf("primary title lost: %q", f.Title())
	}
	if f.Album() != "Supplement" {
		t.Fatalf("SEEK-linked secondary tag not merged: album=%q", f.Album())
	}
}

func TestSaveFormatsAlias(t *testing.T) {
	path := buildTestFile(t, nil, &id3v1.Tag{Title: "Legacy", Genre: 255}, []byte("audio"))
	f, err := mtag.Open(path, mtag.WithCreateV2OnV1Only())
	if err != nil {
		t.Fatal(err)
	}
	f.SetAlbumArtist("Band")
	if err := f.SaveFormats(mtag.FormatID3v24 | mtag.FormatID3v1); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if !g.Formats().Has(mtag.FormatID3v24) || !g.Formats().Has(mtag.FormatID3v1) {
		t.Fatalf("SaveFormats did not persist requested formats: %s", g.Formats())
	}
	if got := g.AlbumArtist(); got != "Band" {
		t.Fatalf("album artist = %q", got)
	}
}

func TestReplayGainPeakClampSurfacesErr(t *testing.T) {
	v2 := id3v2.NewTag(4)
	v2.SetText(id3v2.FrameTitle, "ReplayGain")
	path := buildTestFile(t, v2, nil, []byte("audio"))

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.SetReplayGainTrack(mtag.ReplayGain{Gain: -3, Peak: 1.5})
	if !strings.Contains(errString(f.Err()), "clamped to 1.0") {
		t.Fatalf("expected peak clamp warning, got %v", f.Err())
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
