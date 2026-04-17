package flac

import "testing"

func TestDecodeVorbisCommentWithOptionsSkipsPictureFields(t *testing.T) {
	body := EncodeVorbisComment(&VorbisComment{
		Vendor: "mtag-test",
		Fields: []Field{
			{Name: "TITLE", Value: "Song"},
			{Name: "METADATA_BLOCK_PICTURE", Value: "big-picture"},
			{Name: "COVERART", Value: "legacy-picture"},
		},
	})

	got, err := DecodeVorbisCommentWithOptions(body, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("TITLE") != "Song" {
		t.Fatalf("TITLE = %q, want %q", got.Get("TITLE"), "Song")
	}
	if got.Get("METADATA_BLOCK_PICTURE") != "" {
		t.Fatal("METADATA_BLOCK_PICTURE was not skipped")
	}
	if got.Get("COVERART") != "" {
		t.Fatal("COVERART was not skipped")
	}
	if len(got.Fields) != 1 {
		t.Fatalf("len(Fields) = %d, want 1", len(got.Fields))
	}
}
