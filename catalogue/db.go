// Package catalogue holds the game metadata: chip-8-database (platforms,
// quirks, programs, ROM hashes), merged with chip8Archive's own metadata,
// plus the sources manifest that says where each ROM can be downloaded. It
// resolves the profile a ROM runs with (README §7).
//
// An embedded snapshot makes it work offline; Refresh fetches a newer
// database in the background.
package catalogue

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrInvalid reports a database that doesn't parse or doesn't hang together.
var ErrInvalid = errors.New("catalogue: invalid database")

// Files are the raw chip-8-database JSON files.
type Files struct {
	Platforms []byte // platforms.json
	Quirks    []byte // quirks.json
	Programs  []byte // programs.json
	Hashes    []byte // sha1-hashes.json
}

// DB is a parsed, validated chip-8-database.
type DB struct {
	Platforms map[string]*Platform // by platforms.json id
	Programs  []*Program
	roms      map[string]*ROM // by lower-case SHA-1
}

// Platform is one entry of platforms.json.
type Platform struct {
	ID          string
	Name        string
	Tickrate    int
	Resolutions []string
	Quirks      map[string]bool
}

// Program is one entry of programs.json: a title with one or more ROMs.
type Program struct {
	Title       string
	Authors     []string
	Release     string
	Description string
	ROMs        []*ROM
}

// ROM is one ROM file of a program, identified by its SHA-1.
type ROM struct {
	SHA1            string
	File            string
	Platforms       []string                   // supported platforms, best first
	QuirkyPlatforms map[string]map[string]bool // per platform: quirks that differ from its defaults
	Tickrate        int                        // 0: use the platform's
	Colors          *Colors
	Keys            map[string]int // e.g. "up": 5
	TouchInputMode  string
	Program         *Program
}

// Colors are a ROM's preferred colours, as CSS colour strings.
type Colors struct {
	Pixels  []string `json:"pixels"`  // by plane bits: off, plane 1, plane 2, both
	Buzzer  string   `json:"buzzer"`  // background while the buzzer sounds
	Silence string   `json:"silence"` // background otherwise
}

// Raw JSON shapes.
type (
	rawPlatform struct {
		ID                 string          `json:"id"`
		Name               string          `json:"name"`
		DefaultTickrate    int             `json:"defaultTickrate"`
		DisplayResolutions []string        `json:"displayResolutions"`
		Quirks             map[string]bool `json:"quirks"`
	}
	rawQuirk struct {
		ID string `json:"id"`
	}
	rawROM struct {
		File            string                     `json:"file"`
		Platforms       []string                   `json:"platforms"`
		QuirkyPlatforms map[string]map[string]bool `json:"quirkyPlatforms"`
		Tickrate        int                        `json:"tickrate"`
		Colors          *Colors                    `json:"colors"`
		Keys            map[string]int             `json:"keys"`
		TouchInputMode  string                     `json:"touchInputMode"`
	}
	rawProgram struct {
		Title       string            `json:"title"`
		Authors     []string          `json:"authors"`
		Release     string            `json:"release"`
		Description string            `json:"description"`
		ROMs        map[string]rawROM `json:"roms"`
	}
)

// Parse parses and validates the database files.
func Parse(f Files) (*DB, error) {
	var (
		platforms []rawPlatform
		quirks    []rawQuirk
		programs  []rawProgram
		hashes    map[string]int
	)

	for _, part := range []struct {
		name string
		data []byte
		dst  any
	}{
		{"platforms.json", f.Platforms, &platforms},
		{"quirks.json", f.Quirks, &quirks},
		{"programs.json", f.Programs, &programs},
		{"sha1-hashes.json", f.Hashes, &hashes},
	} {
		if err := json.Unmarshal(part.data, part.dst); err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrInvalid, part.name, err)
		}
	}

	db := &DB{Platforms: make(map[string]*Platform, len(platforms)), roms: make(map[string]*ROM, len(hashes))}

	for _, p := range platforms {
		db.Platforms[p.ID] = &Platform{
			ID: p.ID, Name: p.Name, Tickrate: p.DefaultTickrate,
			Resolutions: p.DisplayResolutions, Quirks: p.Quirks,
		}
	}

	db.Programs = make([]*Program, len(programs))

	for i, rp := range programs {
		prog := &Program{Title: rp.Title, Authors: rp.Authors, Release: rp.Release, Description: rp.Description}
		db.Programs[i] = prog

		for hash, rr := range rp.ROMs {
			rom := &ROM{
				SHA1: strings.ToLower(hash), File: rr.File, Platforms: rr.Platforms,
				QuirkyPlatforms: rr.QuirkyPlatforms, Tickrate: rr.Tickrate, Colors: rr.Colors,
				Keys: rr.Keys, TouchInputMode: rr.TouchInputMode, Program: prog,
			}
			prog.ROMs = append(prog.ROMs, rom)
			db.roms[rom.SHA1] = rom
		}
	}

	if err := validate(db, quirks, hashes); err != nil {
		return nil, err
	}

	return db, nil
}

// validate checks that the files hang together: every quirk is known,
// every referenced platform exists, and sha1-hashes.json points each hash
// at the program that lists it.
func validate(db *DB, quirks []rawQuirk, hashes map[string]int) error {
	known := make(map[string]bool, len(quirks))
	for _, q := range quirks {
		known[q.ID] = true
	}

	for _, id := range quirkIDs {
		if !known[id] {
			return fmt.Errorf("%w: quirks.json lacks %q", ErrInvalid, id)
		}
	}

	for _, p := range db.Platforms {
		if p.ID == "" || p.Tickrate <= 0 {
			return fmt.Errorf("%w: platform %q incomplete", ErrInvalid, p.ID)
		}

		for q := range p.Quirks {
			if !known[q] {
				return fmt.Errorf("%w: platform %s: unknown quirk %q", ErrInvalid, p.ID, q)
			}
		}
	}

	for _, rom := range db.roms {
		for _, p := range rom.Platforms {
			if db.Platforms[p] == nil {
				return fmt.Errorf("%w: ROM %s: unknown platform %q", ErrInvalid, rom.SHA1, p)
			}
		}

		for p := range rom.QuirkyPlatforms {
			if db.Platforms[p] == nil {
				return fmt.Errorf("%w: ROM %s: quirks for unknown platform %q", ErrInvalid, rom.SHA1, p)
			}
		}
	}

	if len(hashes) != len(db.roms) {
		return fmt.Errorf("%w: sha1-hashes.json has %d hashes, programs.json %d ROMs", ErrInvalid, len(hashes), len(db.roms))
	}

	for hash, idx := range hashes {
		rom := db.roms[strings.ToLower(hash)]
		if rom == nil || idx < 0 || idx >= len(db.Programs) || db.Programs[idx] != rom.Program {
			return fmt.Errorf("%w: sha1-hashes.json: %s doesn't point at its program", ErrInvalid, hash)
		}
	}

	return nil
}

// ROM returns the ROM with the given SHA-1 (hex, any case).
func (db *DB) ROM(sha1 string) (*ROM, bool) {
	r, ok := db.roms[strings.ToLower(sha1)]

	return r, ok
}

// ROMCount returns the number of ROMs in the database.
func (db *DB) ROMCount() int { return len(db.roms) }
