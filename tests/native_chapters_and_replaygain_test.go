package tests

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/tommyo123/mtag"
	"github.com/tommyo123/mtag/flac"
)

func buildWAVWithCueChapters() []byte {
	var body bytes.Buffer

	writeChunk := func(id string, chunk []byte) {
		body.WriteString(id)
		var size [4]byte
		binary.LittleEndian.PutUint32(size[:], uint32(len(chunk)))
		body.Write(size[:])
		body.Write(chunk)
		if len(chunk)%2 == 1 {
			body.WriteByte(0)
		}
	}

	var fmtBody [16]byte
	binary.LittleEndian.PutUint16(fmtBody[0:2], 1)
	binary.LittleEndian.PutUint16(fmtBody[2:4], 1)
	binary.LittleEndian.PutUint32(fmtBody[4:8], 44100)
	binary.LittleEndian.PutUint32(fmtBody[8:12], 88200)
	binary.LittleEndian.PutUint16(fmtBody[12:14], 2)
	binary.LittleEndian.PutUint16(fmtBody[14:16], 16)
	writeChunk("fmt ", fmtBody[:])

	var cue bytes.Buffer
	var u32 [4]byte
	binary.LittleEndian.PutUint32(u32[:], 2)
	cue.Write(u32[:])
	writeCue := func(id, sample uint32) {
		binary.LittleEndian.PutUint32(u32[:], id)
		cue.Write(u32[:])
		binary.LittleEndian.PutUint32(u32[:], 0)
		cue.Write(u32[:])
		cue.WriteString("data")
		binary.LittleEndian.PutUint32(u32[:], 0)
		cue.Write(u32[:])
		binary.LittleEndian.PutUint32(u32[:], 0)
		cue.Write(u32[:])
		binary.LittleEndian.PutUint32(u32[:], sample)
		cue.Write(u32[:])
	}
	writeCue(1, 0)
	writeCue(2, 44100)
	writeChunk("cue ", cue.Bytes())

	var adtl bytes.Buffer
	adtl.WriteString("adtl")
	writeLabel := func(id uint32, text string) {
		adtl.WriteString("labl")
		size := 4 + len(text) + 1
		binary.LittleEndian.PutUint32(u32[:], uint32(size))
		adtl.Write(u32[:])
		binary.LittleEndian.PutUint32(u32[:], id)
		adtl.Write(u32[:])
		adtl.WriteString(text)
		adtl.WriteByte(0)
		if size%2 == 1 {
			adtl.WriteByte(0)
		}
	}
	writeLabel(1, "Intro")
	writeLabel(2, "Verse")
	writeChunk("LIST", adtl.Bytes())

	writeChunk("data", make([]byte, 44100*2*2))

	var riff bytes.Buffer
	riff.WriteString("RIFF")
	var size [4]byte
	binary.LittleEndian.PutUint32(size[:], uint32(4+body.Len()))
	riff.Write(size[:])
	riff.WriteString("WAVE")
	riff.Write(body.Bytes())
	return riff.Bytes()
}

func buildMP3WithLAMEReplayGain(trackGain, albumGain, peak float64, validCRC bool) []byte {
	const frameLen = 417
	frame := make([]byte, frameLen)
	copy(frame[:4], []byte{0xFF, 0xFB, 0x90, 0x64})

	const xingAt = 36
	copy(frame[xingAt:], []byte("Xing"))
	binary.BigEndian.PutUint32(frame[xingAt+4:xingAt+8], 0x00000003)
	binary.BigEndian.PutUint32(frame[xingAt+8:xingAt+12], 100)
	binary.BigEndian.PutUint32(frame[xingAt+12:xingAt+16], 41700)

	pos := xingAt + 16
	copy(frame[pos:pos+9], []byte("LAME3.99r"))
	pos += 9
	pos += 2 // revision/vbr method + lowpass
	binary.BigEndian.PutUint32(frame[pos:pos+4], uint32(math.Round(peak*float64(1<<23))))
	pos += 4
	binary.BigEndian.PutUint16(frame[pos:pos+2], encodeMPEGLameGain(trackGain, 1))
	pos += 2
	binary.BigEndian.PutUint16(frame[pos:pos+2], encodeMPEGLameGain(albumGain, 2))
	pos += 2
	pos += 15 // flags, abr/min bitrate, delay/padding, misc, mp3 gain, preset, music length, music crc
	crc := mpegLameCRCTest(frame[:pos])
	if !validCRC {
		crc ^= 0xFFFF
	}
	binary.BigEndian.PutUint16(frame[pos:pos+2], crc)
	return frame
}

func encodeMPEGLameGain(gain float64, kind uint16) uint16 {
	value := uint16(math.Round(math.Abs(gain) * 10))
	if value > 0x01FF {
		value = 0x01FF
	}
	if gain < 0 {
		value |= 1 << 9
	}
	value |= kind << 13
	return value
}

func mpegLameCRCTest(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func TestChapters_VorbisCommentSynthetic(t *testing.T) {
	raw := buildFLACWithBlocks(t, flac.Block{
		Type: flac.BlockVorbisComment,
		Body: flac.EncodeVorbisComment(&flac.VorbisComment{
			Vendor: "mtag-test",
			Fields: []flac.Field{
				{Name: "CHAPTER001", Value: "00:00:05.000"},
				{Name: "CHAPTER001NAME", Value: "Intro"},
				{Name: "CHAPTER002", Value: "00:00:10.500"},
				{Name: "CHAPTER002NAME", Value: "Main"},
				{Name: "CHAPTER002URL", Value: "https://example.invalid/ch2"},
			},
		}),
	})
	f, err := mtag.OpenBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if !f.Capabilities().Chapters.Read {
		t.Fatal("chapters read capability = false")
	}
	chapters := f.Chapters()
	if len(chapters) != 2 {
		t.Fatalf("chapters = %d, want 2", len(chapters))
	}
	if chapters[0].ID != "1" || chapters[0].Title != "Intro" || chapters[0].Start != 5*time.Second {
		t.Fatalf("chapter[0] = %+v", chapters[0])
	}
	if chapters[0].End != 10500*time.Millisecond {
		t.Fatalf("chapter[0].End = %v", chapters[0].End)
	}
	if chapters[1].ID != "2" || chapters[1].Title != "Main" || chapters[1].URL != "https://example.invalid/ch2" {
		t.Fatalf("chapter[1] = %+v", chapters[1])
	}
}

func TestChapters_WAVCueSynthetic(t *testing.T) {
	f, err := mtag.OpenBytes(buildWAVWithCueChapters())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if f.Container() != mtag.ContainerWAV {
		t.Fatalf("container = %v, want WAV", f.Container())
	}
	chapters := f.Chapters()
	if len(chapters) != 2 {
		t.Fatalf("chapters = %d, want 2", len(chapters))
	}
	if chapters[0].ID != "1" || chapters[0].Title != "Intro" || chapters[0].Start != 0 || chapters[0].End != time.Second {
		t.Fatalf("chapter[0] = %+v", chapters[0])
	}
	if chapters[1].ID != "2" || chapters[1].Title != "Verse" || chapters[1].Start != time.Second || chapters[1].End != 2*time.Second {
		t.Fatalf("chapter[1] = %+v", chapters[1])
	}
}

func TestReplayGain_MPEGLameFallbackSynthetic(t *testing.T) {
	f, err := mtag.OpenBytes(buildMP3WithLAMEReplayGain(-6.5, 2.3, 0.95, true))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	track, ok := f.ReplayGainTrack()
	if !ok {
		t.Fatal("ReplayGainTrack() = missing")
	}
	if math.Abs(track.Gain-(-6.5)) > 0.001 {
		t.Fatalf("track gain = %v, want -6.5", track.Gain)
	}
	if math.Abs(track.Peak-0.95) > 0.001 {
		t.Fatalf("track peak = %v, want 0.95", track.Peak)
	}

	album, ok := f.ReplayGainAlbum()
	if !ok {
		t.Fatal("ReplayGainAlbum() = missing")
	}
	if math.Abs(album.Gain-2.3) > 0.001 {
		t.Fatalf("album gain = %v, want 2.3", album.Gain)
	}
	if !math.IsNaN(album.Peak) {
		t.Fatalf("album peak = %v, want NaN", album.Peak)
	}
}

func TestReplayGain_MPEGLameBadCRCIgnored(t *testing.T) {
	f, err := mtag.OpenBytes(buildMP3WithLAMEReplayGain(-6.5, 2.3, 0.95, false))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if rg, ok := f.ReplayGainTrack(); ok {
		t.Fatalf("ReplayGainTrack() = %+v, want missing", rg)
	}
	if rg, ok := f.ReplayGainAlbum(); ok {
		t.Fatalf("ReplayGainAlbum() = %+v, want missing", rg)
	}
}
