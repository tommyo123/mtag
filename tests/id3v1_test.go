package tests

import (
	"bytes"
	"testing"

	"github.com/tommyo123/mtag/id3v1"
)

// Generic placeholder values used by the round-trip tests below.
// They are not music trivia — the tests verify byte-level
// round-tripping of the id3v1 layout, so the specific text content
// is irrelevant beyond being recognisable in failure messages and
// fitting within the format's fixed-width fields.
const (
	sampleTitle   = "sample title"
	sampleArtist  = "sample artist"
	sampleAlbum   = "sample album"
	sampleComment = "sample comment"
	sampleYear    = "2024"
	sampleTrack   = 3
	// genreRock is referenced explicitly because TestID3v1_GenreTable
	// validates the canonical id↔name mapping. Other tests use it
	// just to fill the byte; any spec-defined id (0–191) would do.
	genreRock byte = 17
)

// TestID3v1_RoundTrip exercises the full Encode → Decode cycle for
// both layouts of the format. v1.1 adds a track byte that steals
// the last two bytes of the comment field; v1.0 keeps the full 30-
// byte comment and reports HasTrack()=false on read.
func TestID3v1_RoundTrip(t *testing.T) {
	cases := []struct {
		name         string
		tag          *id3v1.Tag
		wantHasTrack bool
	}{
		{
			name: "v1.1 with track byte",
			tag: &id3v1.Tag{
				Title:   sampleTitle,
				Artist:  sampleArtist,
				Album:   sampleAlbum,
				Year:    sampleYear,
				Comment: sampleComment,
				Track:   sampleTrack,
				Genre:   genreRock,
			},
			wantHasTrack: true,
		},
		{
			// 28-character comment puts non-zero bytes at offset
			// 124, which prevents the v1.1 detector from treating
			// the trailing pair as (NUL, track).
			name: "v1.0 no track byte",
			tag: &id3v1.Tag{
				Title:   sampleTitle,
				Artist:  sampleArtist,
				Album:   sampleAlbum,
				Year:    sampleYear,
				Comment: "twenty-eight char comment ok",
				Genre:   genreRock,
			},
			wantHasTrack: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := tc.tag.Encode()
			got, err := id3v1.Decode(buf[:])
			if err != nil {
				t.Fatal(err)
			}
			if got.HasTrack() != tc.wantHasTrack {
				t.Errorf("HasTrack = %v, want %v (got tag %+v)",
					got.HasTrack(), tc.wantHasTrack, got)
			}
			if got.Title != tc.tag.Title || got.Artist != tc.tag.Artist ||
				got.Album != tc.tag.Album || got.Year != tc.tag.Year ||
				got.Genre != tc.tag.Genre {
				t.Errorf("round-trip core fields mismatch:\n  got  %+v\n  want %+v", got, tc.tag)
			}
			if tc.wantHasTrack && got.Track != tc.tag.Track {
				t.Errorf("track = %d, want %d", got.Track, tc.tag.Track)
			}
		})
	}
}

func TestID3v1_MagicDetection(t *testing.T) {
	var buf [id3v1.Size]byte
	if _, err := id3v1.Decode(buf[:]); err != id3v1.ErrNotPresent {
		t.Fatalf("expected ErrNotPresent, got %v", err)
	}
	copy(buf[:3], id3v1.Magic[:])
	if _, err := id3v1.Decode(buf[:]); err != nil {
		t.Fatalf("unexpected error for empty valid tag: %v", err)
	}
}

// TestID3v1_Latin1ReplacesUnknown checks the encoder substitutes
// '?' for any rune that does not fit in ISO-8859-1, exactly as the
// id3v1 spec requires. Any non-Latin-1 string would do here; we
// pick three CJK characters because they leave the encoder no
// fallback path other than substitution.
func TestID3v1_Latin1ReplacesUnknown(t *testing.T) {
	const nonLatin1 = "日本語"
	orig := &id3v1.Tag{Title: nonLatin1, Genre: id3v1Unset}
	buf := orig.Encode()
	dec, _ := id3v1.Decode(buf[:])
	for _, c := range []byte(dec.Title) {
		if c != '?' {
			t.Fatalf("expected all '?', got %q", dec.Title)
		}
	}
}

// id3v1Unset is the genre value the spec reserves for "no genre".
const id3v1Unset byte = 255

// TestID3v1_GenreTable spot-checks the canonical id↔name mapping
// the spec defines, plus the "no match" fallback.
func TestID3v1_GenreTable(t *testing.T) {
	cases := []struct {
		id   byte
		name string
	}{
		{0, "Blues"},      // §2 — first entry of Eric Kemp's list
		{17, "Rock"},      // §2 — id 17 is "Rock"
		{147, "Synthpop"}, // last entry before the Winamp 5.6 batch
	}
	for _, tc := range cases {
		if got := id3v1.GenreName(tc.id); got != tc.name {
			t.Errorf("GenreName(%d) = %q, want %q", tc.id, got, tc.name)
		}
		if got := id3v1.GenreID(tc.name); got != tc.id {
			t.Errorf("GenreID(%q) = %d, want %d", tc.name, got, tc.id)
		}
	}
	if id3v1.GenreID("bogus-genre-not-in-the-table") != id3v1Unset {
		t.Errorf("unknown genre must map to %d (no genre)", id3v1Unset)
	}
}

func TestID3v1_EncodeWritesMagic(t *testing.T) {
	buf := (&id3v1.Tag{}).Encode()
	if !bytes.Equal(buf[:3], id3v1.Magic[:]) {
		t.Fatalf("encode did not start with TAG magic: %q", buf[:3])
	}
}

// FuzzID3v1Decode makes sure malformed 128-byte footers never panic.
func FuzzID3v1Decode(f *testing.F) {
	f.Add(make([]byte, id3v1.Size))
	valid := (&id3v1.Tag{Title: sampleTitle, Genre: id3v1Unset}).Encode()
	f.Add(valid[:])
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) != id3v1.Size {
			return
		}
		_, _ = id3v1.Decode(b)
	})
}
