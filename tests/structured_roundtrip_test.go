package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tommyo123/mtag"
	"github.com/tommyo123/mtag/ape"
	"github.com/tommyo123/mtag/id3v2"
)

func testUserTextValues(tg *id3v2.Tag, desc string) []string {
	if tg == nil {
		return nil
	}
	for _, fr := range tg.FindAll(id3v2.FrameUserText) {
		ut, ok := fr.(*id3v2.UserTextFrame)
		if !ok || ut.Description != desc {
			continue
		}
		return append([]string(nil), ut.Values...)
	}
	return nil
}

func TestEdge_ID3StructuredNamesRoundTripSynthetic(t *testing.T) {
	path := buildTestFile(t, nil, nil, buildTwoFrameMP3(0))

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.SetMusicBrainzID(mtag.MusicBrainzRecordingID, "rec-id")
	f.SetMusicBrainzID(mtag.MusicBrainzReleaseID, "rel-id")
	f.SetReplayGainTrack(mtag.ReplayGain{Gain: -6.5, Peak: 0.95})
	f.SetCustomValues("MOOD", "Calm", "Night")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()

	if got := g.MusicBrainzID(mtag.MusicBrainzRecordingID); got != "rec-id" {
		t.Fatalf("MusicBrainz recording id = %q, want %q", got, "rec-id")
	}
	if got := g.MusicBrainzID(mtag.MusicBrainzReleaseID); got != "rel-id" {
		t.Fatalf("MusicBrainz release id = %q, want %q", got, "rel-id")
	}
	if rg, ok := g.ReplayGainTrack(); !ok || rg.Gain != -6.5 || rg.Peak != 0.95 {
		t.Fatalf("ReplayGainTrack() = %+v ok=%v", rg, ok)
	}
	if got := g.CustomValues("MOOD"); len(got) != 2 || got[0] != "Calm" || got[1] != "Night" {
		t.Fatalf("CustomValues(MOOD) = %#v, want [Calm Night]", got)
	}
	v2 := g.ID3v2()
	if v2 == nil {
		t.Fatal("ID3v2() = nil")
	}
	if got := testUserTextValues(v2, "MusicBrainz Album Id"); len(got) != 1 || got[0] != "rel-id" {
		t.Fatalf("MusicBrainz Album Id = %#v, want [rel-id]", got)
	}
	if got := testUserTextValues(v2, "REPLAYGAIN_TRACK_GAIN"); len(got) != 1 || got[0] != "-6.50 dB" {
		t.Fatalf("REPLAYGAIN_TRACK_GAIN = %#v, want [-6.50 dB]", got)
	}
	if got := testUserTextValues(v2, "REPLAYGAIN_TRACK_PEAK"); len(got) != 1 || got[0] != "0.950000" {
		t.Fatalf("REPLAYGAIN_TRACK_PEAK = %#v, want [0.950000]", got)
	}
}

func TestEdge_FLACStructuredNamesRoundTripSynthetic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "structured.flac")
	if err := os.WriteFile(path, buildMinimalFLAC(t), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.SetMusicBrainzID(mtag.MusicBrainzReleaseType, "album;live")
	f.SetReplayGainTrack(mtag.ReplayGain{Gain: -4.25, Peak: 0.812345})
	f.SetCustomValues("x-custom", "alpha", "beta")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if got := g.MusicBrainzID(mtag.MusicBrainzReleaseType); got != "album;live" {
		t.Fatalf("MusicBrainz release type = %q, want %q", got, "album;live")
	}
	if rg, ok := g.ReplayGainTrack(); !ok || rg.Gain != -4.25 || rg.Peak != 0.812345 {
		t.Fatalf("ReplayGainTrack() = %+v ok=%v", rg, ok)
	}
	if got := g.CustomValues("x-custom"); len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("CustomValues(x-custom) = %#v, want [alpha beta]", got)
	}
	tag := g.Tag(mtag.TagVorbis)
	if tag == nil {
		t.Fatal("Tag(TagVorbis) = nil")
	}
	if got := tag.Get("RELEASETYPE"); got != "album;live" {
		t.Fatalf("RELEASETYPE = %q, want %q", got, "album;live")
	}
	if got := tag.Get("REPLAYGAIN_TRACK_GAIN"); got != "-4.25 dB" {
		t.Fatalf("REPLAYGAIN_TRACK_GAIN = %q, want %q", got, "-4.25 dB")
	}
	if got := tag.Get("X-CUSTOM"); got != "alpha" {
		t.Fatalf("X-CUSTOM = %q, want %q", got, "alpha")
	}
}

func TestEdge_APEStructuredNamesRoundTripSynthetic(t *testing.T) {
	t0 := ape.New()
	t0.Set("Title", "ape-structured")
	tagBody, err := t0.Encode()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "structured.ape")
	if err := os.WriteFile(path, buildMinimalAPEFile(t, tagBody), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.SetMusicBrainzID(mtag.MusicBrainzReleaseType, "album")
	f.SetReplayGainAlbum(mtag.ReplayGain{Gain: -7.5, Peak: 0.991234})
	f.SetCustomValues("X-Custom", "left", "right")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if got := g.MusicBrainzID(mtag.MusicBrainzReleaseType); got != "album" {
		t.Fatalf("MusicBrainz release type = %q, want %q", got, "album")
	}
	if rg, ok := g.ReplayGainAlbum(); !ok || rg.Gain != -7.5 || rg.Peak != 0.991234 {
		t.Fatalf("ReplayGainAlbum() = %+v ok=%v", rg, ok)
	}
	if got := g.CustomValues("X-Custom"); len(got) != 2 || got[0] != "left" || got[1] != "right" {
		t.Fatalf("CustomValues(X-Custom) = %#v, want [left right]", got)
	}
	tag := g.Tag(mtag.TagAPE)
	if tag == nil {
		t.Fatal("Tag(TagAPE) = nil")
	}
	if got := tag.Get("MUSICBRAINZ_ALBUMTYPE"); got != "album" {
		t.Fatalf("MUSICBRAINZ_ALBUMTYPE = %q, want %q", got, "album")
	}
	if got := tag.Get("REPLAYGAIN_ALBUM_GAIN"); got != "-7.50 dB" {
		t.Fatalf("REPLAYGAIN_ALBUM_GAIN = %q, want %q", got, "-7.50 dB")
	}
	if got := tag.Get("X-Custom"); got != "left" {
		t.Fatalf("X-Custom = %q, want %q", got, "left")
	}
}
