package mtag

import "testing"

func TestMatroskaImageMutations(t *testing.T) {
	f := testFileForKind(ContainerMatroska)
	f.mkv = &matroskaView{
		Attachments: []Attachment{
			{Name: "cover-front.jpg", MIME: "image/jpeg", Data: []byte{1, 2, 3}},
			{Name: "notes.txt", MIME: "text/plain", Data: []byte("hello")},
		},
	}
	syncMatroskaPictures(f.mkv)
	if got := len(f.Images()); got != 1 {
		t.Fatalf("initial Images = %d, want 1", got)
	}

	f.AddImage(Picture{MIME: "image/png", Type: PictureCoverBack, Data: []byte{4, 5, 6}})
	if got := len(f.mkv.Attachments); got != 3 {
		t.Fatalf("attachments after AddImage = %d, want 3", got)
	}
	if got := len(f.Images()); got != 2 {
		t.Fatalf("images after AddImage = %d, want 2", got)
	}

	f.SetCoverArt("image/jpeg", []byte{9, 9, 9})
	imgs := f.Images()
	if len(imgs) != 2 {
		t.Fatalf("images after SetCoverArt = %d, want 2", len(imgs))
	}
	if string(imgs[0].Data) != string([]byte{4, 5, 6}) && string(imgs[1].Data) != string([]byte{4, 5, 6}) {
		t.Fatal("back cover lost after SetCoverArt")
	}

	f.RemoveImages()
	if got := len(f.Images()); got != 0 {
		t.Fatalf("images after RemoveImages = %d, want 0", got)
	}
	if got := len(f.mkv.Attachments); got != 1 {
		t.Fatalf("attachments after RemoveImages = %d, want 1", got)
	}
	if f.mkv.Attachments[0].Name != "notes.txt" {
		t.Fatalf("non-image attachment lost: %+v", f.mkv.Attachments[0])
	}
}

func TestImageSummaries(t *testing.T) {
	f := testFileForKind(ContainerMatroska)
	f.mkv = &matroskaView{
		Pictures: []Picture{
			{MIME: "image/jpeg", Type: PictureCoverFront, Data: []byte{1, 2, 3}},
			{MIME: "image/png", Type: PictureCoverBack, Data: []byte{4, 5}},
		},
	}
	got := f.ImageSummaries()
	if len(got) != 2 {
		t.Fatalf("ImageSummaries len = %d, want 2", len(got))
	}
	if got[0].MIME != "image/jpeg" || got[0].Size != 3 || got[1].Type != PictureCoverBack || got[1].Size != 2 {
		t.Fatalf("ImageSummaries = %+v", got)
	}
}
