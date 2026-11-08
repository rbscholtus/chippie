package catalogue

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// archiveProgram is one entry of chip8Archive's programs.json, keyed by the
// program's id (also its ROM's file name).
type archiveProgram struct {
	Title    string         `json:"title"`
	Platform string         `json:"platform"` // "chip8", "schip" or "xochip"
	Options  archiveOptions `json:"options"`
}

// archiveOptions are Octo's run options. Colours become the ROM's palette.
type archiveOptions struct {
	Tickrate        flexInt `json:"tickrate"`
	FillColor       string  `json:"fillColor"`
	FillColor2      string  `json:"fillColor2"`
	BlendColor      string  `json:"blendColor"`
	BackgroundColor string  `json:"backgroundColor"`
	BuzzColor       string  `json:"buzzColor"`
	QuietColor      string  `json:"quietColor"`
}

// flexInt is a number that chip8Archive sometimes writes as a string.
type flexInt int

func (n *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*n = 0

		return nil
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("not a number: %s: %w", b, err)
	}

	*n = flexInt(v)

	return nil
}

// archiveFamilies maps chip8Archive's platform names to the chip-8-database
// platforms they cover.
var archiveFamilies = map[string][]string{ //nolint:gochecknoglobals // read-only table
	"chip8":  {"originalChip8", "hybridVIP", "modernChip8", "chip8x"},
	"schip":  {"chip48", "superchip1", "superchip"},
	"xochip": {"xochip"},
}

// Conflict is a disagreement between chip8Archive and chip-8-database.
// chip-8-database wins; conflicts are reported by catalogue-gen as
// candidates for upstream fixes (README §10, §18).
type Conflict struct {
	SHA1    string
	Title   string
	Field   string
	DB      string
	Archive string
}

func (c Conflict) String() string {
	return fmt.Sprintf("%s (%s): %s: chip-8-database %q, chip8Archive %q", c.Title, c.SHA1, c.Field, c.DB, c.Archive)
}

// MergeArchive fills in metadata chip-8-database lacks from chip8Archive's
// programs.json: the tick rate and colours. hashes maps archive ids to ROM
// SHA-1s (written by catalogue-gen). chip-8-database wins wherever both have
// a value; disagreements are returned, sorted.
func (db *DB) MergeArchive(programsJSON, hashesJSON []byte) ([]Conflict, error) {
	var (
		programs map[string]archiveProgram
		hashes   map[string]string
	)

	if err := json.Unmarshal(programsJSON, &programs); err != nil {
		return nil, fmt.Errorf("%w: chip8Archive programs.json: %w", ErrInvalid, err)
	}

	if err := json.Unmarshal(hashesJSON, &hashes); err != nil {
		return nil, fmt.Errorf("%w: chip8Archive hashes: %w", ErrInvalid, err)
	}

	var conflicts []Conflict

	for id, ap := range programs {
		rom, ok := db.ROM(hashes[id])
		if !ok {
			continue
		}

		conflicts = append(conflicts, mergeROM(rom, ap)...)
	}

	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].Title != conflicts[j].Title {
			return conflicts[i].Title < conflicts[j].Title
		}

		return conflicts[i].Field < conflicts[j].Field
	})

	return conflicts, nil
}

func mergeROM(rom *ROM, ap archiveProgram) []Conflict {
	var out []Conflict

	conflict := func(field, db, archive string) {
		out = append(out, Conflict{SHA1: rom.SHA1, Title: rom.Program.Title, Field: field, DB: db, Archive: archive})
	}

	o := ap.Options

	switch {
	case rom.Tickrate == 0 && o.Tickrate > 0:
		rom.Tickrate = int(o.Tickrate)
	case rom.Tickrate > 0 && o.Tickrate > 0 && rom.Tickrate != int(o.Tickrate):
		conflict("tickrate", strconv.Itoa(rom.Tickrate), strconv.Itoa(int(o.Tickrate)))
	}

	if rom.Colors == nil && o.FillColor != "" && o.BackgroundColor != "" {
		rom.Colors = &Colors{
			Pixels:  []string{o.BackgroundColor, o.FillColor, o.FillColor2, o.BlendColor},
			Buzzer:  o.BuzzColor,
			Silence: o.QuietColor,
		}
	}

	if family, ok := archiveFamilies[ap.Platform]; ok && len(rom.Platforms) > 0 && !overlaps(rom.Platforms, family) {
		conflict("platform", strings.Join(rom.Platforms, ","), ap.Platform)
	}

	return out
}

func overlaps(a, b []string) bool {
	for _, x := range a {
		if slices.Contains(b, x) {
			return true
		}
	}

	return false
}
