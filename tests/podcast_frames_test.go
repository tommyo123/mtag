package tests

import (
	"testing"

	"github.com/tommyo123/mtag"
)

func TestPodcastFrameAccessors(t *testing.T) {
	src := &writableBuffer{}
	f, err := mtag.OpenWritableSource(src, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.SetPodcast(true)
	f.SetPodcastCategory("Technology")
	f.SetPodcastDescription("Weekly release notes")
	f.SetPodcastIdentifier("podcast-id-1")
	f.SetPodcastFeedURL("https://example.com/feed.xml")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.OpenBytes(src.data)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if !g.Podcast() {
		t.Fatal("Podcast() = false, want true")
	}
	if got := g.PodcastCategory(); got != "Technology" {
		t.Fatalf("PodcastCategory = %q", got)
	}
	if got := g.PodcastDescription(); got != "Weekly release notes" {
		t.Fatalf("PodcastDescription = %q", got)
	}
	if got := g.PodcastIdentifier(); got != "podcast-id-1" {
		t.Fatalf("PodcastIdentifier = %q", got)
	}
	if got := g.PodcastFeedURL(); got != "https://example.com/feed.xml" {
		t.Fatalf("PodcastFeedURL = %q", got)
	}
}

func TestPodcastFrameAccessorsRemove(t *testing.T) {
	src := &writableBuffer{}
	f, err := mtag.OpenWritableSource(src, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.SetPodcast(true)
	f.SetPodcastCategory("Technology")
	f.SetPodcastDescription("Weekly release notes")
	f.SetPodcastIdentifier("podcast-id-1")
	f.SetPodcastFeedURL("https://example.com/feed.xml")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}

	f.SetPodcast(false)
	f.SetPodcastCategory("")
	f.SetPodcastDescription("")
	f.SetPodcastIdentifier("")
	f.SetPodcastFeedURL("")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.OpenBytes(src.data)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.Podcast() || g.PodcastCategory() != "" || g.PodcastDescription() != "" || g.PodcastIdentifier() != "" || g.PodcastFeedURL() != "" {
		t.Fatalf("podcast frames survived removal")
	}
}
