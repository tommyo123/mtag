package tests

import (
	"testing"

	"github.com/tommyo123/mtag"
	"github.com/tommyo123/mtag/id3v2"
)

func TestEdge_PlayCountUpdateSyncsPopularimeterCountSynthetic(t *testing.T) {
	v2 := id3v2.NewTag(4)
	v2.SetText(id3v2.FrameTitle, "count-sync")
	v2.Set(&id3v2.PopularimeterFrame{
		Email:  "user@example.com",
		Rating: 180,
		Count:  42,
	})
	path := buildTestFile(t, v2, nil, []byte("audio"))

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.SetPlayCount(7)
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if got := g.PlayCount(); got != 7 {
		t.Fatalf("PlayCount() = %d, want 7", got)
	}
	if got := g.Rating(); got != 180 {
		t.Fatalf("Rating() = %d, want 180", got)
	}
	pop, ok := g.ID3v2().Find(id3v2.FramePopularimeter).(*id3v2.PopularimeterFrame)
	if !ok || pop == nil {
		t.Fatal("POPM frame missing")
	}
	if pop.Count != 7 {
		t.Fatalf("POPM.Count = %d, want 7", pop.Count)
	}
}
