// Command write demonstrates modifying audio metadata with mtag.
//
// Usage:
//
//	go run ./examples/write song.mp3
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tommyo123/mtag"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: write <audio-file>")
		os.Exit(2)
	}

	f, err := mtag.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Set common text fields.
	f.SetTitle("Hello World")
	f.SetArtist("mtag Example")
	f.SetAlbum("Examples Album")
	f.SetYear(2025)
	f.SetTrack(1, 10)
	f.SetGenre("Electronic")
	f.SetComment("Written by the mtag example program")

	// Check for setter errors (e.g. field not supported on this format).
	if err := f.Err(); err != nil {
		log.Printf("setter warning: %v", err)
		f.ResetErr()
	}

	// Save persists every tag present on disk.
	if err := f.Save(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("metadata updated successfully")
}
