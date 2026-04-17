// Command images demonstrates cover art handling with mtag.
//
// Usage:
//
//	go run ./examples/images song.mp3           # list images
//	go run ./examples/images song.mp3 cover.jpg # set front cover
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tommyo123/mtag"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: images <audio-file> [cover-image]")
		os.Exit(2)
	}

	if len(os.Args) == 2 {
		listImages(os.Args[1])
		return
	}
	setCover(os.Args[1], os.Args[2])
}

func listImages(path string) {
	f, err := mtag.Open(path, mtag.WithReadOnly())
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	imgs := f.Images()
	if len(imgs) == 0 {
		fmt.Println("no embedded images")
		return
	}
	for i, img := range imgs {
		fmt.Printf("[%d] %s  type=%d  %d bytes  %q\n",
			i, img.MIME, img.Type, len(img.Data), img.Description)
	}
}

func setCover(audioPath, imagePath string) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		log.Fatal(err)
	}
	mime := guessMIME(imagePath)

	f, err := mtag.Open(audioPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	f.SetCoverArt(mime, data)
	if err := f.Save(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("set front cover: %s (%d bytes)\n", mime, len(data))
}

func guessMIME(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	default:
		return "image/jpeg"
	}
}
