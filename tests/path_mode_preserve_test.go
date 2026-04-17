package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tommyo123/mtag"
)

func TestEdge_ModePreservedOnMP3Rewrite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-bit preservation is not meaningful on Windows")
	}

	path := buildTestFile(t, nil, nil, []byte("audio"))
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	wantMode, err := filePerms(path)
	if err != nil {
		t.Fatal(err)
	}

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.SetTitle("mode-test")
	f.SetCoverArt("image/png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 1, 2, 3, 4})
	if err := f.Save(); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	gotMode, err := filePerms(path)
	if err != nil {
		t.Fatal(err)
	}
	if gotMode != wantMode {
		t.Fatalf("mode after rewrite = %03o, want %03o", gotMode, wantMode)
	}
}

func TestEdge_ModePreservedOnFLACRewrite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-bit preservation is not meaningful on Windows")
	}

	path := filepath.Join(t.TempDir(), "mode.flac")
	if err := os.WriteFile(path, buildMinimalFLAC(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	wantMode, err := filePerms(path)
	if err != nil {
		t.Fatal(err)
	}

	f, err := mtag.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.SetComment("mode-test")
	f.SetCoverArt("image/png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 1, 2, 3, 4})
	if err := f.Save(); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	gotMode, err := filePerms(path)
	if err != nil {
		t.Fatal(err)
	}
	if gotMode != wantMode {
		t.Fatalf("mode after rewrite = %03o, want %03o", gotMode, wantMode)
	}
}

func filePerms(path string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Mode().Perm(), nil
}
