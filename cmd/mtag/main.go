// Command mtag prints and updates audio metadata using the mtag
// library. It is intentionally tiny; its purpose is to serve as an
// end-to-end smoke test and a usage example.
//
// Usage:
//
//	mtag show <file>
//	mtag set   <file> key=value [key=value ...]
//	mtag cover <file> <image-path>     # writes front-cover APIC
//	mtag strip <file> v1|v2|all
//	mtag copy  <src> <dst> [field ...] # copy tags between files
//	mtag diff  <a> <b>                 # show field-by-field deltas
//
// Supported keys for "set": title, artist, album, albumartist,
// composer, year, track, disc, genre, comment.
package main

import (
	"crypto/md5"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tommyo123/mtag"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	var err error
	switch os.Args[1] {
	case "show":
		err = cmdShow(os.Args[2:])
	case "set":
		err = cmdSet(os.Args[2:])
	case "cover":
		err = cmdCover(os.Args[2:])
	case "strip":
		err = cmdStrip(os.Args[2:])
	case "copy":
		err = cmdCopy(os.Args[2:])
	case "diff":
		err = cmdDiff(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "mtag:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: mtag show|set|cover|strip|copy|diff <file> [...]")
}

func cmdShow(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("show: missing file")
	}
	f, err := mtag.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Printf("file:    %s\n", f.Path())
	fmt.Printf("formats: %s\n", f.Formats())
	fmt.Printf("title:   %s\n", f.Title())
	fmt.Printf("artist:  %s\n", f.Artist())
	fmt.Printf("album:   %s\n", f.Album())
	fmt.Printf("band:    %s\n", f.AlbumArtist())
	fmt.Printf("year:    %d\n", f.Year())
	if total := f.TrackTotal(); total > 0 {
		fmt.Printf("track:   %d/%d\n", f.Track(), total)
	} else if tr := f.Track(); tr > 0 {
		fmt.Printf("track:   %d\n", tr)
	}
	fmt.Printf("genre:   %s\n", f.Genre())
	if c := f.Comment(); c != "" {
		fmt.Printf("comment: %s\n", c)
	}
	for i, p := range f.Images() {
		fmt.Printf("image %d: %s (%d bytes, type %d) %q\n",
			i, p.MIME, len(p.Data), p.Type, p.Description)
	}
	return nil
}

func cmdSet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("set: need <file> key=value ...")
	}
	f, err := mtag.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	for _, kv := range args[1:] {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			return fmt.Errorf("set: malformed %q (want key=value)", kv)
		}
		key := strings.ToLower(strings.TrimSpace(kv[:eq]))
		val := kv[eq+1:]
		switch key {
		case "title":
			f.SetTitle(val)
		case "artist":
			f.SetArtist(val)
		case "album":
			f.SetAlbum(val)
		case "albumartist", "band":
			f.SetAlbumArtist(val)
		case "composer":
			f.SetComposer(val)
		case "year":
			y, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			f.SetYear(y)
		case "track":
			n, total := parseTrack(val)
			f.SetTrack(n, total)
		case "disc":
			n, total := parseTrack(val)
			f.SetDisc(n, total)
		case "genre":
			f.SetGenre(val)
		case "comment":
			f.SetComment(val)
		default:
			return fmt.Errorf("set: unknown key %q", key)
		}
	}
	return f.Save()
}

func parseTrack(s string) (int, int) {
	slash := strings.IndexByte(s, '/')
	if slash < 0 {
		n, _ := strconv.Atoi(s)
		return n, 0
	}
	a, _ := strconv.Atoi(s[:slash])
	b, _ := strconv.Atoi(s[slash+1:])
	return a, b
}

func cmdCover(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("cover: need <file> <image>")
	}
	f, err := mtag.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := os.ReadFile(args[1])
	if err != nil {
		return err
	}
	f.SetCoverArt(guessMIME(args[1]), data)
	return f.Save()
}

func cmdStrip(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("strip: need <file> v1|v2|all")
	}
	f, err := mtag.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()
	var want mtag.Format
	switch args[1] {
	case "v1":
		want = f.Formats() & mtag.FormatID3v2Any
	case "v2":
		want = f.Formats() & mtag.FormatID3v1
	case "all":
		want = 0
	default:
		return fmt.Errorf("strip: want v1, v2, or all")
	}
	return f.SaveWith(want)
}

// cmdCopy lifts metadata from src into dst. With no extra args
// every supported polymorphic field is copied (including images);
// when a list of field names follows only those are touched, so
// "mtag copy a.mp3 b.mp3 title artist" leaves dst's other tags
// alone.
func cmdCopy(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("copy: need <src> <dst> [field ...]")
	}
	src, err := mtag.Open(args[0])
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer src.Close()
	dst, err := mtag.Open(args[1])
	if err != nil {
		return fmt.Errorf("open dst: %w", err)
	}
	defer dst.Close()

	wanted := map[string]bool{}
	for _, name := range args[2:] {
		wanted[strings.ToLower(name)] = true
	}
	want := func(name string) bool {
		return len(wanted) == 0 || wanted[name]
	}

	if want("title") {
		dst.SetTitle(src.Title())
	}
	if want("artist") {
		dst.SetArtist(src.Artist())
	}
	if want("album") {
		dst.SetAlbum(src.Album())
	}
	if want("albumartist") || want("band") {
		dst.SetAlbumArtist(src.AlbumArtist())
	}
	if want("composer") {
		dst.SetComposer(src.Composer())
	}
	if want("year") {
		dst.SetYear(src.Year())
	}
	if want("track") {
		dst.SetTrack(src.Track(), src.TrackTotal())
	}
	if want("disc") {
		dst.SetDisc(src.Disc(), src.DiscTotal())
	}
	if want("genre") {
		dst.SetGenre(src.Genre())
	}
	if want("comment") {
		dst.SetComment(src.Comment())
	}
	if want("lyrics") {
		dst.SetLyrics(src.Lyrics())
	}
	if want("compilation") {
		dst.SetCompilation(src.IsCompilation())
	}
	if want("images") || want("image") || want("cover") {
		dst.RemoveImages()
		for _, p := range src.Images() {
			dst.AddImage(p)
		}
	}
	return dst.Save()
}

// cmdDiff prints every polymorphic field where a and b disagree.
// Identical fields are omitted; image lists are compared by count
// and per-image MD5 fingerprint so a re-encoded cover surfaces as a
// single "differs" line rather than dumping the bytes.
func cmdDiff(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("diff: need <a> <b>")
	}
	a, err := mtag.Open(args[0])
	if err != nil {
		return fmt.Errorf("open %s: %w", args[0], err)
	}
	defer a.Close()
	b, err := mtag.Open(args[1])
	if err != nil {
		return fmt.Errorf("open %s: %w", args[1], err)
	}
	defer b.Close()

	pairs := []struct {
		name string
		av   string
		bv   string
	}{
		{"formats", a.Formats().String(), b.Formats().String()},
		{"container", a.Container().String(), b.Container().String()},
		{"title", a.Title(), b.Title()},
		{"artist", a.Artist(), b.Artist()},
		{"album", a.Album(), b.Album()},
		{"album_artist", a.AlbumArtist(), b.AlbumArtist()},
		{"composer", a.Composer(), b.Composer()},
		{"year", strconv.Itoa(a.Year()), strconv.Itoa(b.Year())},
		{"track", trackString(a.Track(), a.TrackTotal()), trackString(b.Track(), b.TrackTotal())},
		{"disc", trackString(a.Disc(), a.DiscTotal()), trackString(b.Disc(), b.DiscTotal())},
		{"genre", a.Genre(), b.Genre()},
		{"comment", a.Comment(), b.Comment()},
		{"compilation", strconv.FormatBool(a.IsCompilation()), strconv.FormatBool(b.IsCompilation())},
	}
	any := false
	for _, p := range pairs {
		if p.av == p.bv {
			continue
		}
		fmt.Printf("%-12s  - %s\n%-12s  + %s\n", p.name, p.av, "", p.bv)
		any = true
	}

	// Image comparison.
	aImg := imageDigests(a.Images())
	bImg := imageDigests(b.Images())
	if !equalStringSlices(aImg, bImg) {
		fmt.Printf("%-12s  - %v\n%-12s  + %v\n", "images", aImg, "", bImg)
		any = true
	}
	if !any {
		fmt.Println("(no differences)")
	}
	return nil
}

func trackString(n, total int) string {
	if n <= 0 {
		return ""
	}
	if total > 0 {
		return fmt.Sprintf("%d/%d", n, total)
	}
	return strconv.Itoa(n)
}

func imageDigests(imgs []mtag.Picture) []string {
	out := make([]string, len(imgs))
	for i, p := range imgs {
		sum := md5.Sum(p.Data)
		out[i] = fmt.Sprintf("%s:%d:%x", p.MIME, p.Type, sum[:6])
	}
	return out
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func guessMIME(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".bmp"):
		return "image/bmp"
	}
	return "image/jpeg"
}
