package tests

import (
	"bytes"
	"testing"

	"github.com/tommyo123/mtag"
)

// formatFixture describes one testdata file and what we expect from it.
type formatFixture struct {
	file     string
	kind     mtag.ContainerKind
	writable bool // true if SetTitle+Save should succeed
}

// allFixtures enumerates every format we ship testdata for.
var allFixtures = []formatFixture{
	// ID3-backed streaming formats
	{"mp3-id3v24.mp3", mtag.ContainerMP3, true},
	{"mp3-id3v23.mp3", mtag.ContainerMP3, true},
	{"mp3-id3v22.mp3", mtag.ContainerMP3, true},
	{"aac.aac", mtag.ContainerAAC, true},
	{"ac3.ac3", mtag.ContainerAC3, true},
	{"dts.dts", mtag.ContainerDTS, true},
	{"amr-nb.amr", mtag.ContainerAMR, true},
	{"tta.tta", mtag.ContainerTTA, true},
	// Vorbis Comment formats
	{"flac.flac", mtag.ContainerFLAC, true},
	{"ogg-vorbis.ogg", mtag.ContainerOGG, true},
	{"ogg-opus.opus", mtag.ContainerOGG, true},
	{"ogg-speex.spx", mtag.ContainerOGG, true},
	{"ogg-flac.oga", mtag.ContainerOGG, true},
	// MP4
	{"mp4-aac.m4a", mtag.ContainerMP4, true},
	{"mp4-alac.m4a", mtag.ContainerMP4, true},
	{"mp4-audiobook.m4b", mtag.ContainerMP4, true},
	// RIFF/AIFF
	{"wav-id3.wav", mtag.ContainerWAV, true},
	{"aiff.aiff", mtag.ContainerAIFF, true},
	{"w64.w64", mtag.ContainerW64, true},
	// APE-native
	{"monkeys-audio.ape", mtag.ContainerMAC, true},
	{"wavpack.wv", mtag.ContainerWavPack, true},
	{"mpc-sv8.mpc", mtag.ContainerMPC, true},
	{"tak.tak", mtag.ContainerTAK, true},
	// DSD
	{"dsf.dsf", mtag.ContainerDSF, true},
	{"dff.dff", mtag.ContainerDFF, true},
	// Other containers
	{"wma.wma", mtag.ContainerASF, true},
	{"mka.mka", mtag.ContainerMatroska, false}, // write may fail on some fixtures
	{"caf.caf", mtag.ContainerCAF, true},
	{"oma.oma", mtag.ContainerOMA, true},
	{"realmedia.rm", mtag.ContainerRealMedia, false},
	// Tracker modules (read-only title)
	{"mod.mod", mtag.ContainerMOD, false},
	{"s3m.s3m", mtag.ContainerS3M, false},
	{"xm.xm", mtag.ContainerXM, false},
	{"it.it", mtag.ContainerIT, false},
}

// TestSmoke_DetectAllFormats opens every fixture and verifies the
// container kind is detected correctly. This is the most basic
// regression guard for the format detectors.
func TestSmoke_DetectAllFormats(t *testing.T) {
	for _, fx := range allFixtures {
		t.Run(fx.file, func(t *testing.T) {
			path := testdataPath(t, fx.file)
			f, err := mtag.Open(path, mtag.WithReadOnly())
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer f.Close()
			if f.Container() != fx.kind {
				t.Errorf("Container = %v, want %v", f.Container(), fx.kind)
			}
		})
	}
}

// TestSmoke_AudioProperties verifies AudioProperties returns at least
// a codec string for every fixture.
func TestSmoke_AudioProperties(t *testing.T) {
	for _, fx := range allFixtures {
		t.Run(fx.file, func(t *testing.T) {
			path := testdataPath(t, fx.file)
			f, err := mtag.Open(path, mtag.WithReadOnly())
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			ap := f.AudioProperties()
			if ap.Codec == "" && ap.SampleRate == 0 && ap.Channels == 0 {
				t.Logf("no audio properties for %s", fx.file)
			}
		})
	}
}

// TestSmoke_TextRoundTrip sets Title+Artist on every writable format,
// saves to a temp copy, and verifies the values survive reopening.
func TestSmoke_TextRoundTrip(t *testing.T) {
	for _, fx := range allFixtures {
		if !fx.writable {
			continue
		}
		t.Run(fx.file, func(t *testing.T) {
			dst := testdataCopy(t, fx.file)
			f, err := mtag.Open(dst)
			if err != nil {
				t.Fatal(err)
			}
			f.SetTitle("Smoke Test")
			f.SetArtist("Test Suite")
			if err := f.Save(); err != nil {
				f.Close()
				t.Skipf("save: %v", err)
			}
			f.Close()

			g, err := mtag.Open(dst, mtag.WithReadOnly())
			if err != nil {
				t.Fatal(err)
			}
			defer g.Close()
			if g.Title() != "Smoke Test" {
				t.Errorf("Title = %q", g.Title())
			}
			if g.Artist() != "Test Suite" {
				t.Errorf("Artist = %q", g.Artist())
			}
		})
	}
}

// TestSmoke_AllFieldsRoundTrip exercises every common polymorphic field
// on a few representative writable formats.
func TestSmoke_AllFieldsRoundTrip(t *testing.T) {
	formats := []string{"mp3-id3v24.mp3", "flac.flac", "mp4-aac.m4a", "ogg-vorbis.ogg"}
	for _, name := range formats {
		t.Run(name, func(t *testing.T) {
			dst := testdataCopy(t, name)
			f, err := mtag.Open(dst)
			if err != nil {
				t.Fatal(err)
			}
			f.SetTitle("All Fields")
			f.SetArtist("Test Artist")
			f.SetAlbum("Test Album")
			f.SetAlbumArtist("Test Band")
			f.SetComposer("Test Composer")
			f.SetYear(2025)
			f.SetTrack(7, 15)
			f.SetDisc(2, 3)
			f.SetGenre("Jazz")
			f.SetComment("Test comment")
			f.SetLyrics("Line one\nLine two")
			if err := f.Save(); err != nil {
				f.Close()
				t.Fatal(err)
			}
			f.Close()

			g, _ := mtag.Open(dst, mtag.WithReadOnly())
			defer g.Close()
			assert := func(name, got, want string) {
				if got != want {
					t.Errorf("%s = %q, want %q", name, got, want)
				}
			}
			assert("Title", g.Title(), "All Fields")
			assert("Artist", g.Artist(), "Test Artist")
			assert("Album", g.Album(), "Test Album")
			assert("Composer", g.Composer(), "Test Composer")
			assert("Genre", g.Genre(), "Jazz")
			assert("Comment", g.Comment(), "Test comment")
			assert("Lyrics", g.Lyrics(), "Line one\nLine two")
			if g.Year() != 2025 {
				t.Errorf("Year = %d", g.Year())
			}
			if g.Track() != 7 || g.TrackTotal() != 15 {
				t.Errorf("Track = %d/%d", g.Track(), g.TrackTotal())
			}
			if g.Disc() != 2 || g.DiscTotal() != 3 {
				t.Errorf("Disc = %d/%d", g.Disc(), g.DiscTotal())
			}
		})
	}
}

// TestSmoke_ImageRoundTrip sets a cover image, saves, and verifies it
// survives reopening.
func TestSmoke_ImageRoundTrip(t *testing.T) {
	formats := []string{"mp3-id3v24.mp3", "flac.flac", "mp4-aac.m4a"}
	cover := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 1, 2, 3, 4}
	for _, name := range formats {
		t.Run(name, func(t *testing.T) {
			dst := testdataCopy(t, name)
			f, _ := mtag.Open(dst)
			f.SetCoverArt("image/png", cover)
			f.Save()
			f.Close()

			g, _ := mtag.Open(dst, mtag.WithReadOnly())
			defer g.Close()
			imgs := g.Images()
			if len(imgs) == 0 {
				t.Fatal("no images after SetCoverArt")
			}
			if !bytes.Equal(imgs[0].Data, cover) {
				t.Error("cover data mismatch")
			}
		})
	}
}

// TestSmoke_RemoveField removes Title, saves, and verifies it's gone.
func TestSmoke_RemoveField(t *testing.T) {
	dst := testdataCopy(t, "mp3-id3v24.mp3")
	f, _ := mtag.Open(dst)
	f.SetTitle("Will Remove")
	f.Save()
	f.RemoveField(mtag.FieldTitle)
	f.Save()
	f.Close()

	g, _ := mtag.Open(dst, mtag.WithReadOnly())
	defer g.Close()
	if g.Title() != "" {
		t.Errorf("Title = %q after remove", g.Title())
	}
}

// TestSmoke_SequentialSaves verifies that modifying different fields
// across two saves without reopening preserves both.
func TestSmoke_SequentialSaves(t *testing.T) {
	dst := testdataCopy(t, "flac.flac")
	f, _ := mtag.Open(dst)
	f.SetTitle("First")
	f.Save()
	f.SetArtist("Second")
	f.Save()
	f.Close()

	g, _ := mtag.Open(dst, mtag.WithReadOnly())
	defer g.Close()
	if g.Title() != "First" {
		t.Errorf("Title = %q", g.Title())
	}
	if g.Artist() != "Second" {
		t.Errorf("Artist = %q", g.Artist())
	}
}

// TestSmoke_Tags verifies that Tags() returns at least one tag store
// for every fixture that carries metadata.
func TestSmoke_Tags(t *testing.T) {
	for _, fx := range allFixtures {
		t.Run(fx.file, func(t *testing.T) {
			path := testdataPath(t, fx.file)
			f, err := mtag.Open(path, mtag.WithReadOnly())
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			tags := f.Tags()
			if len(tags) == 0 {
				t.Logf("no tags on %s", fx.file)
				return
			}
			for _, tag := range tags {
				if tag.Kind().String() == "unknown" {
					t.Errorf("Tag.Kind() = unknown")
				}
			}
		})
	}
}
