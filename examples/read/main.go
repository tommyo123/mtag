// Command read demonstrates reading audio metadata with mtag.
//
// Usage:
//
//	go run ./examples/read song.mp3
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tommyo123/mtag"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: read <audio-file>")
		os.Exit(2)
	}

	f, err := mtag.Open(os.Args[1], mtag.WithReadOnly())
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	fmt.Printf("Container: %s\n", f.Container())
	fmt.Printf("Formats:   %s\n", f.Formats())
	fmt.Printf("Title:     %s\n", f.Title())
	fmt.Printf("Artist:    %s\n", f.Artist())
	fmt.Printf("Album:     %s\n", f.Album())
	fmt.Printf("Year:      %d\n", f.Year())

	if total := f.TrackTotal(); total > 0 {
		fmt.Printf("Track:     %d/%d\n", f.Track(), total)
	} else if tr := f.Track(); tr > 0 {
		fmt.Printf("Track:     %d\n", tr)
	}

	if total := f.DiscTotal(); total > 0 {
		fmt.Printf("Disc:      %d/%d\n", f.Disc(), total)
	}

	fmt.Printf("Genre:     %s\n", f.Genre())

	if c := f.Comment(); c != "" {
		fmt.Printf("Comment:   %s\n", c)
	}

	// Audio properties.
	ap := f.AudioProperties()
	if ap.Codec != "" {
		fmt.Printf("Codec:     %s\n", ap.Codec)
		fmt.Printf("Duration:  %s\n", ap.Duration)
		fmt.Printf("Bitrate:   %d bps\n", ap.Bitrate)
		fmt.Printf("Sample:    %d Hz, %d ch\n", ap.SampleRate, ap.Channels)
	}

	// Images.
	for i, img := range f.ImageSummaries() {
		fmt.Printf("Image %d:   %s (%d bytes, type %d)\n", i, img.MIME, img.Size, img.Type)
	}

	// Tag stores.
	for _, tag := range f.Tags() {
		fmt.Printf("Tag:       %s (%d keys)\n", tag.Kind(), len(tag.Keys()))
	}
}
