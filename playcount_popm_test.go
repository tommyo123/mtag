package mtag

import (
	"testing"

	"github.com/tommyo123/mtag/id3v2"
)

func TestSetPlayCountZeroClearsPopularimeterCount(t *testing.T) {
	f := testFileForKind(ContainerMP3)
	f.ensureV2()
	f.v2.Set(&id3v2.PopularimeterFrame{
		Email:  "user@example.com",
		Rating: 200,
		Count:  42,
	})
	f.v2.Set(&id3v2.PlayCountFrame{Count: 99})

	f.SetPlayCount(0)

	if got := f.PlayCount(); got != 0 {
		t.Fatalf("PlayCount() = %d, want 0", got)
	}
	pop, ok := f.v2.Find(id3v2.FramePopularimeter).(*id3v2.PopularimeterFrame)
	if !ok || pop == nil {
		t.Fatal("POPM frame missing")
	}
	if pop.Count != 0 {
		t.Fatalf("POPM.Count = %d, want 0", pop.Count)
	}
	if pop.Rating != 200 {
		t.Fatalf("POPM.Rating = %d, want 200", pop.Rating)
	}
	if f.v2.Find(id3v2.FramePlayCount) != nil {
		t.Fatal("PCNT frame still present")
	}
}
