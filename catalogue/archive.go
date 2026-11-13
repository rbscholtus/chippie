package catalogue

import (
	"encoding/json"
	"fmt"
	"maps"
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

	// Octo's compatibility switches. A program from before a switch existed
	// doesn't list it, which means Octo's default: off.
	ShiftQuirks     flexBool `json:"shiftQuirks"`
	LoadStoreQuirks flexBool `json:"loadStoreQuirks"`
	JumpQuirks      flexBool `json:"jumpQuirks"`
	ClipQuirks      flexBool `json:"clipQuirks"`
	LogicQuirks     flexBool `json:"logicQuirks"`
	VBlankQuirks    flexBool `json:"vBlankQuirks"`
}

// flexBool is a flag that chip8Archive writes as true/false or 1/0.
type flexBool bool

func (f *flexBool) UnmarshalJSON(b []byte) error {
	switch string(b) {
	case "true", "1":
		*f = true
	case "false", "0", "null":
		*f = false
	default:
		return fmt.Errorf("%w: not a flag: %s", ErrInvalid, b)
	}

	return nil
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
// programs.json: the tick rate, colours and, for SUPER-CHIP and XO-CHIP
// games, the quirks. hashes maps archive ids to ROM
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

	mergeQuirks(rom, ap, conflict)

	return out
}

// mergeQuirks records the quirks the author ran the game with, for its
// SUPER-CHIP and XO-CHIP platforms. These games were written in Octo, which
// has none of the HP48's quirks unless they are asked for, so the platform
// defaults in platforms.json break many of them. CHIP-8 games are left to
// chip-8-database, whose per-platform defaults describe the real machines.
func mergeQuirks(rom *ROM, ap archiveProgram, conflict func(field, db, archive string)) {
	if ap.Platform == "chip8" {
		return
	}

	quirks := ap.Options.quirks()

	for _, p := range rom.Platforms {
		if !slices.Contains(archiveFamilies[ap.Platform], p) {
			continue
		}

		if rom.QuirkyPlatforms == nil {
			rom.QuirkyPlatforms = map[string]map[string]bool{}
		}

		if rom.QuirkyPlatforms[p] == nil {
			rom.QuirkyPlatforms[p] = map[string]bool{}
		}

		for _, id := range slices.Sorted(maps.Keys(quirks)) {
			have, set := rom.QuirkyPlatforms[p][id]

			switch {
			case !set:
				rom.QuirkyPlatforms[p][id] = quirks[id]
			case have != quirks[id]:
				conflict("quirk "+id, strconv.FormatBool(have), strconv.FormatBool(quirks[id]))
			}
		}
	}
}

// quirks translates Octo's switches into quirks.json ids. Octo's load/store
// switch leaves I unchanged, and it has no "increment by X" behaviour; its
// clip switch is the opposite of the wrap quirk.
func (o archiveOptions) quirks() map[string]bool {
	return map[string]bool{
		quirkShift:                 bool(o.ShiftQuirks),
		quirkMemoryLeaveIUnchanged: bool(o.LoadStoreQuirks),
		quirkMemoryIncrementByX:    false,
		quirkJump:                  bool(o.JumpQuirks),
		quirkWrap:                  !bool(o.ClipQuirks),
		quirkLogic:                 bool(o.LogicQuirks),
		quirkVBlank:                bool(o.VBlankQuirks),
	}
}

func overlaps(a, b []string) bool {
	for _, x := range a {
		if slices.Contains(b, x) {
			return true
		}
	}

	return false
}
