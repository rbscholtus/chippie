package catalogue

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rbscholtus/chippie/chip8"
)

// Hashes from the snapshot used below.
const (
	glitchGhost  = "2cdcb3c29a5f013a991db5909ca8e18e27b3c42b" // modernChip8, tickrate 200
	turnover77   = "d03f27f85a1cf68465e0853cc0c4abee4a94a4e5" // superchip, xochip
	superNeatBoy = "440c5fbe9f5f840e76c308738fb0d37772d66674" // xochip, quirkyPlatforms: wrap off
	ibmTest      = "b9bbc12cee3f7b9d3b1f69161f7d7a2d86953379" // Timendus test 2
)

func loadSnapshot(tb testing.TB) *Catalogue {
	tb.Helper()

	c, err := LoadSnapshot()
	if err != nil {
		tb.Fatal(err)
	}

	return c
}

func TestSnapshotLoads(t *testing.T) {
	t.Parallel()

	c := loadSnapshot(t)

	if len(c.DB.Programs) < 400 || c.DB.ROMCount() < 500 || len(c.DB.Platforms) < 9 {
		t.Errorf("snapshot looks short: %d programs, %d ROMs, %d platforms",
			len(c.DB.Programs), c.DB.ROMCount(), len(c.DB.Platforms))
	}

	if n := len(c.Sources.ROMs); n < 200 {
		t.Errorf("only %d downloadable ROMs in sources.json", n)
	}

	// Every ROM in the manifest is in the database.
	for hash := range c.Sources.ROMs {
		if _, ok := c.DB.ROM(hash); !ok {
			t.Errorf("sources.json lists %s, which isn't in the database", hash)
		}
	}
}

// TestChip8DefaultsMatchDatabase catches drift between chip8's built-in
// platform table and the database's platforms.json.
func TestChip8DefaultsMatchDatabase(t *testing.T) {
	t.Parallel()

	db := loadSnapshot(t).DB

	for id, p := range db.Platforms {
		cp, ok := chip8.ParsePlatform(id)
		if !ok {
			t.Errorf("platforms.json has %q, which chip8 doesn't know", id)

			continue
		}

		var want chip8.Quirks
		applyQuirks(&want, p.Quirks)

		if got := chip8.DefaultQuirks(cp); got != want {
			t.Errorf("%s quirks: chip8 %+v, database %+v", id, got, want)
		}

		if cp.DefaultTicks() != p.Tickrate {
			t.Errorf("%s tickrate: chip8 %d, database %d", id, cp.DefaultTicks(), p.Tickrate)
		}
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()

	db := loadSnapshot(t).DB

	tests := []struct {
		name     string
		hash     string
		o        Overrides
		platform string
		ticks    int
		check    func(chip8.Quirks) bool
	}{
		{"known ROM: platform and tickrate", glitchGhost, Overrides{}, "modernChip8", 200, nil},
		{"hash in upper case", strings.ToUpper(glitchGhost), Overrides{}, "modernChip8", 200, nil},
		{
			"first platform when none is implemented", turnover77,
			Overrides{},
			"superchip", 200,
			func(q chip8.Quirks) bool { return q.Shift && q.Jump && q.MemoryLeaveIUnchanged },
		},
		{
			"quirkyPlatforms override", superNeatBoy,
			Overrides{},
			"xochip", 1000,
			func(q chip8.Quirks) bool { return !q.Wrap }, // xochip wraps by default
		},
		{"unknown ROM: modern CHIP-8", strings.Repeat("0", 40), Overrides{}, "modernChip8", 12, nil},
		{
			"user picks a platform", glitchGhost,
			Overrides{Platform: "originalChip8"},
			"originalChip8", 200,
			func(q chip8.Quirks) bool { return q.VBlank && q.Logic },
		},
		{
			"user quirk and speed", glitchGhost,
			Overrides{Quirks: map[string]bool{"shift": true}, Tickrate: 30},
			"modernChip8", 30, func(q chip8.Quirks) bool { return q.Shift },
		},
		{"unknown override platform is ignored", glitchGhost, Overrides{Platform: "nope"}, "modernChip8", 200, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := db.Resolve(tt.hash, tt.o)

			if r.PlatformID != tt.platform || r.Profile.Platform.String() != tt.platform {
				t.Errorf("platform %s (%s), want %s", r.PlatformID, r.Profile.Platform, tt.platform)
			}

			if r.Profile.TicksPerFrame != tt.ticks {
				t.Errorf("ticks %d, want %d", r.Profile.TicksPerFrame, tt.ticks)
			}

			if tt.check != nil && !tt.check(r.Profile.Quirks) {
				t.Errorf("quirks %+v", r.Profile.Quirks)
			}
		})
	}
}

func TestResolvePrefersImplementedPlatform(t *testing.T) {
	t.Parallel()

	db := loadSnapshot(t).DB
	rom := &ROM{SHA1: "abc", Platforms: []string{"superchip", "modernChip8"}, Program: &Program{}}
	db.roms["abc"] = rom

	if got := db.Resolve("abc", Overrides{}).PlatformID; got != "modernChip8" {
		t.Errorf("platform %s, want the implemented modernChip8", got)
	}
}

func TestOverridesForRoundTrip(t *testing.T) {
	t.Parallel()

	db := loadSnapshot(t).DB

	if o := db.OverridesFor(glitchGhost, db.Resolve(glitchGhost, Overrides{}).Profile); !o.Empty() {
		t.Errorf("recommended profile gave overrides %+v", o)
	}

	for _, want := range []Overrides{
		{Tickrate: 30},
		{Quirks: map[string]bool{"shift": true, "logic": true}},
		{Platform: "originalChip8"},
		{Platform: "originalChip8", Quirks: map[string]bool{"vblank": false}, Tickrate: 1000},
	} {
		p := db.Resolve(glitchGhost, want).Profile
		got := db.OverridesFor(glitchGhost, p)

		if db.Resolve(glitchGhost, got).Profile != p {
			t.Errorf("%+v: OverridesFor gave %+v, which resolves differently", want, got)
		}
	}
}

func TestQuirkMapRoundTrip(t *testing.T) {
	t.Parallel()

	q := chip8.Quirks{Shift: true, VBlank: true}

	var back chip8.Quirks
	applyQuirks(&back, QuirkMap(q))

	if back != q {
		t.Errorf("round trip %+v, want %+v", back, q)
	}

	if !(Overrides{}).Empty() || (Overrides{Tickrate: 1}).Empty() {
		t.Error("Overrides.Empty is wrong")
	}
}

// files returns a small valid database, which tests then break.
func files(t *testing.T) Files {
	t.Helper()

	return Files{
		Platforms: []byte(`[{"id":"modernChip8","name":"Modern","defaultTickrate":12,"quirks":{"shift":false}}]`),
		Quirks: []byte(`[{"id":"shift"},{"id":"memoryIncrementByX"},{"id":"memoryLeaveIUnchanged"},` +
			`{"id":"wrap"},{"id":"jump"},{"id":"vblank"},{"id":"logic"}]`),
		Programs: []byte(`[{"title":"A","roms":{"aaaa":{"file":"a.ch8","platforms":["modernChip8"]}}}]`),
		Hashes:   []byte(`{"aaaa":0}`),
	}
}

func TestParseValidates(t *testing.T) {
	t.Parallel()

	if _, err := Parse(files(t)); err != nil {
		t.Fatalf("valid database: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Files)
	}{
		{"bad JSON", func(f *Files) { f.Programs = []byte("[") }},
		{"unknown quirk on a platform", func(f *Files) {
			f.Platforms = []byte(`[{"id":"modernChip8","defaultTickrate":12,"quirks":{"nope":true}}]`)
		}},
		{"quirks.json lacks a quirk", func(f *Files) { f.Quirks = []byte(`[{"id":"shift"}]`) }},
		{"ROM on an unknown platform", func(f *Files) {
			f.Programs = []byte(`[{"title":"A","roms":{"aaaa":{"platforms":["nope"]}}}]`)
		}},
		{"quirks for an unknown platform", func(f *Files) {
			f.Programs = []byte(`[{"title":"A","roms":{"aaaa":{"quirkyPlatforms":{"nope":{"shift":true}}}}}]`)
		}},
		{"hash points at the wrong program", func(f *Files) { f.Hashes = []byte(`{"aaaa":5}`) }},
		{"hash missing from sha1-hashes", func(f *Files) { f.Hashes = []byte(`{}`) }},
		{"platform without tickrate", func(f *Files) {
			f.Platforms = []byte(`[{"id":"modernChip8","quirks":{}}]`)
		}},
	}

	for _, tt := range tests {
		f := files(t)
		tt.mutate(&f)

		if _, err := Parse(f); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err=%v, want ErrInvalid", tt.name, err)
		}
	}
}

func TestSources(t *testing.T) {
	t.Parallel()

	manifest := map[string]any{
		"version": 1,
		"sources": map[string]any{"a": map[string]string{"name": "A"}, "chippie": map[string]string{"name": "Hosted"}},
		"roms": map[string]any{"abcd": []map[string]string{
			{"source": "chippie", "url": "roms/x.ch8"},
			{"source": "a", "url": "https://example.com/x.ch8"},
		}},
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	s, err := ParseSources(data)
	if err != nil {
		t.Fatal(err)
	}

	urls := s.URLs("ABCD")
	if len(urls) != 2 || urls[0].Source != "a" || urls[1].Source != HostedSource {
		t.Errorf("URLs = %+v, want the original source first and hosted last", urls)
	}

	if !s.Available("abcd") || s.Available("ffff") {
		t.Error("Available is wrong")
	}

	for name, bad := range map[string]string{
		"bad JSON":       `{`,
		"wrong version":  `{"version":2}`,
		"unknown source": `{"version":1,"sources":{},"roms":{"ab":[{"source":"x","url":"https://e/x"}]}}`,
		"not https":      `{"version":1,"sources":{"a":{}},"roms":{"ab":[{"source":"a","url":"http://e/x"}]}}`,
	} {
		if _, err := ParseSources([]byte(bad)); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err=%v, want ErrInvalid", name, err)
		}
	}
}

func TestMergeArchive(t *testing.T) {
	t.Parallel()

	db, err := Parse(files(t))
	if err != nil {
		t.Fatal(err)
	}

	programs := `{"a":{"title":"A","platform":"xochip",` +
		`"options":{"tickrate":"20","fillColor":"#fff","backgroundColor":"#000"}}}`

	conflicts, err := db.MergeArchive([]byte(programs), []byte(`{"a":"aaaa"}`))
	if err != nil {
		t.Fatal(err)
	}

	rom, _ := db.ROM("aaaa")
	if rom.Tickrate != 20 || rom.Colors == nil || rom.Colors.Pixels[1] != "#fff" {
		t.Errorf("archive values not merged: tickrate %d colors %+v", rom.Tickrate, rom.Colors)
	}

	if len(conflicts) != 1 || conflicts[0].Field != "platform" || !strings.Contains(conflicts[0].String(), "xochip") {
		t.Errorf("conflicts = %v, want one platform conflict", conflicts)
	}

	// The database wins: a different tickrate is reported, not applied.
	rom.Tickrate = 50

	conflicts, _ = db.MergeArchive([]byte(programs), []byte(`{"a":"aaaa"}`))
	if rom.Tickrate != 50 || len(conflicts) != 2 {
		t.Errorf("tickrate %d, conflicts %v", rom.Tickrate, conflicts)
	}

	if _, err := db.MergeArchive([]byte(`{`), []byte(`{}`)); !errors.Is(err, ErrInvalid) {
		t.Errorf("bad archive JSON: err=%v", err)
	}
}

func BenchmarkResolveProfile(b *testing.B) {
	db := loadSnapshot(b).DB

	b.ReportAllocs()

	for b.Loop() {
		db.Resolve(ibmTest, Overrides{})
	}
}
