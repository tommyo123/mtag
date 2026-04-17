package mtag_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tommyo123/mtag"
)

type memoryWritableSource struct {
	data []byte
}

func (m *memoryWritableSource) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n := copy(p, m.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (m *memoryWritableSource) WriteAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, io.ErrUnexpectedEOF
	}
	end := int(off) + len(p)
	if end > len(m.data) {
		grow := make([]byte, end)
		copy(grow, m.data)
		m.data = grow
	}
	copy(m.data[int(off):end], p)
	return len(p), nil
}

func (m *memoryWritableSource) Truncate(size int64) error {
	if size < 0 {
		return io.ErrUnexpectedEOF
	}
	switch {
	case int(size) < len(m.data):
		m.data = m.data[:size]
	case int(size) > len(m.data):
		grow := make([]byte, size)
		copy(grow, m.data)
		m.data = grow
	}
	return nil
}

func exampleTempMP3() (string, func(), error) {
	dir, err := os.MkdirTemp("", "mtag-example-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	path := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}

func ExampleOpen() {
	path, cleanup, err := exampleTempMP3()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	f, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.SetTitle("Song")
	f.SetArtist("Artist")
	if err := f.Save(); err != nil {
		panic(err)
	}

	fmt.Println(f.Title(), "-", f.Artist())
	// Output: Song - Artist
}

func ExampleOpenWritableSource() {
	src := &memoryWritableSource{}

	f, err := mtag.OpenWritableSource(src, 0)
	if err != nil {
		panic(err)
	}

	f.SetTitle("In Memory")
	if err := f.Save(); err != nil {
		panic(err)
	}

	again, err := mtag.OpenSource(src, int64(len(src.data)))
	if err != nil {
		panic(err)
	}
	defer again.Close()

	fmt.Println(again.Title())
	// Output: In Memory
}

func ExampleFile_Capabilities() {
	path, cleanup, err := exampleTempMP3()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	f, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	caps := f.Capabilities()
	fmt.Println(caps.Images.Write, caps.CustomFields.Write)
	// Output: true true
}

func ExampleFile_SetCustomValues() {
	path, cleanup, err := exampleTempMP3()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	f, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.SetCustomValues("MOOD", "Calm", "Late night")
	if err := f.Save(); err != nil {
		panic(err)
	}

	fmt.Println(len(f.CustomValues("MOOD")), f.CustomValue("MOOD"))
	// Output: 2 Calm
}

func ExampleFile_SetReplayGainTrack() {
	path, cleanup, err := exampleTempMP3()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	f, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.SetReplayGainTrack(mtag.ReplayGain{Gain: -6.5, Peak: 0.98})
	if err := f.Save(); err != nil {
		panic(err)
	}

	rg, _ := f.ReplayGainTrack()
	fmt.Printf("%.1f %.2f\n", rg.Gain, rg.Peak)
	// Output: -6.5 0.98
}

func ExampleFile_SetCoverArt() {
	path, cleanup, err := exampleTempMP3()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	f, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.SetCoverArt("image/jpeg", []byte{0xFF, 0xD8, 0xFF, 0xD9})
	if err := f.Save(); err != nil {
		panic(err)
	}

	fmt.Println(len(f.Images()))
	// Output: 1
}

func ExampleFile_RemoveField() {
	path, cleanup, err := exampleTempMP3()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	f, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	f.SetTitle("Song")
	if err := f.Save(); err != nil {
		panic(err)
	}
	if err := f.RemoveField(mtag.FieldTitle); err != nil {
		panic(err)
	}
	if err := f.Save(); err != nil {
		panic(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	fmt.Println(g.Title() == "")
	// Output: true
}

func ExampleFile_RemoveTag() {
	path, cleanup, err := exampleTempMP3()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	f, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	f.SetTitle("Song")
	if err := f.Save(); err != nil {
		panic(err)
	}
	if err := f.RemoveTag(mtag.TagID3v2); err != nil {
		panic(err)
	}
	if err := f.Save(); err != nil {
		panic(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	fmt.Println(g.Tag(mtag.TagID3v2) == nil)
	// Output: true
}

func ExampleFile_ID3v2() {
	path, cleanup, err := exampleTempMP3()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	f, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.SetTitle("Song")
	if err := f.Save(); err != nil {
		panic(err)
	}

	fmt.Println(f.ID3v2().Version)
	// Output: 4
}

func ExampleFile_Err() {
	path, cleanup, err := exampleTempMP3()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	f, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	f.SetTitle("Song")
	if err := f.SaveFormats(mtag.FormatID3v1); err != nil {
		panic(err)
	}
	_ = f.Close()

	f, err = mtag.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.SetLyrics("not representable in ID3v1")
	fmt.Println(f.Err() != nil)
	// Output: true
}

func ExampleFile_SaveFormats() {
	path, cleanup, err := exampleTempMP3()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	f, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	f.SetTitle("Song")
	if err := f.SaveFormats(mtag.FormatID3v23 | mtag.FormatID3v1); err != nil {
		panic(err)
	}
	f.Close()

	g, err := mtag.Open(path)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	fmt.Println(g.ID3v2() != nil, g.ID3v1() != nil)
	// Output: true true
}
