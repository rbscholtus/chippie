package catalogue

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// The overlay is Chippie's pending additions and corrections to
// chip-8-database (catalogue/overlay.json), applied on top of the database
// until upstream has them (README §18). Every entry names where it was
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
	Add    []OverlayAdd    `json:"add"`
	Change []OverlayChange `json:"change"`
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
// its program). Only the fields given change.
type OverlayChange struct {
	Upstream        string                     `json:"upstream"`
	SHA1            string                     `json:"sha1"`
	Title           string                     `json:"title,omitempty"`
	Platforms       []string                   `json:"platforms,omitempty"`
	QuirkyPlatforms map[string]map[string]bool `json:"quirkyPlatforms,omitempty"`
	Tickrate        int                        `json:"tickrate,omitempty"`
}

// OverlayStatus is what one entry did.
type OverlayStatus struct {
	Upstream string
	What     string // e.g. "add Dino Run", "change Mr: title"
	Done     bool   // upstream already has it: the entry can be deleted
}

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
// upstream already has, and reports each entry's status. Changes to ROMs
// already in the database go first (so a renamed program shows its new
// name), then the additions, then changes to ROMs the overlay added.
func (db *DB) ApplyOverlay(o *Overlay) ([]OverlayStatus, error) {
	var (
		status []OverlayStatus
		later  []OverlayChange
	)

	change := func(c OverlayChange) error {
		st, err := db.applyChange(c)
		status = append(status, st)

		return err
	}

	for _, c := range o.Change {
		if _, ok := db.roms[c.SHA1]; !ok {
			later = append(later, c)

			continue
		}

		if err := change(c); err != nil {
			return nil, err
		}
	}

	for _, a := range o.Add {
		st, err := db.applyAdd(a)
		if err != nil {
			return nil, err
		}

		status = append(status, st)
	}

	for _, c := range later {
		if err := change(c); err != nil {
			return nil, err
		}
	}

	return status, nil
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
	set("platforms", c.Platforms != nil, reflect.DeepEqual(c.Platforms, rom.Platforms),
		func() { rom.Platforms = c.Platforms })
	set("quirkyPlatforms", c.QuirkyPlatforms != nil, reflect.DeepEqual(c.QuirkyPlatforms, rom.QuirkyPlatforms),
		func() { rom.QuirkyPlatforms = c.QuirkyPlatforms })
	set("tickrate", c.Tickrate > 0, c.Tickrate == rom.Tickrate, func() { rom.Tickrate = c.Tickrate })

	st := OverlayStatus{
		Upstream: c.Upstream, What: "change " + rom.Program.Title, Done: len(fields) == 0,
	}

	if !st.Done {
		st.What += ": " + strings.Join(fields, ", ")
	}

	return st, db.checkROM(rom)
}

// checkROM checks a ROM's platforms and quirks against the database.
func (db *DB) checkROM(rom *ROM) error {
	for _, p := range rom.Platforms {
		if db.Platforms[p] == nil {
			return fmt.Errorf("%w: overlay: ROM %s: unknown platform %q", ErrInvalid, rom.SHA1, p)
		}
	}

	for p, qs := range rom.QuirkyPlatforms {
		if db.Platforms[p] == nil {
			return fmt.Errorf("%w: overlay: ROM %s: quirks for unknown platform %q", ErrInvalid, rom.SHA1, p)
		}

		for q := range qs {
			if !db.quirks[q] {
				return fmt.Errorf("%w: overlay: ROM %s: unknown quirk %q", ErrInvalid, rom.SHA1, q)
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
		Keys: rr.Keys, TouchInputMode: rr.TouchInputMode, Program: prog,
	}
}

// EmbeddedOverlay is the overlay compiled into the app.
func EmbeddedOverlay() (*Overlay, error) { return ParseOverlay(overlayJSON) }
