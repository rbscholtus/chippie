package app

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/rbscholtus/chippie/romstore"
)

func TestReadDropped(t *testing.T) {
	t.Parallel()

	// A folder and two files: the first regular file wins.
	fsys := fstest.MapFS{
		"a-folder/inner.ch8": {Data: []byte{9}},
		"b.ch8":              {Data: []byte{1, 2, 3}},
		"c.ch8":              {Data: []byte{4}},
	}

	got := readDropped(fsys)
	if got.Err != nil || got.Name != "a-folder/inner.ch8" || string(got.Data) != "\x09" {
		t.Errorf("first file: %+v", got)
	}

	big := readDropped(fstest.MapFS{"big.ch8": {Data: make([]byte, romstore.MaxROMSize+1)}})
	if !errors.Is(big.Err, romstore.ErrUnavailable) {
		t.Errorf("oversized drop: err=%v", big.Err)
	}

	if empty := readDropped(fstest.MapFS{}); empty.Name != "" || empty.Data != nil || empty.Err != nil {
		t.Errorf("empty drop: %+v", empty)
	}
}

func TestBaseName(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]string{"roms/a.ch8": "a.ch8", `C:\x\b.ch8`: "b.ch8", "c.ch8": "c.ch8"} {
		if got := baseName(in); got != want {
			t.Errorf("baseName(%q) = %q, want %q", in, got, want)
		}
	}
}
