package tests

import (
	"math"
	"testing"

	"github.com/tommyo123/mtag"
)

func TestEdge_ReplayGainID3v2PartialClearRoundTripSynthetic(t *testing.T) {
	path := buildTestFile(t, nil, nil, buildTwoFrameMP3(0))

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.SetReplayGainTrack(mtag.ReplayGain{Gain: -6.5, Peak: 0.95})
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	g.SetReplayGainTrack(mtag.ReplayGain{Gain: math.NaN(), Peak: 0.5})
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	g.Close()

	h, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	rg, ok := h.ReplayGainTrack()
	if !ok {
		t.Fatal("ReplayGainTrack() = missing")
	}
	if !math.IsNaN(rg.Gain) {
		t.Fatalf("ReplayGainTrack gain = %.4f, want NaN", rg.Gain)
	}
	if math.Abs(rg.Peak-0.5) > 0.000001 {
		t.Fatalf("ReplayGainTrack peak = %.6f, want 0.5", rg.Peak)
	}
}
