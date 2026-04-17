package mtag

import (
	"testing"

	"github.com/tommyo123/mtag/id3v2"
)

func TestSetRatingClearPreservesPopularimeterCount(t *testing.T) {
	f := testFileForKind(ContainerMP3)
	f.ensureV2()
	f.v2.Set(&id3v2.PopularimeterFrame{
		Email:  "user@example.com",
		Rating: 180,
		Count:  42,
	})

	f.SetRating("", 0)

	if got := f.Rating(); got != 0 {
		t.Fatalf("Rating() = %d, want 0", got)
	}
	if got := f.PlayCount(); got != 42 {
		t.Fatalf("PlayCount() = %d, want 42", got)
	}
	pop, ok := f.v2.Find(id3v2.FramePopularimeter).(*id3v2.PopularimeterFrame)
	if !ok || pop == nil {
		t.Fatal("POPM frame missing")
	}
	if pop.Email != "user@example.com" {
		t.Fatalf("POPM.Email = %q, want %q", pop.Email, "user@example.com")
	}
	if pop.Rating != 0 {
		t.Fatalf("POPM.Rating = %d, want 0", pop.Rating)
	}
	if pop.Count != 42 {
		t.Fatalf("POPM.Count = %d, want 42", pop.Count)
	}
}
