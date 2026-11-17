package catalogue

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rbscholtus/chippie/chip8"
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

// testPlatform is an overlay platform entry for twoPageChip8, which
// dbWithoutTwoPage's database doesn't have.
const testPlatform = `{"upstream": "pending", "platform": {"id": "twoPageChip8", "name": "Hires",
	"displayResolutions": ["64x64"], "defaultTickrate": 15, "quirks": {"vblank": true, "logic": true}}}`

// dbWithoutTwoPage parses the snapshot, without the embedded overlay, as
// if upstream didn't have twoPageChip8 and its programs. Every platform
// Chippie knows is in chip-8-database, so the overlay tests that add one
// take this one out, whatever upstream has merged.
func dbWithoutTwoPage(t *testing.T) *DB {
	t.Helper()

	const id = "twoPageChip8"

	f, err := SnapshotFiles()
	if err != nil {
		t.Fatal(err)
	}

	var platforms, programs []json.RawMessage
	if err := json.Unmarshal(f.Platforms, &platforms); err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(f.Programs, &programs); err != nil {
		t.Fatal(err)
	}

	var keptPlatforms, keptPrograms []json.RawMessage

	for _, p := range platforms {
		if !bytes.Contains(p, []byte(`"`+id+`"`)) {
			keptPlatforms = append(keptPlatforms, p)
		}
	}

	hashes := map[string]int{}

	for _, p := range programs {
		if bytes.Contains(p, []byte(`"`+id+`"`)) {
			continue
		}

		var prog struct {
			ROMs map[string]json.RawMessage `json:"roms"`
		}
		if err := json.Unmarshal(p, &prog); err != nil {
			t.Fatal(err)
		}

		for h := range prog.ROMs {
			hashes[h] = len(keptPrograms)
		}

		keptPrograms = append(keptPrograms, p)
	}

	for _, part := range []struct {
		v   any
		dst *[]byte
	}{{keptPlatforms, &f.Platforms}, {keptPrograms, &f.Programs}, {hashes, &f.Hashes}} {
		if *part.dst, err = json.Marshal(part.v); err != nil {
			t.Fatal(err)
		}
	}

	db, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	if db.Platforms[id] != nil {
		t.Fatal("dbWithoutTwoPage still has " + id)
	}

	return db
}

func TestOverlayAddsPlatform(t *testing.T) {
	t.Parallel()

	db := dbWithoutTwoPage(t)

	o := mustOverlay(t, `{"platforms": [`+testPlatform+`],
		"add": [{"upstream": "pending", "program": {"title": "Hires Game",
			"roms": {"`+newHash+`": {"file": "h.ch8", "platforms": ["twoPageChip8"]}}}}]}`)

	status, err := db.ApplyOverlay(o)
	if err != nil {
		t.Fatal(err)
	}

	if len(status) != 2 || status[0].What != "add platform twoPageChip8" || status[0].Done {
		t.Errorf("status %+v, want the platform first, not done", status)
	}

	r := db.Resolve(newHash, Overrides{})
	if r.PlatformID != "twoPageChip8" || r.Profile.Platform != chip8.TwoPageChip8 || r.Profile.TicksPerFrame != 15 ||
		!r.Profile.Quirks.VBlank || !r.Profile.Quirks.Logic {
		t.Errorf("resolved %+v", r)
	}

	// The user can switch another ROM to it, and back.
	if r := db.Resolve(glitchGhost, Overrides{Platform: "twoPageChip8"}); r.Profile.Platform != chip8.TwoPageChip8 {
		t.Errorf("override: %s, want twoPageChip8", r.Profile.Platform)
	}

	// Once upstream has the platform, the entry is done.
	if status, err = db.ApplyOverlay(o); err != nil || !status[0].Done {
		t.Errorf("applied twice: %v %+v", err, status)
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
		"platform chip8 doesn't know": `{"platforms": [{"upstream": "pending",
			"platform": {"id": "nope", "name": "Nope", "defaultTickrate": 15}}]}`,
		"platform without a name": `{"platforms": [{"upstream": "pending",
			"platform": {"id": "twoPageChip8", "defaultTickrate": 15}}]}`,
		"platform without upstream": `{"platforms": [{"platform": {"id": "twoPageChip8", "name": "H"}}]}`,
	} {
		if _, err := ParseOverlay([]byte(s)); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}

	for name, s := range map[string]string{
		"unknown platform": `{"add": [{"upstream": "pending", "program": {"title": "X",
			"roms": {"` + newHash + `": {"platforms": ["nope"]}}}}]}`,
		"platform the overlay doesn't add": `{"add": [{"upstream": "pending", "program": {"title": "X",
			"roms": {"` + newHash + `": {"platforms": ["twoPageChip8"]}}}}]}`,
		"unknown quirk": `{"add": [{"upstream": "pending", "program": {"title": "X",
			"roms": {"` + newHash + `": {"quirkyPlatforms": {"superchip": {"nope": true}}}}}}]}`,
		"add to unknown": `{"add": [{"upstream": "pending", "to": "` + otherHash + `",
			"program": {"roms": {"` + newHash + `": {}}}}]}`,
		"change unknown": `{"change": [{"upstream": "pending", "sha1": "` + otherHash + `", "tickrate": 5}]}`,
		"bad rotation": `{"add": [{"upstream": "pending", "program": {"title": "X",
			"roms": {"` + newHash + `": {"screenRotation": 45}}}}]}`,
		"platform with an unknown quirk": `{"platforms": [{"upstream": "pending", "platform":
			{"id": "twoPageChip8", "name": "H", "defaultTickrate": 15, "quirks": {"nope": true}}}]}`,
		"platform without a tickrate": `{"platforms": [{"upstream": "pending", "platform":
			{"id": "twoPageChip8", "name": "H"}}]}`,
	} {
		// A fresh database each time: an entry that wrongly applied would
		// add newHash, and the next case would skip it as done.
		if _, err := dbWithoutTwoPage(t).ApplyOverlay(mustOverlay(t, s)); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
}

// TestEmbeddedOverlay checks Chippie's own overlay: it applies cleanly,
// every entry reports a status (an empty overlay, once upstream has
// everything, is fine), and the games it added, which upstream may have
// by now, resolve to a platform Chippie can play.
func TestEmbeddedOverlay(t *testing.T) {
	t.Parallel()

	cat := loadSnapshot(t)

	o, err := EmbeddedOverlay()
	if err != nil {
		t.Fatal(err)
	}

	if n := len(o.Platforms) + len(o.Add) + len(o.Change); len(cat.Overlay) != n {
		t.Fatalf("%d statuses for %d overlay entries", len(cat.Overlay), n)
	}

	for _, sha1 := range []string{
		"1a7a2138161b06e083722f978f8cbaaa1ad5344d", // Dino Run
		"673f98c66168190aa5b1e5dda5d0d80da7826dd1", // Mato8
		"3d81c0462c722c2d8a79f6e1b571e9aa21d36b1f", // Tick
		"08cff17fa88aaecb11dd439e2f9780f77831de02", // Mr Worm
	} {
		checkPlayable(t, cat, sha1)
	}

	// The 2 Page Display ROMs, on twoPageChip8 (from the overlay or upstream).
	for _, sha1 := range []string{
		"066e7a84efde433e4d937d8aa41518666955086c", // Astro Dodge Hires
		"70aa0e7f25f0f0fd6ec7c59e427bf1d03ee95617", // Hires Maze
		"1ebcb2ec0be2ec9fa209d5c73be19b2d408399bf", // Hires Particle Demo
		"200b313e4d4c1970641142cc7ff578d7956b93da", // Hires Sierpinski
		"af98ee11adae28a6153cae8e4c16afa00f861907", // Hires Stars
		"8d56a781bf16acccb307177b80ff326f62aabbdc", // Hires Test
		"71d06da9e605804d2099b808c02548ab2b3511b2", // Hires Worm V4
		"b2c55b6aba3e2910036d5b5bc3956cf7493e0221", // Trip8 Hires Demo
	} {
		checkPlayable(t, cat, sha1)

		prof := cat.DB.Resolve(sha1, Overrides{}).Profile
		if prof.Platform != chip8.TwoPageChip8 {
			t.Errorf("%s: platform %s, want twoPageChip8", sha1, prof.Platform)
		}

		// Sergey Naydenov's compiled demos store back to the address they
		// loaded from (A684 F065 7001 F055), so FX55/FX65 must leave I alone.
		naydenov := sha1 == "200b313e4d4c1970641142cc7ff578d7956b93da" || sha1 == "af98ee11adae28a6153cae8e4c16afa00f861907"
		if prof.Quirks.MemoryLeaveIUnchanged != naydenov {
			t.Errorf("%s: memoryLeaveIUnchanged = %v, want %v", sha1, prof.Quirks.MemoryLeaveIUnchanged, naydenov)
		}

		if len(cat.Sources.ROMs[sha1]) == 0 {
			t.Errorf("%s: no download location in sources.json", sha1)
		}
	}
}

// checkPlayable checks that a ROM is in the catalogue and resolves to a
// platform Chippie can play.
func checkPlayable(t *testing.T, cat *Catalogue, sha1 string) {
	t.Helper()

	rom, ok := cat.DB.ROM(sha1)
	if !ok {
		t.Errorf("%s not in the catalogue", sha1)

		return
	}

	if p := cat.DB.Resolve(sha1, Overrides{}).Profile.Platform; !p.Implemented() {
		t.Errorf("%s: platform %s isn't playable", rom.Program.Title, p)
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

func TestOverlayChangesProgramFields(t *testing.T) {
	t.Parallel()

	db := loadSnapshot(t).DB
	o := mustOverlay(t, `{"change": [{"upstream": "pending", "sha1": "`+glitchGhost+`",
		"authors": ["A", "B"], "release": "2015"}]}`)

	status, err := db.ApplyOverlay(o)
	if err != nil || status[0].Done || !strings.Contains(status[0].What, "authors, release") {
		t.Fatalf("change: %v %+v", err, status)
	}

	if rom, _ := db.ROM(glitchGhost); rom.Program.Release != "2015" || len(rom.Program.Authors) != 2 {
		t.Errorf("program not changed: %+v", rom.Program)
	}

	if status, _ = db.ApplyOverlay(o); !status[0].Done {
		t.Error("applied twice: not reported done")
	}
}

// TestPruneOverlayKeepsFormatting checks that pruning nothing gives back
// overlay.json byte for byte, so a pruned file differs only by the entries
// removed.
func TestPruneOverlayKeepsFormatting(t *testing.T) {
	t.Parallel()

	got, err := PruneOverlay(overlayJSON, nil)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != string(overlayJSON) {
		t.Error("PruneOverlay changed overlay.json's formatting; keep it indented like json.MarshalIndent")
	}
}

// TestPruneOverlay removes the done entries, including a change applied
// after the additions (so its status comes out of order).
func TestPruneOverlay(t *testing.T) {
	t.Parallel()

	const dinoRun = "1a7a2138161b06e083722f978f8cbaaa1ad5344d" // upstream has it

	data := []byte(`{
  "add": [
    {"upstream": "` + upstream + `", "program": {"title": "Dino Run",
      "roms": {"` + dinoRun + `": {"file": "dinorun.ch8", "platforms": ["originalChip8"]}}}},
    {"upstream": "pending", "program": {"title": "New Game",
      "roms": {"` + newHash + `": {"file": "n.ch8", "platforms": ["originalChip8"]}}}}
  ],
  "change": [
    {"upstream": "` + upstream + `", "sha1": "` + dinoRun + `", "title": "Dino Run"},
    {"upstream": "pending", "sha1": "` + newHash + `", "tickrate": 30},
    {"upstream": "pending", "sha1": "` + newHash + `", "tickrate": 30}
  ]
}`)

	db := snapshotDB(t)

	status, err := db.ApplyOverlay(mustOverlay(t, string(data)))
	if err != nil {
		t.Fatal(err)
	}

	pruned, err := PruneOverlay(data, status)
	if err != nil {
		t.Fatal(err)
	}

	o := mustOverlay(t, string(pruned))

	if len(o.Platforms) != 0 || len(o.Add) != 1 || o.Add[0].Program.Title != "New Game" ||
		len(o.Change) != 1 || o.Change[0].SHA1 != newHash {
		t.Fatalf("pruned overlay = %s, want only New Game and one change to it", pruned)
	}

	if !strings.Contains(string(pruned), `"platforms": []`) {
		t.Errorf("an empty section should stay [], got:\n%s", pruned)
	}

	again, err := snapshotDB(t).ApplyOverlay(o)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range again {
		if s.Done {
			t.Errorf("%s still done after pruning", s.What)
		}
	}
}

// snapshotDB parses the committed snapshot without the embedded overlay.
func snapshotDB(t *testing.T) *DB {
	t.Helper()

	f, err := SnapshotFiles()
	if err != nil {
		t.Fatal(err)
	}

	db, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	return db
}
