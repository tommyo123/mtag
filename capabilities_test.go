package mtag

import "testing"

type stubWritableSource struct{}

func (stubWritableSource) ReadAt(p []byte, off int64) (int, error)  { return 0, nil }
func (stubWritableSource) WriteAt(p []byte, off int64) (int, error) { return len(p), nil }
func (stubWritableSource) Truncate(size int64) error                { return nil }

func testContainer(kind ContainerKind) Container {
	info := containerInfo{kind: kind}
	switch kind {
	case ContainerMP3:
		return &mp3Container{containerInfo: info}
	case ContainerWAV:
		return &wavContainer{containerInfo: info}
	case ContainerAIFF:
		return &aiffContainer{containerInfo: info}
	case ContainerFLAC:
		return &flacContainer{containerInfo: info}
	case ContainerMP4:
		return &mp4Container{containerInfo: info}
	case ContainerOGG:
		return &oggContainer{containerInfo: info}
	case ContainerMAC:
		return &macContainer{containerInfo: info}
	case ContainerAAC:
		return &aacContainer{containerInfo: info}
	case ContainerAC3:
		return &ac3Container{containerInfo: info}
	case ContainerDTS:
		return &dtsContainer{containerInfo: info}
	case ContainerAMR:
		return &amrContainer{containerInfo: info}
	case ContainerWavPack:
		return &wavPackContainer{containerInfo: info}
	case ContainerMPC:
		return &mpcContainer{containerInfo: info}
	case ContainerASF:
		return &asfContainer{containerInfo: info}
	case ContainerDSF:
		return &dsfContainer{containerInfo: info}
	case ContainerDFF:
		return &dffContainer{containerInfo: info}
	case ContainerMatroska:
		return &matroskaContainer{containerInfo: info}
	case ContainerTTA:
		return &ttaContainer{containerInfo: info}
	case ContainerMOD, ContainerS3M, ContainerXM, ContainerIT:
		return &trackerContainer{containerInfo: info}
	case ContainerRealMedia:
		return &realMediaContainer{containerInfo: info}
	case ContainerCAF:
		return &cafContainer{containerInfo: info}
	case ContainerOMA:
		return &omaContainer{containerInfo: info}
	default:
		return &mp3Container{containerInfo: info}
	}
}

func testFileForKind(kind ContainerKind) *File {
	return &File{
		sourceState: sourceState{rw: stubWritableSource{}},
		nativeState: nativeState{
			container: testContainer(kind),
		},
	}
}

func TestCapabilities_MP3(t *testing.T) {
	f := testFileForKind(ContainerMP3)
	caps := f.Capabilities()

	if !caps.CanSave {
		t.Fatal("CanSave = false, want true")
	}
	if !caps.Images.Write || !caps.Lyrics.Write || !caps.CustomFields.Write || !caps.ReplayGain.Write || !caps.MusicBrainzIDs.Write {
		t.Fatalf("mp3 write capabilities incomplete: %+v", caps)
	}
	if !caps.Chapters.Read || !caps.Chapters.Write {
		t.Fatalf("mp3 chapters capability = %+v", caps.Chapters)
	}
	if caps.NativeAttachments.Read || caps.NativeAttachments.Write {
		t.Fatalf("mp3 native attachments = %+v, want none", caps.NativeAttachments)
	}
}

func TestCapabilities_Matroska(t *testing.T) {
	f := testFileForKind(ContainerMatroska)
	caps := f.Capabilities()

	if !caps.Images.Read || !caps.Images.Write {
		t.Fatalf("matroska images = %+v", caps.Images)
	}
	if !caps.Lyrics.Read || !caps.Lyrics.Write {
		t.Fatalf("matroska lyrics = %+v", caps.Lyrics)
	}
	if !caps.Chapters.Read || !caps.Chapters.Write {
		t.Fatalf("matroska chapters = %+v", caps.Chapters)
	}
	if !caps.CustomFields.Read || !caps.CustomFields.Write {
		t.Fatalf("matroska custom fields = %+v", caps.CustomFields)
	}
	if !caps.NativeAttachments.Read || caps.NativeAttachments.Write {
		t.Fatalf("matroska native attachments = %+v", caps.NativeAttachments)
	}
}

func TestCapabilities_RealMedia(t *testing.T) {
	f := testFileForKind(ContainerRealMedia)
	caps := f.Capabilities()

	if !caps.CustomFields.Read || caps.CustomFields.Write {
		t.Fatalf("realmedia custom fields = %+v", caps.CustomFields)
	}
	if caps.Images.Read || caps.Images.Write {
		t.Fatalf("realmedia images = %+v", caps.Images)
	}
	if caps.Lyrics.Write || caps.ReplayGain.Write || caps.MusicBrainzIDs.Write {
		t.Fatalf("realmedia write capabilities unexpectedly enabled: %+v", caps)
	}
}

func TestUnsupportedID3FallbackRecordsError(t *testing.T) {
	f := testFileForKind(ContainerRealMedia)

	f.SetLyrics("text")
	if f.v2 != nil {
		t.Fatal("SetLyrics created an ID3v2 tag for RealMedia")
	}
	if f.Err() == nil {
		t.Fatal("SetLyrics did not record a recoverable error")
	}

	f.ResetErr()
	f.SetCustomValues("example", "value")
	if f.v2 != nil {
		t.Fatal("SetCustomValues created an ID3v2 tag for RealMedia")
	}
	if f.Err() == nil {
		t.Fatal("SetCustomValues did not record a recoverable error")
	}
}

func TestSetCompilationUsesMP4Store(t *testing.T) {
	f := testFileForKind(ContainerMP4)

	f.SetCompilation(true)
	if f.v2 != nil {
		t.Fatal("SetCompilation created an ID3v2 tag for MP4")
	}
	if f.mp4 == nil || !f.mp4IsCompilation() {
		t.Fatal("SetCompilation did not write MP4 cpil")
	}

	f.SetCompilation(false)
	if f.mp4IsCompilation() {
		t.Fatal("SetCompilation(false) did not clear MP4 cpil")
	}
}

func TestContainerUnknownOnZeroFile(t *testing.T) {
	var f File
	if got := f.Container(); got != ContainerUnknown {
		t.Fatalf("Container() = %v, want %v", got, ContainerUnknown)
	}
	caps := f.Capabilities()
	if caps.Container != ContainerUnknown {
		t.Fatalf("Capabilities.Container = %v, want %v", caps.Container, ContainerUnknown)
	}
	if caps.CanSave {
		t.Fatal("Capabilities.CanSave = true on zero File")
	}
}

func TestCloseClearsWritableState(t *testing.T) {
	f := &File{
		sourceState: sourceState{
			src:      stubWritableSource{},
			rw:       stubWritableSource{},
			openedRW: true,
		},
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if f.Writable() {
		t.Fatal("Writable() = true after Close")
	}
	if f.src != nil || f.rw != nil || f.fd != nil || f.openedRW {
		t.Fatalf("source handles not cleared: %+v", f.sourceState)
	}
}

func TestSaveOnUninitializedFile(t *testing.T) {
	f := &File{
		sourceState: sourceState{
			src:      stubWritableSource{},
			rw:       stubWritableSource{},
			openedRW: true,
		},
	}
	if err := f.Save(); err == nil {
		t.Fatal("Save() on uninitialized file returned nil error")
	}
}

func TestMappedTextFields_Matroska(t *testing.T) {
	f := testFileForKind(ContainerMatroska)

	f.SetPublisher("Publisher")
	f.SetCopyright("(c)")
	f.SetEncodedBy("Encoder")

	if got := f.Publisher(); got != "Publisher" {
		t.Fatalf("Publisher() = %q", got)
	}
	if got := f.Copyright(); got != "(c)" {
		t.Fatalf("Copyright() = %q", got)
	}
	if got := f.EncodedBy(); got != "Encoder" {
		t.Fatalf("EncodedBy() = %q", got)
	}
	if f.mkv == nil {
		t.Fatal("matroska store not created")
	}
	if got := f.mkv.Get("PUBLISHER"); got != "Publisher" {
		t.Fatalf("Matroska PUBLISHER = %q", got)
	}
	if got := f.mkv.Get("ENCODED_BY"); got != "Encoder" {
		t.Fatalf("Matroska ENCODED_BY = %q", got)
	}
}

func TestMappedTextFields_ASF(t *testing.T) {
	f := testFileForKind(ContainerASF)

	f.SetAlbumArtist("Band")
	f.SetGenre("Rock")
	f.SetComment("Comment")

	if f.asf == nil {
		t.Fatal("ASF store not created")
	}
	if got := f.asf.Get("WM/AlbumArtist"); got != "Band" {
		t.Fatalf("ASF album artist = %q", got)
	}
	if got := f.asf.Get("WM/Genre"); got != "Rock" {
		t.Fatalf("ASF genre = %q", got)
	}
	if got := f.asf.Get("Description"); got != "Comment" {
		t.Fatalf("ASF comment = %q", got)
	}
}
