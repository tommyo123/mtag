package mtag

import (
	"testing"

	"github.com/tommyo123/mtag/id3v2"
)

func TestSetRatingPreservesPopularimeterCount(t *testing.T) {
	f := testFileForKind(ContainerMP3)
	f.ensureV2()
	f.v2.Set(&id3v2.PopularimeterFrame{
		Email:  "user@example.com",
		Rating: 100,
		Count:  42,
	})

	f.SetRating("user@example.com", 220)

	if got := f.Rating(); got != 220 {
		t.Fatalf("Rating() = %d, want 220", got)
	}
	if got := f.PlayCount(); got != 42 {
		t.Fatalf("PlayCount() = %d, want 42", got)
	}
	pop, ok := f.v2.Find(id3v2.FramePopularimeter).(*id3v2.PopularimeterFrame)
	if !ok || pop == nil {
		t.Fatal("POPM frame missing")
	}
	if pop.Count != 42 {
		t.Fatalf("POPM.Count = %d, want 42", pop.Count)
	}
	if pop.Rating != 220 {
		t.Fatalf("POPM.Rating = %d, want 220", pop.Rating)
	}
}
