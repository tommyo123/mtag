package tests

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/tommyo123/mtag"
)

type cancelAfterFirstWriteSource struct {
	data   []byte
	writes int
	cancel context.CancelFunc
}

func (w *cancelAfterFirstWriteSource) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(w.data)) {
		return 0, errors.New("EOF")
	}
	n := copy(p, w.data[off:])
	if n < len(p) {
		return n, errors.New("EOF")
	}
	return n, nil
}

func (w *cancelAfterFirstWriteSource) WriteAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("EOF")
	}
	end := int(off) + len(p)
	if end > len(w.data) {
		grow := make([]byte, end)
		copy(grow, w.data)
		w.data = grow
	}
	copy(w.data[off:], p)
	w.writes++
	if w.writes == 1 && w.cancel != nil {
		w.cancel()
	}
	return len(p), nil
}

func (w *cancelAfterFirstWriteSource) Truncate(size int64) error {
	if size < 0 {
		return errors.New("EOF")
	}
	switch {
	case int(size) < len(w.data):
		w.data = w.data[:size]
	case int(size) > len(w.data):
		grow := make([]byte, size)
		copy(grow, w.data)
		w.data = grow
	}
	return nil
}

type failOnSecondWriteSource struct {
	data   []byte
	writes int
	failed bool
}

func (w *failOnSecondWriteSource) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(w.data)) {
		return 0, errors.New("EOF")
	}
	n := copy(p, w.data[off:])
	if n < len(p) {
		return n, errors.New("EOF")
	}
	return n, nil
}

func (w *failOnSecondWriteSource) WriteAt(p []byte, off int64) (int, error) {
	w.writes++
	if w.writes == 2 && !w.failed {
		w.failed = true
		return 0, errors.New("injected write failure")
	}
	end := int(off) + len(p)
	if end > len(w.data) {
		grow := make([]byte, end)
		copy(grow, w.data)
		w.data = grow
	}
	copy(w.data[off:], p)
	return len(p), nil
}

func (w *failOnSecondWriteSource) Truncate(size int64) error {
	if size < 0 {
		return errors.New("EOF")
	}
	switch {
	case int(size) < len(w.data):
		w.data = w.data[:size]
	case int(size) > len(w.data):
		grow := make([]byte, size)
		copy(grow, w.data)
		w.data = grow
	}
	return nil
}

type failOnFirstTruncateSource struct {
	data          []byte
	truncateCalls int
	failed        bool
}

func (w *failOnFirstTruncateSource) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(w.data)) {
		return 0, errors.New("EOF")
	}
	n := copy(p, w.data[off:])
	if n < len(p) {
		return n, errors.New("EOF")
	}
	return n, nil
}

func (w *failOnFirstTruncateSource) WriteAt(p []byte, off int64) (int, error) {
	end := int(off) + len(p)
	if end > len(w.data) {
		grow := make([]byte, end)
		copy(grow, w.data)
		w.data = grow
	}
	copy(w.data[off:], p)
	return len(p), nil
}

func (w *failOnFirstTruncateSource) Truncate(size int64) error {
	w.truncateCalls++
	if w.truncateCalls == 1 && !w.failed {
		w.failed = true
		return errors.New("injected truncate failure")
	}
	if size < 0 {
		return errors.New("EOF")
	}
	switch {
	case int(size) < len(w.data):
		w.data = w.data[:size]
	case int(size) > len(w.data):
		grow := make([]byte, size)
		copy(grow, w.data)
		w.data = grow
	}
	return nil
}

func TestPathlessSaveContextCancelLeavesSourceUntouched(t *testing.T) {
	original := buildTwoFrameMP3(0)
	ctx, cancel := context.WithCancel(context.Background())
	src := &cancelAfterFirstWriteSource{
		data:   append([]byte(nil), original...),
		cancel: cancel,
	}

	f, err := mtag.OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.SetCoverArt("image/jpeg", bytes.Repeat([]byte{0xFF, 0xD8, 0xFF, 0xE0}, 300_000))

	err = f.SaveContext(ctx)
	if err == nil {
		t.Fatal("SaveContext() error = nil, want cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SaveContext() error = %v, want context.Canceled", err)
	}
	if !bytes.Equal(src.data, original) {
		t.Fatal("writable source changed despite cancelled SaveContext")
	}
}

func TestPathlessRewriteFailureLeavesSourceUntouched(t *testing.T) {
	original := buildTwoFrameMP3(0)
	src := &failOnSecondWriteSource{
		data: append([]byte(nil), original...),
	}

	f, err := mtag.OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.SetCoverArt("image/jpeg", bytes.Repeat([]byte{0xFF, 0xD8, 0xFF, 0xE0}, 300_000))

	err = f.Save()
	if err == nil {
		t.Fatal("Save() error = nil, want injected write failure")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("injected write failure")) {
		t.Fatalf("Save() error = %v", err)
	}
	if !bytes.Equal(src.data, original) {
		t.Fatal("writable source changed after injected WriteAt failure")
	}
}

func TestPathlessTruncateFailureLeavesSourceUntouched(t *testing.T) {
	original := buildTwoFrameMP3(0)
	src := &failOnFirstTruncateSource{
		data: append([]byte(nil), original...),
	}

	f, err := mtag.OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.SetCoverArt("image/jpeg", bytes.Repeat([]byte{0xFF, 0xD8, 0xFF, 0xE0}, 300_000))

	err = f.Save()
	if err == nil {
		t.Fatal("Save() error = nil, want injected truncate failure")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("injected truncate failure")) {
		t.Fatalf("Save() error = %v", err)
	}
	if !bytes.Equal(src.data, original) {
		t.Fatal("writable source changed after injected Truncate failure")
	}
}

func TestPathlessSaveCanRetryAfterRollback(t *testing.T) {
	original := buildTwoFrameMP3(0)
	src := &failOnSecondWriteSource{
		data: append([]byte(nil), original...),
	}

	f, err := mtag.OpenWritableSource(src, int64(len(src.data)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.SetTitle("retry-title")
	f.SetArtist("retry-artist")
	f.SetCoverArt("image/jpeg", bytes.Repeat([]byte{0xFF, 0xD8, 0xFF, 0xE0}, 300_000))

	if err := f.Save(); err == nil {
		t.Fatal("first Save() error = nil, want injected write failure")
	}
	if !bytes.Equal(src.data, original) {
		t.Fatal("writable source changed after failed save")
	}

	if err := f.Save(); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}

	g, err := mtag.OpenBytes(src.data)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if got := g.Title(); got != "retry-title" {
		t.Fatalf("Title() = %q, want retry-title", got)
	}
	if got := g.Artist(); got != "retry-artist" {
		t.Fatalf("Artist() = %q, want retry-artist", got)
	}
	if len(g.Images()) == 0 {
		t.Fatal("cover art missing after retry save")
	}
}
