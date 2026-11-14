package catalogue

import (
	"errors"
	"strings"
	"testing"
)

const (
	newHash   = "1111111111111111111111111111111111111111"
	otherHash = "2222222222222222222222222222222222222222"
	upstream  = "https://github.com/chip-8/chip-8-database/pull/99"
)

func mustOverlay(t *testing.T, s string) *Overlay {
	t.Helper()

	o, err := ParseOverlay([]byte(s))
	if err != nil {
		t.Fatal(err)
	}

	return o
}

func TestOverlayAddsAndChanges(t *testing.T) {
	t.Parallel()

	db := loadSnapshot(t).DB
	programs := len(db.Programs)

	o := mustOverlay(t, `{
		"add": [
			{"upstream": "pending", "program": {"title": "New Game", "authors": ["Me"],
				"roms": {"`+newHash+`": {"file": "new.ch8", "platforms": ["xochip"]}}}},
			{"upstream": "`+upstream+`", "to": "`+glitchGhost+`",
				"program": {"roms": {"`+otherHash+`": {"file": "alt.ch8", "platforms": ["modernChip8"]}}}}
		],
		"change": [
			{"upstream": "`+upstream+`", "sha1": "`+glitchGhost+`", "title": "Glitch Ghost (fixed)", "tickrate": 30}
		]}`)

	status, err := db.ApplyOverlay(o)
	if err != nil {
		t.Fatal(err)
	}

	if rom, ok := db.ROM(newHash); !ok || rom.Program.Title != "New Game" || len(db.Programs) != programs+1 {
		t.Errorf("new program not added: %+v (programs %d → %d)", rom, programs, len(db.Programs))
	}

	if rom, ok := db.ROM(otherHash); !ok || rom.Program.Title != "Glitch Ghost (fixed)" {
		t.Errorf("ROM not added to the existing program: %+v", rom)
	}

	if rom, _ := db.ROM(glitchGhost); rom.Tickrate != 30 {
		t.Errorf("tickrate not changed: %d", rom.Tickrate)
	}

	for _, st := range status {
		if st.Done {
			t.Errorf("%s: done, but upstream doesn't have it", st.What)
		}
	}

	if !strings.Contains(status[0].What, "title, tickrate") {
		t.Errorf("change status %q doesn't list its fields", status[0].What)
	}

	if status[2].What != "add ROMs to Glitch Ghost (fixed)" {
		t.Errorf("the addition was reported as %q, before the rename", status[2].What)
	}

	// Applied again (upstream caught up), every entry is done.
	status, err = db.ApplyOverlay(o)
	if err != nil {
		t.Fatal(err)
	}

	for _, st := range status {
		if !st.Done {
			t.Errorf("%s: not done although the database has it", st.What)
		}
	}
}

func TestOverlayRejects(t *testing.T) {
	t.Parallel()

	for name, s := range map[string]string{
		"bad JSON":      `{`,
		"no upstream":   `{"add": [{"program": {"title": "X", "roms": {"` + newHash + `": {}}}}]}`,
		"not a PR link": `{"add": [{"upstream": "soon", "program": {"title": "X", "roms": {"` + newHash + `": {}}}}]}`,
		"no ROMs":       `{"add": [{"upstream": "pending", "program": {"title": "X"}}]}`,
		"no title":      `{"add": [{"upstream": "pending", "program": {"roms": {"` + newHash + `": {}}}}]}`,
		"upper-case hash": `{"add": [{"upstream": "pending", "program": {"title": "X", "roms": {"` +
			strings.ToUpper("abcdef") + newHash[6:] + `": {}}}}]}`,
		"change no hash": `{"change": [{"upstream": "pending", "sha1": "x"}]}`,
	} {
		if _, err := ParseOverlay([]byte(s)); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}

	db := loadSnapshot(t).DB

	for name, s := range map[string]string{
		"unknown platform": `{"add": [{"upstream": "pending", "program": {"title": "X",
			"roms": {"` + newHash + `": {"platforms": ["nope"]}}}}]}`,
		"unknown quirk": `{"add": [{"upstream": "pending", "program": {"title": "X",
			"roms": {"` + newHash + `": {"quirkyPlatforms": {"superchip": {"nope": true}}}}}}]}`,
		"add to unknown": `{"add": [{"upstream": "pending", "to": "` + otherHash + `",
			"program": {"roms": {"` + newHash + `": {}}}}]}`,
		"change unknown": `{"change": [{"upstream": "pending", "sha1": "` + otherHash + `", "tickrate": 5}]}`,
		"bad rotation": `{"add": [{"upstream": "pending", "program": {"title": "X",
			"roms": {"` + newHash + `": {"screenRotation": 45}}}}]}`,
	} {
		if _, err := db.ApplyOverlay(mustOverlay(t, s)); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
}

// TestEmbeddedOverlay checks Chippie's own overlay: it applies cleanly and
// its games resolve to a platform Chippie can play.
func TestEmbeddedOverlay(t *testing.T) {
	t.Parallel()

	cat := loadSnapshot(t)

	if len(cat.Overlay) == 0 {
		t.Fatal("the embedded overlay did nothing")
	}

	for _, sha1 := range []string{
		"1a7a2138161b06e083722f978f8cbaaa1ad5344d", // Dino Run
		"673f98c66168190aa5b1e5dda5d0d80da7826dd1", // Mato8
		"3d81c0462c722c2d8a79f6e1b571e9aa21d36b1f", // Tick
		"08cff17fa88aaecb11dd439e2f9780f77831de02", // Mr Worm
	} {
		rom, ok := cat.DB.ROM(sha1)
		if !ok {
			t.Errorf("%s not in the catalogue", sha1)

			continue
		}

		if p := cat.DB.Resolve(sha1, Overrides{}).Profile.Platform; !p.Implemented() {
			t.Errorf("%s: platform %s isn't playable", rom.Program.Title, p)
		}
	}
}

func TestScreenRotation(t *testing.T) {
	t.Parallel()

	db := loadSnapshot(t).DB

	// Sub-Terr8nia, from chip-8-database itself.
	if rom, ok := db.ROM("64536d549c986e9edf25de9fa89db60d2ade85c0"); !ok || rom.ScreenRotation != 270 {
		t.Errorf("Sub-Terr8nia: %+v, want screenRotation 270", rom)
	}

	status, err := db.ApplyOverlay(mustOverlay(t, `{"change": [{"upstream": "pending",
		"sha1": "`+glitchGhost+`", "screenRotation": 90}]}`))
	if err != nil || status[0].Done {
		t.Fatalf("change: %v %+v", err, status)
	}

	if rom, _ := db.ROM(glitchGhost); rom.ScreenRotation != 90 {
		t.Errorf("rotation not changed: %d", rom.ScreenRotation)
	}
}
