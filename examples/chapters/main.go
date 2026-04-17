// Command chapters demonstrates reading and writing chapter markers.
//
// Usage:
//
//	go run ./examples/chapters song.mp3              # list chapters
//	go run ./examples/chapters song.mp3 set          # set example chapters
//	go run ./examples/chapters song.mp3 clear        # remove all chapters
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tommyo123/mtag"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: chapters <audio-file> [set|clear]")
		os.Exit(2)
	}

	switch {
	case len(os.Args) == 2:
		listChapters(os.Args[1])
	case os.Args[2] == "set":
		setChapters(os.Args[1])
	case os.Args[2] == "clear":
		clearChapters(os.Args[1])
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[2])
		os.Exit(2)
	}
}

func listChapters(path string) {
	f, err := mtag.Open(path, mtag.WithReadOnly())
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	chapters := f.Chapters()
	if len(chapters) == 0 {
		fmt.Println("no chapters")
		return
	}
	for i, ch := range chapters {
		fmt.Printf("[%d] %s  %s - %s\n", i, ch.Title, ch.Start, ch.End)
	}
}

func setChapters(path string) {
	f, err := mtag.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	chapters := []mtag.Chapter{
		{Title: "Intro", Start: 0, End: 30 * time.Second},
		{Title: "Verse 1", Start: 30 * time.Second, End: 90 * time.Second},
		{Title: "Chorus", Start: 90 * time.Second, End: 120 * time.Second},
		{Title: "Outro", Start: 120 * time.Second, End: 150 * time.Second},
	}
	if err := f.SetChapters(chapters); err != nil {
		log.Fatal(err)
	}
	if err := f.Save(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("set %d chapters\n", len(chapters))
}

func clearChapters(path string) {
	f, err := mtag.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := f.RemoveChapters(); err != nil {
		log.Fatal(err)
	}
	if err := f.Save(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("chapters cleared")
}
