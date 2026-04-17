package mtag

import (
	"math"
	"testing"

	"github.com/tommyo123/mtag/flac"
	"github.com/tommyo123/mtag/id3v2"
)

func userTextValuesForTest(tg *id3v2.Tag, desc string) []string {
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

func TestStructuredNativeMappings_ID3v2UsesCanonicalMusicBrainzReplayGainAndCustomFrames(t *testing.T) {
	f := testFileForKind(ContainerMP3)

	f.SetMusicBrainzID(MusicBrainzRecordingID, "rec-id")
	f.SetMusicBrainzID(MusicBrainzReleaseID, "rel-id")
	f.SetReplayGainTrack(ReplayGain{Gain: -6.5, Peak: 0.95})
	f.SetCustomValues("MOOD", "Calm", "Night")

	if f.v2 == nil {
		t.Fatal("ID3v2 tag = nil")
	}
	if got := f.ufidValue(musicBrainzUFIDOwner); got != "rec-id" {
		t.Fatalf("UFID recording id = %q, want %q", got, "rec-id")
	}
	if got := userTextValuesForTest(f.v2, "MusicBrainz Album Id"); len(got) != 1 || got[0] != "rel-id" {
		t.Fatalf("MusicBrainz Album Id = %#v, want [rel-id]", got)
	}
	if got := userTextValuesForTest(f.v2, "MUSICBRAINZ_ALBUMID"); len(got) != 0 {
		t.Fatalf("MUSICBRAINZ_ALBUMID = %#v, want empty", got)
	}
	if got := userTextValuesForTest(f.v2, "MusicBrainz Track Id"); len(got) != 0 {
		t.Fatalf("MusicBrainz Track Id TXXX = %#v, want empty because UFID is primary", got)
	}
	if got := userTextValuesForTest(f.v2, "REPLAYGAIN_TRACK_GAIN"); len(got) != 1 || got[0] != "-6.50 dB" {
		t.Fatalf("REPLAYGAIN_TRACK_GAIN = %#v, want [-6.50 dB]", got)
	}
	if got := userTextValuesForTest(f.v2, "REPLAYGAIN_TRACK_PEAK"); len(got) != 1 || got[0] != "0.950000" {
		t.Fatalf("REPLAYGAIN_TRACK_PEAK = %#v, want [0.950000]", got)
	}
	if got := userTextValuesForTest(f.v2, "MOOD"); len(got) != 2 || got[0] != "Calm" || got[1] != "Night" {
		t.Fatalf("MOOD = %#v, want [Calm Night]", got)
	}
	if fr := f.v2.Find(id3v2.FrameRVA2); fr == nil {
		t.Fatal("RVA2 frame missing for v2.4 ReplayGain write")
	}
}

func TestStructuredNativeMappings_VorbisUsesCanonicalKeys(t *testing.T) {
	f := testFileForKind(ContainerFLAC)

	f.SetMusicBrainzID(MusicBrainzReleaseType, "album;live")
	f.SetReplayGainTrack(ReplayGain{Gain: -4.25, Peak: 0.812345})
	f.SetCustomValues("x-custom", "alpha", "beta")

	if f.flac == nil || f.flac.comment == nil {
		t.Fatal("vorbis comment = nil")
	}
	if got := f.flac.comment.Get("RELEASETYPE"); got != "album;live" {
		t.Fatalf("RELEASETYPE = %q, want %q", got, "album;live")
	}
	if got := f.flac.comment.Get("MUSICBRAINZ_ALBUMTYPE"); got != "" {
		t.Fatalf("MUSICBRAINZ_ALBUMTYPE = %q, want empty", got)
	}
	if got := f.flac.comment.Get("REPLAYGAIN_TRACK_GAIN"); got != "-4.25 dB" {
		t.Fatalf("REPLAYGAIN_TRACK_GAIN = %q, want %q", got, "-4.25 dB")
	}
	if got := f.flac.comment.Get("REPLAYGAIN_TRACK_PEAK"); got != "0.812345" {
		t.Fatalf("REPLAYGAIN_TRACK_PEAK = %q, want %q", got, "0.812345")
	}
	if got := f.flac.comment.GetAll("X-CUSTOM"); len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("X-CUSTOM = %#v, want [alpha beta]", got)
	}
}

func TestStructuredNativeMappings_APEUsesCanonicalKeys(t *testing.T) {
	f := testFileForKind(ContainerMAC)

	f.SetMusicBrainzID(MusicBrainzReleaseType, "album")
	f.SetReplayGainAlbum(ReplayGain{Gain: -7.5, Peak: 0.991234})
	f.SetCustomValues("X-Custom", "left", "right")

	if f.ape == nil {
		t.Fatal("APE tag = nil")
	}
	if fld := f.ape.Find("MUSICBRAINZ_ALBUMTYPE"); fld == nil || fld.Text() != "album" {
		if fld == nil {
			t.Fatal("MUSICBRAINZ_ALBUMTYPE missing")
		}
		t.Fatalf("MUSICBRAINZ_ALBUMTYPE = %q, want %q", fld.Text(), "album")
	}
	if fld := f.ape.Find("RELEASETYPE"); fld != nil {
		t.Fatalf("RELEASETYPE = %q, want missing", fld.Text())
	}
	if fld := f.ape.Find("REPLAYGAIN_ALBUM_GAIN"); fld == nil || fld.Text() != "-7.50 dB" {
		if fld == nil {
			t.Fatal("REPLAYGAIN_ALBUM_GAIN missing")
		}
		t.Fatalf("REPLAYGAIN_ALBUM_GAIN = %q, want %q", fld.Text(), "-7.50 dB")
	}
	if fld := f.ape.Find("REPLAYGAIN_ALBUM_PEAK"); fld == nil || fld.Text() != "0.991234" {
		if fld == nil {
			t.Fatal("REPLAYGAIN_ALBUM_PEAK missing")
		}
		t.Fatalf("REPLAYGAIN_ALBUM_PEAK = %q, want %q", fld.Text(), "0.991234")
	}
	if fld := f.ape.Find("X-Custom"); fld == nil {
		t.Fatal("X-Custom missing")
	} else if got := fld.TextValues(); len(got) != 2 || got[0] != "left" || got[1] != "right" {
		t.Fatalf("X-Custom = %#v, want [left right]", got)
	}
}

func TestStructuredNativeMappings_ASFAndMatroskaUseCanonicalKeys(t *testing.T) {
	asfFile := testFileForKind(ContainerASF)
	asfFile.SetReplayGainTrack(ReplayGain{Gain: -3.75, Peak: 0.9234})
	asfFile.SetCustomValues("UserCustom1", "alpha", "beta")

	if asfFile.asf == nil {
		t.Fatal("asf view = nil")
	}
	if got := asfFile.asf.Get("REPLAYGAIN_TRACK_GAIN"); got != "-3.75 dB" {
		t.Fatalf("ASF REPLAYGAIN_TRACK_GAIN = %q, want %q", got, "-3.75 dB")
	}
	if got := asfFile.asf.Get("REPLAYGAIN_TRACK_PEAK"); got != "0.923400" {
		t.Fatalf("ASF REPLAYGAIN_TRACK_PEAK = %q, want %q", got, "0.923400")
	}
	if got := asfFile.asf.GetAll("UserCustom1"); len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("ASF UserCustom1 raw order = %#v, want [alpha beta]", got)
	}

	mkvFile := testFileForKind(ContainerMatroska)
	mkvFile.SetMusicBrainzID(MusicBrainzReleaseType, "album")
	mkvFile.SetReplayGainTrack(ReplayGain{Gain: -2.25, Peak: math.NaN()})
	mkvFile.SetCustomValues("x-custom", "one", "two")

	if mkvFile.mkv == nil {
		t.Fatal("matroska view = nil")
	}
	if got := mkvFile.mkv.Get("RELEASETYPE"); got != "album" {
		t.Fatalf("Matroska RELEASETYPE = %q, want %q", got, "album")
	}
	if got := mkvFile.mkv.Get("MUSICBRAINZ_ALBUMTYPE"); got != "" {
		t.Fatalf("Matroska MUSICBRAINZ_ALBUMTYPE = %q, want empty", got)
	}
	if got := mkvFile.mkv.Get("REPLAYGAIN_TRACK_GAIN"); got != "-2.25 dB" {
		t.Fatalf("Matroska REPLAYGAIN_TRACK_GAIN = %q, want %q", got, "-2.25 dB")
	}
	if got := mkvFile.mkv.Get("REPLAYGAIN_TRACK_PEAK"); got != "" {
		t.Fatalf("Matroska REPLAYGAIN_TRACK_PEAK = %q, want empty", got)
	}
	if got := mkvFile.mkv.GetAll("X-CUSTOM"); len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("Matroska X-CUSTOM = %#v, want [one two]", got)
	}
}

func TestStructuredNativeMappings_MusicBrainzAliasCleanupUsesCanonicalStore(t *testing.T) {
	id3File := testFileForKind(ContainerMP3)
	id3File.ensureV2()
	id3File.setUserText("MusicBrainz Track Id", "legacy-recording")
	id3File.SetMusicBrainzID(MusicBrainzRecordingID, "canonical-recording")

	if got := id3File.ufidValue(musicBrainzUFIDOwner); got != "canonical-recording" {
		t.Fatalf("UFID recording id = %q, want %q", got, "canonical-recording")
	}
	if got := userTextValuesForTest(id3File.v2, "MusicBrainz Track Id"); len(got) != 0 {
		t.Fatalf("MusicBrainz Track Id alias = %#v, want empty", got)
	}
	if got := userTextValuesForTest(id3File.v2, "MUSICBRAINZ_TRACKID"); len(got) != 0 {
		t.Fatalf("MUSICBRAINZ_TRACKID alias = %#v, want empty", got)
	}

	vorbisFile := testFileForKind(ContainerFLAC)
	vorbisFile.ensureFLAC()
	vorbisFile.flac.comment.Fields = append(vorbisFile.flac.comment.Fields, flac.Field{
		Name:  "MUSICBRAINZ_ALBUMTYPE",
		Value: "legacy-type",
	})
	vorbisFile.SetMusicBrainzID(MusicBrainzReleaseType, "album;live")

	if got := vorbisFile.flac.comment.Get("RELEASETYPE"); got != "album;live" {
		t.Fatalf("RELEASETYPE = %q, want %q", got, "album;live")
	}
	if got := vorbisFile.flac.comment.Get("MUSICBRAINZ_ALBUMTYPE"); got != "" {
		t.Fatalf("MUSICBRAINZ_ALBUMTYPE = %q, want empty", got)
	}
}

func TestStructuredNativeMappings_RemoveCustomClearsAllStoredValues(t *testing.T) {
	asfFile := testFileForKind(ContainerASF)
	asfFile.SetCustomValues("UserCustom1", "alpha", "beta")
	asfFile.RemoveCustom("UserCustom1")
	if asfFile.asf == nil {
		t.Fatal("asf view = nil")
	}
	if got := asfFile.asf.GetAll("UserCustom1"); len(got) != 0 {
		t.Fatalf("ASF custom values = %#v, want empty", got)
	}

	mkvFile := testFileForKind(ContainerMatroska)
	mkvFile.SetCustomValues("x-custom", "one", "two")
	mkvFile.RemoveCustom("x-custom")
	if mkvFile.mkv == nil {
		t.Fatal("matroska view = nil")
	}
	if got := mkvFile.mkv.GetAll("X-CUSTOM"); len(got) != 0 {
		t.Fatalf("Matroska custom values = %#v, want empty", got)
	}

	apeFile := testFileForKind(ContainerMAC)
	apeFile.SetCustomValues("X-Custom", "left", "right")
	apeFile.RemoveCustom("X-Custom")
	if apeFile.ape == nil {
		t.Fatal("APE tag = nil")
	}
	if fld := apeFile.ape.Find("X-Custom"); fld != nil {
		t.Fatalf("APE field still present: %#v", fld)
	}
}

func TestStructuredNativeMappings_ASFCustomValuesPreserveOrder(t *testing.T) {
	f := testFileForKind(ContainerASF)

	f.SetCustomValues("UserCustom1", "alpha", "beta")

	got := f.CustomValues("UserCustom1")
	if len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("CustomValues(UserCustom1) = %#v, want [alpha beta]", got)
	}
	if f.asf == nil {
		t.Fatal("asf view = nil")
	}
	raw := f.asf.GetAll("UserCustom1")
	if len(raw) != 2 || raw[0] != "alpha" || raw[1] != "beta" {
		t.Fatalf("asf.GetAll(UserCustom1) = %#v, want [alpha beta]", raw)
	}
	if got := f.asf.Get("UserCustom1"); got != "beta" {
		t.Fatalf("asf.Get(UserCustom1) = %q, want %q", got, "beta")
	}
}
