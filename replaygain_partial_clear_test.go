package mtag

import (
	"math"
	"testing"

	"github.com/tommyo123/mtag/id3v2"
)

func TestReplayGainID3v2PartialUpdateClearsStaleFields(t *testing.T) {
	f := testFileForKind(ContainerMP3)

	f.SetReplayGainTrack(ReplayGain{Gain: -6.5, Peak: 0.95})
	f.SetReplayGainTrack(ReplayGain{Gain: math.NaN(), Peak: 0.5})

	if f.v2 == nil {
		t.Fatal("ID3v2 tag = nil")
	}
	if got := userTextValuesForTest(f.v2, "REPLAYGAIN_TRACK_GAIN"); len(got) != 0 {
		t.Fatalf("REPLAYGAIN_TRACK_GAIN = %#v, want empty", got)
	}
	if got := userTextValuesForTest(f.v2, "REPLAYGAIN_TRACK_PEAK"); len(got) != 1 || got[0] != "0.500000" {
		t.Fatalf("REPLAYGAIN_TRACK_PEAK = %#v, want [0.500000]", got)
	}
	if fr := f.v2.Find(id3v2.FrameRVA2); fr != nil {
		t.Fatalf("RVA2 frame = %#v, want nil when gain is absent", fr)
	}
}

func TestReplayGainID3v2PeakCanBeClearedWithoutDroppingGain(t *testing.T) {
	f := testFileForKind(ContainerMP3)

	f.SetReplayGainTrack(ReplayGain{Gain: -6.5, Peak: 0.95})
	f.SetReplayGainTrack(ReplayGain{Gain: -4.25, Peak: math.NaN()})

	if f.v2 == nil {
		t.Fatal("ID3v2 tag = nil")
	}
	if got := userTextValuesForTest(f.v2, "REPLAYGAIN_TRACK_GAIN"); len(got) != 1 || got[0] != "-4.25 dB" {
		t.Fatalf("REPLAYGAIN_TRACK_GAIN = %#v, want [-4.25 dB]", got)
	}
	if got := userTextValuesForTest(f.v2, "REPLAYGAIN_TRACK_PEAK"); len(got) != 0 {
		t.Fatalf("REPLAYGAIN_TRACK_PEAK = %#v, want empty", got)
	}
	fr, ok := f.v2.Find(id3v2.FrameRVA2).(*id3v2.RVA2Frame)
	if !ok {
		t.Fatal("RVA2 frame missing")
	}
	ch, ok := rva2MasterChannel(fr)
	if !ok {
		t.Fatal("RVA2 master channel missing")
	}
	if math.Abs(ch.AdjustmentDB()-(-4.25)) > 0.01 {
		t.Fatalf("RVA2 gain = %.4f, want -4.25", ch.AdjustmentDB())
	}
	if _, ok := ch.PeakRatio(); ok {
		t.Fatalf("RVA2 peak should be absent, got bits=%d peak=%v", ch.PeakBits, ch.Peak)
	}
}
