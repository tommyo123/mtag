package mtag

import (
	"errors"
	"testing"

	"github.com/tommyo123/mtag/ape"
	"github.com/tommyo123/mtag/flac"
	"github.com/tommyo123/mtag/id3v1"
	"github.com/tommyo123/mtag/id3v2"
)

func TestRemoveField_ID3Stores(t *testing.T) {
	f := testFileForKind(ContainerMP3)
	f.v1 = &id3v1.Tag{
		Title:   "Title",
		Artist:  "Artist",
		Album:   "Album",
		Year:    "1999",
		Comment: "Comment",
		Genre:   17,
		Track:   7,
	}
	f.v2 = id3v2.NewTag(4)
	f.v2.SetText(id3v2.FrameTitle, "Title")
	f.v2.SetText(id3v2.FrameArtist, "Artist")
	f.v2.SetText(id3v2.FrameAlbum, "Album")
	f.v2.SetText(id3v2.FrameBand, "Band")
	f.v2.SetText(id3v2.FrameRecordingTime, "1999")
	f.v2.SetText(id3v2.FrameTrack, "7/10")
	f.v2.SetText(id3v2.FramePart, "2/3")
	f.v2.SetText(id3v2.FrameGenre, "Rock")
	f.SetComment("Comment")
	f.SetLyrics("Lyrics")
	f.SetCompilation(true)

	for _, key := range []FieldKey{
		FieldTitle,
		FieldArtist,
		FieldAlbum,
		FieldAlbumArtist,
		FieldYear,
		FieldTrack,
		FieldDisc,
		FieldGenre,
		FieldComment,
		FieldLyrics,
		FieldCompilation,
	} {
		if err := f.RemoveField(key); err != nil {
			t.Fatalf("RemoveField(%s) error = %v", key, err)
		}
	}

	if f.v1.Title != "" || f.v1.Artist != "" || f.v1.Album != "" {
		t.Fatalf("v1 text fields not cleared: %+v", f.v1)
	}
	if f.v1.Year != "" || f.v1.Comment != "" || f.v1.Track != 0 || f.v1.Genre != 255 {
		t.Fatalf("v1 auxiliary fields not cleared: %+v", f.v1)
	}
	if f.v2.Text(id3v2.FrameTitle) != "" || f.v2.Text(id3v2.FrameArtist) != "" || f.v2.Text(id3v2.FrameAlbum) != "" {
		t.Fatal("basic v2 fields survived RemoveField")
	}
	if f.v2.Text(id3v2.FrameBand) != "" || f.v2.Text(id3v2.FrameTrack) != "" || f.v2.Text(id3v2.FramePart) != "" {
		t.Fatal("paired v2 fields survived RemoveField")
	}
	if f.v2.Text(id3v2.FrameGenre) != "" || f.v2.Text(id3v2.FrameRecordingTime) != "" {
		t.Fatal("genre/year survived RemoveField")
	}
	if f.v2.Find(id3v2.FrameComment) != nil {
		t.Fatal("comment frame survived RemoveField")
	}
	if f.v2.Find(id3v2.FrameLyrics) != nil {
		t.Fatal("lyrics frame survived RemoveField")
	}
	if f.v2.Find("TCMP") != nil {
		t.Fatal("compilation flag survived RemoveField")
	}
}

func TestRemoveField_VorbisAliases(t *testing.T) {
	f := testFileForKind(ContainerOGG)
	f.flac = &flacView{comment: &flac.VorbisComment{
		Vendor: "mtag",
		Fields: []flac.Field{
			{Name: "TITLE", Value: "Song"},
			{Name: "TRACKNUMBER", Value: "2"},
			{Name: "TRACKTOTAL", Value: "10"},
			{Name: "TOTALTRACKS", Value: "10"},
			{Name: "ALBUMARTIST", Value: "Band"},
			{Name: "ALBUM ARTIST", Value: "Band"},
			{Name: "METADATA_BLOCK_PICTURE", Value: "picture"},
		},
	}}
	if err := f.RemoveField(FieldTrack); err != nil {
		t.Fatal(err)
	}
	if err := f.RemoveField(FieldAlbumArtist); err != nil {
		t.Fatal(err)
	}
	if err := f.RemoveField(FieldTitle); err != nil {
		t.Fatal(err)
	}
	if got := f.flac.comment.Get("TRACKNUMBER"); got != "" {
		t.Fatalf("TRACKNUMBER = %q", got)
	}
	if got := f.flac.comment.Get("TRACKTOTAL"); got != "" || f.flac.comment.Get("TOTALTRACKS") != "" {
		t.Fatal("track totals survived RemoveField")
	}
	if got := f.flac.comment.Get("ALBUMARTIST"); got != "" || f.flac.comment.Get("ALBUM ARTIST") != "" {
		t.Fatal("album-artist aliases survived RemoveField")
	}
	if got := f.flac.comment.Get("TITLE"); got != "" {
		t.Fatalf("TITLE = %q", got)
	}
	if got := f.flac.comment.Get("METADATA_BLOCK_PICTURE"); got == "" {
		t.Fatal("unrelated vorbis field was removed")
	}
}

func TestRemoveTag_StateTransitions(t *testing.T) {
	t.Run("id3", func(t *testing.T) {
		f := testFileForKind(ContainerMP3)
		f.v1 = &id3v1.Tag{Title: "legacy"}
		f.v2 = id3v2.NewTag(4)
		f.v2size = 128
		f.formats = FormatID3v1 | FormatID3v24

		if err := f.RemoveTag(TagID3v1); err != nil {
			t.Fatalf("RemoveTag(TagID3v1) error = %v", err)
		}
		if f.v1 != nil || f.formats.Has(FormatID3v1) {
			t.Fatal("ID3v1 state not cleared")
		}
		if err := f.RemoveTag(TagID3v2); err != nil {
			t.Fatalf("RemoveTag(TagID3v2) error = %v", err)
		}
		if f.v2 != nil || f.formats.HasAny(FormatID3v2Any) {
			t.Fatal("ID3v2 state not cleared")
		}
	})

	t.Run("ape", func(t *testing.T) {
		f := testFileForKind(ContainerMAC)
		f.ape = ape.New()
		f.apeLen = 64
		if err := f.RemoveTag(TagAPE); err != nil {
			t.Fatalf("RemoveTag(TagAPE) error = %v", err)
		}
		if f.ape != nil {
			t.Fatal("APE state not cleared")
		}
	})

	t.Run("matroska", func(t *testing.T) {
		f := testFileForKind(ContainerMatroska)
		f.mkv = &matroskaView{
			Fields:      []matroskaField{{Name: "TITLE", Value: "Song"}},
			Attachments: []Attachment{{Name: "cover.jpg", Data: []byte{1, 2, 3}}},
			Pictures:    []Picture{{MIME: "image/jpeg", Data: []byte{1, 2, 3}}},
			Chapters:    []Chapter{{Title: "Intro"}},
		}
		if err := f.RemoveTag(TagMatroska); err != nil {
			t.Fatalf("RemoveTag(TagMatroska) error = %v", err)
		}
		if len(f.mkv.Fields) != 0 {
			t.Fatal("matroska fields survived RemoveTag")
		}
		if len(f.mkv.Attachments) != 1 || len(f.mkv.Chapters) != 1 {
			t.Fatal("attachments or chapters were dropped")
		}
	})

	t.Run("caf", func(t *testing.T) {
		f := testFileForKind(ContainerCAF)
		f.caf = &cafView{
			keys:      []string{"title"},
			values:    map[string]string{"title": "Song"},
			chunkAt:   64,
			chunkSize: 24,
		}
		if err := f.RemoveTag(TagCAF); err != nil {
			t.Fatalf("RemoveTag(TagCAF) error = %v", err)
		}
		if f.caf == nil || !f.caf.dirty || len(f.caf.keys) != 0 {
			t.Fatalf("CAF deletion state = %+v", f.caf)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		f := testFileForKind(ContainerMP4)
		if err := f.RemoveTag(TagVorbis); !errors.Is(err, ErrUnsupportedOperation) {
			t.Fatalf("RemoveTag(TagVorbis) error = %v, want ErrUnsupportedOperation", err)
		}
	})
}

func TestRemoveField_UnknownKey(t *testing.T) {
	f := testFileForKind(ContainerMP3)
	if err := f.RemoveField(FieldKey(0)); !errors.Is(err, ErrUnsupportedOperation) {
		t.Fatalf("RemoveField(0) error = %v, want ErrUnsupportedOperation", err)
	}
}
