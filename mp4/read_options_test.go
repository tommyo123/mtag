package mp4

import (
	"bytes"
	"testing"
)

func wrapAtom(name string, payload []byte) []byte {
	out := make([]byte, 0, 8+len(payload))
	out = append(out, encodeAtomHeader(name, 8+len(payload))...)
	out = append(out, payload...)
	return out
}

func TestReadMetadataWithOptionsSkipsCoverArt(t *testing.T) {
	ftyp := append(encodeAtomHeader("ftyp", 16), []byte("M4A ")...)
	ftyp = append(ftyp, 0, 0, 0, 0)

	var titleName [4]byte
	copy(titleName[:], []byte{0xA9, 'n', 'a', 'm'})
	var coverName [4]byte
	copy(coverName[:], []byte("covr"))

	ilstBody := append(EncodeItem(Item{Name: titleName, Type: DataUTF8, Data: []byte("Song")}),
		EncodeItem(Item{Name: coverName, Type: DataPNG, Data: []byte("\x89PNGcover")})...)
	ilst := wrapAtom("ilst", ilstBody)
	meta := wrapAtom("meta", append([]byte{0, 0, 0, 0}, ilst...))
	udta := wrapAtom("udta", meta)
	moov := wrapAtom("moov", udta)
	data := append(ftyp, moov...)

	items, _, err := ReadMetadataWithOptions(bytes.NewReader(data), int64(len(data)), true)
	if err != nil {
		t.Fatalf("ReadMetadataWithOptions() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Key() != string(titleName[:]) {
		t.Fatalf("item key = %q, want %q", items[0].Key(), string(titleName[:]))
	}
}
