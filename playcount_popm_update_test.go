package mtag

import (
	"testing"

	"github.com/tommyo123/mtag/id3v2"
)

func TestSetPlayCountUpdatesPopularimeterCount(t *testing.T) {
	f := testFileForKind(ContainerMP3)
	f.ensureV2()
	f.v2.Set(&id3v2.PopularimeterFrame{
		Email:  "user@example.com",
		Rating: 200,
		Count:  42,
	})

	f.SetPlayCount(7)

	if got := f.PlayCount(); got != 7 {
		t.Fatalf("PlayCount() = %d, want 7", got)
	}
	pop, ok := f.v2.Find(id3v2.FramePopularimeter).(*id3v2.PopularimeterFrame)
	if !ok || pop == nil {
		t.Fatal("POPM frame missing")
	}
	if pop.Count != 7 {
		t.Fatalf("POPM.Count = %d, want 7", pop.Count)
	}
	if pop.Rating != 200 {
		t.Fatalf("POPM.Rating = %d, want 200", pop.Rating)
	}
	pc, ok := f.v2.Find(id3v2.FramePlayCount).(*id3v2.PlayCountFrame)
	if !ok || pc == nil {
		t.Fatal("PCNT frame missing")
	}
	if pc.Count != 7 {
		t.Fatalf("PCNT.Count = %d, want 7", pc.Count)
	}
}

func TestSetRatingWithEmptyEmailPreservesExistingPopularimeterEmail(t *testing.T) {
	f := testFileForKind(ContainerMP3)
	f.ensureV2()
	f.v2.Set(&id3v2.PopularimeterFrame{
		Email:  "user@example.com",
		Rating: 100,
		Count:  42,
	})

	f.SetRating("", 220)

	pop, ok := f.v2.Find(id3v2.FramePopularimeter).(*id3v2.PopularimeterFrame)
	if !ok || pop == nil {
		t.Fatal("POPM frame missing")
	}
	if pop.Email != "user@example.com" {
		t.Fatalf("POPM.Email = %q, want %q", pop.Email, "user@example.com")
	}
	if pop.Rating != 220 {
		t.Fatalf("POPM.Rating = %d, want 220", pop.Rating)
	}
	if pop.Count != 42 {
		t.Fatalf("POPM.Count = %d, want 42", pop.Count)
	}
}
