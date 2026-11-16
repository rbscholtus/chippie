package catalogue

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/rbscholtus/chippie/chip8"
)

// The overlay is Chippie's pending additions and corrections to
// chip-8-database (catalogue/overlay.json), applied on top of the database
// until upstream has them (DESIGN §10). It can add platforms, programs and
// ROMs, and change ROMs. Every entry names where it was
// proposed upstream. Where upstream already has what an entry adds or sets,
// the entry does nothing and its status says so: then it can be deleted.

//go:embed overlay.json
var overlayJSON []byte

// pendingUpstream marks an entry not yet proposed upstream.
const pendingUpstream = "pending"

var (
	sha1Pattern     = regexp.MustCompile(`^[0-9a-f]{40}$`)
	upstreamPattern = regexp.MustCompile(`^https://github\.com/[\w.-]+/[\w.-]+/(pull|issues)/\d+$`)
)

// Overlay is a parsed overlay.json.
type Overlay struct {
	Platforms []OverlayPlatform `json:"platforms"`
	Add       []OverlayAdd      `json:"add"`
	Change    []OverlayChange   `json:"change"`
}

// OverlayPlatform adds a platform, in platforms.json's shape. Chippie must
// know its id (chip8.ParsePlatform), so its ROMs can run.
type OverlayPlatform struct {
	Upstream string      `json:"upstream"` // the upstream PR or issue, or "pending"
	Platform rawPlatform `json:"platform"`
}

// OverlayAdd adds a program, or (with To) more ROMs to an existing one.
type OverlayAdd struct {
	Upstream string `json:"upstream"` // the upstream PR or issue, or "pending"
	// To, if set, is the SHA-1 of a ROM already in the database: the new
	// ROMs join its program instead of a new one.
	To      string     `json:"to,omitempty"`
	Program rawProgram `json:"program"`
}

// OverlayChange corrects fields of a ROM in the database (or, for Title,
// Authors and Release, its program). Only the fields given change.
type OverlayChange struct {
	Upstream        string                     `json:"upstream"`
	SHA1            string                     `json:"sha1"`
	Title           string                     `json:"title,omitempty"`
	Authors         []string                   `json:"authors,omitempty"`
	Release         string                     `json:"release,omitempty"`
	Platforms       []string                   `json:"platforms,omitempty"`
	QuirkyPlatforms map[string]map[string]bool `json:"quirkyPlatforms,omitempty"`
	Tickrate        int                        `json:"tickrate,omitempty"`
	ScreenRotation  int                        `json:"screenRotation,omitempty"` // 0: not changed
}

// OverlayStatus is what one entry did.
type OverlayStatus struct {
	Upstream string
	What     string // e.g. "add Dino Run", "change Mr: title"
	Done     bool   // upstream already has it: the entry can be deleted
	Entry    OverlayEntry
}

// OverlayEntry is where an entry is in overlay.json: its section and its
// index in that section.
type OverlayEntry struct {
	Section string // "platforms", "add" or "change"
	Index   int
}

// The overlay.json sections, as in OverlayEntry.
const (
	sectionPlatforms = "platforms"
	sectionAdd       = "add"
	sectionChange    = "change"
)

// ParseOverlay reads and checks an overlay.
func ParseOverlay(data []byte) (*Overlay, error) {
	var o Overlay
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, fmt.Errorf("%w: overlay: %w", ErrInvalid, err)
	}

	check := func(upstream, what string) error {
		if upstream != pendingUpstream && !upstreamPattern.MatchString(upstream) {
			return fmt.Errorf("%w: overlay %s: upstream %q is not a GitHub PR or issue link, or %q",
				ErrInvalid, what, upstream, pendingUpstream)
		}

		return nil
	}

	for _, p := range o.Platforms {
		if err := check(p.Upstream, p.Platform.ID); err != nil {
			return nil, err
		}

		if _, ok := chip8.ParsePlatform(p.Platform.ID); !ok || p.Platform.Name == "" {
			return nil, fmt.Errorf("%w: overlay: platform %q needs a name and an id Chippie knows",
				ErrInvalid, p.Platform.ID)
		}
	}

	for _, a := range o.Add {
		if err := check(a.Upstream, a.Program.Title); err != nil {
			return nil, err
		}

		if len(a.Program.ROMs) == 0 || (a.To == "" && a.Program.Title == "") {
			return nil, fmt.Errorf("%w: overlay: an add needs ROMs, and a title unless it has to", ErrInvalid)
		}

		for hash := range a.Program.ROMs {
			if !sha1Pattern.MatchString(hash) {
				return nil, fmt.Errorf("%w: overlay %s: %q is not a lower-case SHA-1", ErrInvalid, a.Program.Title, hash)
			}
		}
	}

	for _, c := range o.Change {
		if err := check(c.Upstream, c.SHA1); err != nil {
			return nil, err
		}

		if !sha1Pattern.MatchString(c.SHA1) {
			return nil, fmt.Errorf("%w: overlay change: %q is not a lower-case SHA-1", ErrInvalid, c.SHA1)
		}
	}

	return &o, nil
}

// ApplyOverlay adds and changes what the overlay says, skipping whatever
// upstream already has, and reports each entry's status. New platforms go
// first (the ROMs may need them), then changes to ROMs already in the
// database (so a renamed program shows its new name), then the additions,
// then changes to ROMs the overlay added.
func (db *DB) ApplyOverlay(o *Overlay) ([]OverlayStatus, error) {
	var (
		status []OverlayStatus
		later  []int // indexes of changes to ROMs the overlay adds
	)

	for i, p := range o.Platforms {
		st, err := db.applyPlatform(p)
		if err != nil {
			return nil, err
		}

		st.Entry = OverlayEntry{Section: sectionPlatforms, Index: i}
		status = append(status, st)
	}

	change := func(i int) error {
		st, err := db.applyChange(o.Change[i])
		st.Entry = OverlayEntry{Section: sectionChange, Index: i}
		status = append(status, st)

		return err
	}

	for i, c := range o.Change {
		if _, ok := db.roms[c.SHA1]; !ok {
			later = append(later, i)

			continue
		}

		if err := change(i); err != nil {
			return nil, err
		}
	}

	for i, a := range o.Add {
		st, err := db.applyAdd(a)
		if err != nil {
			return nil, err
		}

		st.Entry = OverlayEntry{Section: sectionAdd, Index: i}
		status = append(status, st)
	}

	for _, i := range later {
		if err := change(i); err != nil {
			return nil, err
		}
	}

	return status, nil
}

// applyPlatform adds a platform, unless upstream has one with its id.
func (db *DB) applyPlatform(p OverlayPlatform) (OverlayStatus, error) {
	st := OverlayStatus{Upstream: p.Upstream, What: "add platform " + p.Platform.ID}

	if db.Platforms[p.Platform.ID] != nil {
		st.Done = true

		return st, nil
	}

	plat := newPlatform(p.Platform)
	if err := checkPlatform(plat, db.quirks); err != nil {
		return st, fmt.Errorf("overlay: %w", err)
	}

	db.Platforms[plat.ID] = plat

	return st, nil
}

func (db *DB) applyAdd(a OverlayAdd) (OverlayStatus, error) {
	prog := &Program{
		Title: a.Program.Title, Authors: a.Program.Authors,
		Release: a.Program.Release, Description: a.Program.Description,
	}
	st := OverlayStatus{Upstream: a.Upstream, What: "add " + a.Program.Title}

	if a.To != "" {
		to, ok := db.roms[a.To]
		if !ok {
			return st, fmt.Errorf("%w: overlay: add to %s, which isn't in the database", ErrInvalid, a.To)
		}

		prog = to.Program
		st.What = "add ROMs to " + prog.Title
	}

	added := 0

	for hash, rr := range a.Program.ROMs {
		if _, ok := db.roms[hash]; ok {
			continue // upstream has it now
		}

		rom := newROM(hash, rr, prog)
		if err := db.checkROM(rom); err != nil {
			return st, err
		}

		prog.ROMs = append(prog.ROMs, rom)
		db.roms[hash] = rom
		added++
	}

	st.Done = added == 0

	if a.To == "" && added > 0 {
		db.Programs = append(db.Programs, prog)
	}

	return st, nil
}

func (db *DB) applyChange(c OverlayChange) (OverlayStatus, error) {
	rom, ok := db.roms[c.SHA1]
	if !ok {
		return OverlayStatus{}, fmt.Errorf("%w: overlay: change %s, which isn't in the database", ErrInvalid, c.SHA1)
	}

	var fields []string

	set := func(name string, given bool, same bool, apply func()) {
		if given && !same {
			apply()

			fields = append(fields, name)
		}
	}

	set("title", c.Title != "", c.Title == rom.Program.Title, func() { rom.Program.Title = c.Title })
	set("authors", c.Authors != nil, reflect.DeepEqual(c.Authors, rom.Program.Authors),
		func() { rom.Program.Authors = c.Authors })
	set("release", c.Release != "", c.Release == rom.Program.Release, func() { rom.Program.Release = c.Release })
	set("platforms", c.Platforms != nil, reflect.DeepEqual(c.Platforms, rom.Platforms),
		func() { rom.Platforms = c.Platforms })
	set("quirkyPlatforms", c.QuirkyPlatforms != nil, reflect.DeepEqual(c.QuirkyPlatforms, rom.QuirkyPlatforms),
		func() { rom.QuirkyPlatforms = c.QuirkyPlatforms })
	set("tickrate", c.Tickrate > 0, c.Tickrate == rom.Tickrate, func() { rom.Tickrate = c.Tickrate })
	set("screenRotation", c.ScreenRotation != 0, c.ScreenRotation == rom.ScreenRotation,
		func() { rom.ScreenRotation = c.ScreenRotation })

	st := OverlayStatus{
		Upstream: c.Upstream, What: "change " + rom.Program.Title, Done: len(fields) == 0,
	}

	if !st.Done {
		st.What += ": " + strings.Join(fields, ", ")
	}

	return st, db.checkROM(rom)
}

// checkROM checks a ROM's platforms, quirks and rotation against the
// database (for its own ROMs and the overlay's).
func (db *DB) checkROM(rom *ROM) error {
	if !validRotation(rom.ScreenRotation) {
		return fmt.Errorf("%w: ROM %s: screenRotation %d", ErrInvalid, rom.SHA1, rom.ScreenRotation)
	}

	for _, p := range rom.Platforms {
		if db.Platforms[p] == nil {
			return fmt.Errorf("%w: ROM %s: unknown platform %q", ErrInvalid, rom.SHA1, p)
		}
	}

	for p, qs := range rom.QuirkyPlatforms {
		if db.Platforms[p] == nil {
			return fmt.Errorf("%w: ROM %s: quirks for unknown platform %q", ErrInvalid, rom.SHA1, p)
		}

		for q := range qs {
			if !db.quirks[q] {
				return fmt.Errorf("%w: ROM %s: unknown quirk %q", ErrInvalid, rom.SHA1, q)
			}
		}
	}

	return nil
}

// newROM builds a ROM from its programs.json entry.
func newROM(hash string, rr rawROM, prog *Program) *ROM {
	return &ROM{
		SHA1: strings.ToLower(hash), File: rr.File, Platforms: rr.Platforms,
		QuirkyPlatforms: rr.QuirkyPlatforms, Tickrate: rr.Tickrate, Colors: rr.Colors,
		Keys: rr.Keys, TouchInputMode: rr.TouchInputMode, ScreenRotation: rr.ScreenRotation, Program: prog,
	}
}

// PruneOverlay returns the overlay in data without the entries that status
// reports done, so upstream's merges empty it (DESIGN §10). The entries it
// keeps are unchanged, and so is their formatting, as long as data is
// indented like json.MarshalIndent with two spaces (as overlay.json is).
func PruneOverlay(data []byte, status []OverlayStatus) ([]byte, error) {
	// The entries stay raw, so their keys keep their order; the struct
	// keeps the sections' order.
	var raw struct {
		Platforms []json.RawMessage `json:"platforms"`
		Add       []json.RawMessage `json:"add"`
		Change    []json.RawMessage `json:"change"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: overlay: %w", ErrInvalid, err)
	}

	done := map[OverlayEntry]bool{}

	for _, s := range status {
		if s.Done {
			done[s.Entry] = true
		}
	}

	keep := func(section string, entries []json.RawMessage) []json.RawMessage {
		kept := []json.RawMessage{} // never null, even when empty

		for i, e := range entries {
			if !done[OverlayEntry{Section: section, Index: i}] {
				kept = append(kept, e)
			}
		}

		return kept
	}

	raw.Platforms = keep(sectionPlatforms, raw.Platforms)
	raw.Add = keep(sectionAdd, raw.Add)
	raw.Change = keep(sectionChange, raw.Change)

	var b strings.Builder

	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	if err := enc.Encode(raw); err != nil {
		return nil, fmt.Errorf("encode overlay: %w", err)
	}

	return []byte(b.String()), nil
}

// EmbeddedOverlay is the overlay compiled into the app.
func EmbeddedOverlay() (*Overlay, error) { return ParseOverlay(overlayJSON) }
