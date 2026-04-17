package mtag

import (
	"bytes"
	"errors"
	"testing"

	"github.com/tommyo123/mtag/flac"
)

func TestDetectOGGRecordsParseError(t *testing.T) {
	f := testFileForKind(ContainerOGG)
	data := []byte("OggS")
	f.src = bytes.NewReader(data)
	f.size = int64(len(data))

	if err := f.detectOGG(); err != nil {
		t.Fatalf("detectOGG() error = %v, want nil", err)
	}
	if f.flac != nil {
		t.Fatal("detectOGG() populated flac view for malformed data")
	}
	if f.oggErr == nil {
		t.Fatal("detectOGG() did not record parse failure")
	}
}

func TestSaveOGGRejectsUnparsedCommentStore(t *testing.T) {
	f := testFileForKind(ContainerOGG)
	f.oggErr = errors.New("comment packet not found")

	err := f.saveOGG()
	if !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("saveOGG() error = %v, want ErrInvalidTag", err)
	}
}

func TestEnsureFLACClearsOGGParseError(t *testing.T) {
	f := testFileForKind(ContainerOGG)
	f.oggErr = errors.New("comment packet not found")

	f.ensureFLAC()

	if f.oggErr != nil {
		t.Fatalf("ensureFLAC() left oggErr = %v, want nil", f.oggErr)
	}
	if f.flac == nil || f.flac.comment == nil {
		t.Fatal("ensureFLAC() did not create comment view")
	}
}

func TestTrackStringUsesVorbisTotalFallback(t *testing.T) {
	f := testFileForKind(ContainerOGG)
	f.flac = &flacView{
		comment: &flac.VorbisComment{
			Fields: []flac.Field{
				{Name: "TRACKNUMBER", Value: "3"},
				{Name: "TOTALTRACKS", Value: "9"},
			},
		},
	}

	if got := f.trackString(); got != "3/9" {
		t.Fatalf("trackString() = %q, want %q", got, "3/9")
	}
}
