package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tommyo123/mtag"
)

func TestEdge_MP4_ExplicitFreeformRoundTripSynthetic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "explicit-freeform.m4a")
	if err := os.WriteFile(path, buildMinimalMP4(nil, nil), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.SetCustomValues("----:com.example:FIELD", "alpha", "beta")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := g.CustomValues("----:com.example:FIELD"); len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		g.Close()
		t.Fatalf("CustomValues(explicit key) = %#v, want [alpha beta]", got)
	}
	if got := g.CustomValues("FIELD"); len(got) != 0 {
		g.Close()
		t.Fatalf("CustomValues(plain key) = %#v, want empty for non-iTunes mean", got)
	}
	tag := g.Tag(mtag.TagMP4)
	if tag == nil {
		g.Close()
		t.Fatal("Tag(TagMP4) = nil")
	}
	if got := tag.Get("----:com.example:FIELD"); got != "alpha" {
		g.Close()
		t.Fatalf("Tag(TagMP4).Get(explicit key) = %q, want %q", got, "alpha")
	}
	g.SetCustomValues("----:com.example:FIELD")
	if err := g.Save(); err != nil {
		g.Close()
		t.Fatal(err)
	}
	g.Close()

	h, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if got := h.CustomValues("----:com.example:FIELD"); len(got) != 0 {
		t.Fatalf("CustomValues(explicit key) after remove = %#v, want empty", got)
	}
}
