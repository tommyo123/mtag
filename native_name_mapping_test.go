package mtag

import "testing"

func TestNativeWriteKeyNames_MP4EncodedByUsesEncAtom(t *testing.T) {
	f := testFileForKind(ContainerMP4)

	f.SetEncodedBy("Encoded By")

	if f.mp4 == nil {
		t.Fatal("mp4 view = nil")
	}
	if len(f.mp4.items) != 1 {
		t.Fatalf("mp4 item count = %d, want 1", len(f.mp4.items))
	}
	if got := string(f.mp4.items[0].Name[:]); got != "\xa9enc" {
		t.Fatalf("mp4 item name = %q, want %q", got, "\xa9enc")
	}
}

func TestNativeWriteKeyNames_MP4PublisherDoesNotInventBogusILSTAtom(t *testing.T) {
	f := testFileForKind(ContainerMP4)

	f.SetPublisher("Publisher")

	if f.mp4 == nil {
		t.Fatal("mp4 view = nil")
	}
	if len(f.mp4.items) != 0 {
		t.Fatalf("mp4 item count = %d, want 0", len(f.mp4.items))
	}
	if err := f.Err(); err == nil {
		t.Fatal("Err() = nil, want recoverable write error")
	}
}

func TestNativeWriteKeyNames_RIFFUsesCanonicalINFOKeys(t *testing.T) {
	f := testFileForKind(ContainerW64)

	f.SetTrack(2, 4)
	f.SetComposer("Composer")
	f.SetPublisher("Publisher")
	f.SetEncodedBy("Encoded By")

	if f.riffInfo == nil {
		t.Fatal("riffInfo = nil")
	}
	if got := f.riffInfo.Get(riffIPRT); got != "2/4" {
		t.Fatalf("IPRT = %q, want %q", got, "2/4")
	}
	if got := f.riffInfo.Get(riffIMUS); got != "Composer" {
		t.Fatalf("IMUS = %q, want %q", got, "Composer")
	}
	if got := f.riffInfo.Get(riffIPUB); got != "Publisher" {
		t.Fatalf("IPUB = %q, want %q", got, "Publisher")
	}
	if got := f.riffInfo.Get(riffITCH); got != "Encoded By" {
		t.Fatalf("ITCH = %q, want %q", got, "Encoded By")
	}
	if got := f.riffInfo.Get(riffITRK); got != "" {
		t.Fatalf("ITRK = %q, want empty", got)
	}
	if got := f.riffInfo.Get(riffIENG); got != "" {
		t.Fatalf("IENG = %q, want empty", got)
	}
}

func TestNativeWriteKeyNames_ASFTrackAndDiscUseOfficialFields(t *testing.T) {
	f := testFileForKind(ContainerASF)

	f.SetTrack(2, 4)
	f.SetDisc(3, 5)
	f.SetLyrics("Lyrics")

	if f.asf == nil {
		t.Fatal("asf view = nil")
	}
	if got := f.asf.Get("WM/TrackNumber"); got != "2/4" {
		t.Fatalf("WM/TrackNumber = %q, want %q", got, "2/4")
	}
	if got := f.asf.Get("WM/PartOfSet"); got != "3/5" {
		t.Fatalf("WM/PartOfSet = %q, want %q", got, "3/5")
	}
	if got := f.asf.Get("WM/Lyrics"); got != "Lyrics" {
		t.Fatalf("WM/Lyrics = %q, want %q", got, "Lyrics")
	}
	if got := f.asf.Get("TotalTracks"); got != "" {
		t.Fatalf("TotalTracks = %q, want empty", got)
	}
}

func TestNativeWriteKeyNames_MatroskaEncodedByUsesOfficialTag(t *testing.T) {
	f := testFileForKind(ContainerMatroska)

	f.SetEncodedBy("Encoded By")

	if f.mkv == nil {
		t.Fatal("matroska view = nil")
	}
	if got := f.mkv.Get("ENCODED_BY"); got != "Encoded By" {
		t.Fatalf("ENCODED_BY = %q, want %q", got, "Encoded By")
	}
	if got := f.mkv.Get("ENCODER"); got != "" {
		t.Fatalf("ENCODER = %q, want empty", got)
	}
}
