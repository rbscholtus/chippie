// Package roms lists the ROMs Chippie can play before the catalogue exists
// (M4): the bundled CC0 games and the Timendus test suite. It loads them
// asynchronously, so the game loop never waits.
package roms

import (
	"context"
	"crypto/sha1" //nolint:gosec // G505: SHA-1 identifies ROMs, as in chip-8-database; not for security
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/rbscholtus/chippie/assets"
	"github.com/rbscholtus/chippie/chip8"
	"github.com/rbscholtus/chippie/platform"
)

// Errors from loading ROMs.
var (
	ErrNotFound     = errors.New("roms: no such ROM")
	ErrHashMismatch = errors.New("roms: ROM doesn't match its SHA-1")
	ErrTooLarge     = errors.New("roms: ROM too large")
	ErrBadManifest  = errors.New("roms: bad bundled manifest")
)

// maxROMSize bounds downloads and files: 64 KB is the largest memory of any
// supported platform.
const maxROMSize = 64 << 10

// Group says where an entry comes from.
type Group uint8

// Entry groups.
const (
	Games     Group = iota // bundled CC0 games, compiled in
	TestSuite              // Timendus test suite, loaded from beside the app
)

// Entry is one playable ROM.
type Entry struct {
	ID          string // "game/<id>" or "test/<name>"
	Title       string
	Authors     []string
	Release     string
	Description string
	Group       Group
	File        string
	SHA1        string
	Platform    chip8.Platform
	Ticks       int // instructions per frame
}

// Playable reports whether the entry's platform is implemented yet.
func (e Entry) Playable() bool { return e.Platform.Implemented() }

// Profile returns the profile to run the entry with.
func (e Entry) Profile() chip8.Profile {
	p := chip8.DefaultProfile(e.Platform)
	if e.Ticks > 0 {
		p.TicksPerFrame = e.Ticks
	}

	return p
}

// bundledEntry is one item in assets/roms/bundled.json.
type bundledEntry struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Authors     []string `json:"authors"`
	Release     string   `json:"release"`
	File        string   `json:"file"`
	SHA1        string   `json:"sha1"`
	Platform    string   `json:"platform"`
	Tickrate    int      `json:"tickrate"`
	Description string   `json:"description"`
}

// testSuite lists the Timendus test ROMs (tag v4.2). They run on modern
// CHIP-8, except the scrolling test, which needs SUPER-CHIP past its menu.
var testSuite = []Entry{ //nolint:gochecknoglobals // read-only table
	{
		Title: "1. CHIP-8 logo", File: "1-chip8-logo.ch8",
		SHA1: "30f27e5cee5b325fd1681ee98a14de60bfbe951f", Platform: chip8.ModernChip8,
	},
	{
		Title: "2. IBM logo", File: "2-ibm-logo.ch8",
		SHA1: "b9bbc12cee3f7b9d3b1f69161f7d7a2d86953379", Platform: chip8.ModernChip8,
	},
	{
		Title: "3. Corax+ opcodes", File: "3-corax+.ch8",
		SHA1: "b2dacf6d85785d6c2315ce449912c8a8a5954e2e", Platform: chip8.ModernChip8,
	},
	{
		Title: "4. Flags", File: "4-flags.ch8",
		SHA1: "55a6716dacc2f93dce3d39fb8d231083016a1cc0", Platform: chip8.ModernChip8,
	},
	{
		Title: "5. Quirks", File: "5-quirks.ch8",
		SHA1: "e2149cb836131a142ca7e2dc2f2283381ae5faaa", Platform: chip8.ModernChip8,
	},
	{
		Title: "6. Keypad", File: "6-keypad.ch8",
		SHA1: "455b9fc69cc06e2b5b72f7d1ac5f6c86ac349e77", Platform: chip8.ModernChip8,
	},
	{
		Title: "7. Beep", File: "7-beep.ch8",
		SHA1: "b119651b5aa08557a85ca2ad5de3d1a86796b66b", Platform: chip8.ModernChip8,
	},
	{
		Title: "8. Scrolling", File: "8-scrolling.ch8", SHA1: "477b3e09c43839ea5478b4f0e24536edab594f89",
		Platform: chip8.SuperChip,
	},
}

// Library holds the entries and knows how to load them.
type Library struct {
	entries []Entry
	// testBase is where test-suite ROMs live: a directory natively, or an
	// http(s) URL ending in "/" in the browser.
	testBase string
}

// NewLibrary reads the bundled manifest and adds the test suite, loaded
// from testBase (a directory or a URL).
func NewLibrary(testBase string) (*Library, error) {
	games, err := bundledEntries()
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(games)+len(testSuite))
	entries = append(entries, games...)

	for _, t := range testSuite {
		t.ID = "test/" + strings.TrimSuffix(t.File, ".ch8")
		t.Group = TestSuite
		t.Authors = []string{"Timendus"}

		entries = append(entries, t)
	}

	return &Library{entries: entries, testBase: testBase}, nil
}

func bundledEntries() ([]Entry, error) {
	data, err := assets.ROMs.ReadFile("roms/bundled.json")
	if err != nil {
		return nil, fmt.Errorf("read bundled manifest: %w", err)
	}

	var raw []bundledEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse bundled manifest: %w", err)
	}

	out := make([]Entry, 0, len(raw))

	for _, b := range raw {
		p, ok := chip8.ParsePlatform(b.Platform)
		if !ok {
			return nil, fmt.Errorf("%w: %s: unknown platform %q", ErrBadManifest, b.ID, b.Platform)
		}

		out = append(out, Entry{
			ID: "game/" + b.ID, Title: b.Title, Authors: b.Authors, Release: b.Release,
			Description: b.Description, Group: Games, File: b.File, SHA1: b.SHA1,
			Platform: p, Ticks: b.Tickrate,
		})
	}

	return out, nil
}

// Entries returns all entries: games first, then the test suite.
func (l *Library) Entries() []Entry { return l.entries }

// Find returns the entry with the given ID.
func (l *Library) Find(id string) (Entry, bool) {
	for _, e := range l.entries {
		if e.ID == id {
			return e, true
		}
	}

	return Entry{}, false
}

// Result is the outcome of an asynchronous load.
type Result struct {
	Entry Entry
	ROM   []byte
	Err   error
}

// LoadAsync loads the entry in a goroutine and delivers the result on the
// returned channel, which has room for it, so the sender never blocks.
func (l *Library) LoadAsync(ctx context.Context, e Entry) <-chan Result {
	ch := make(chan Result, 1)

	go func() {
		rom, err := l.Load(ctx, e)
		ch <- Result{Entry: e, ROM: rom, Err: err}
	}()

	return ch
}

// Load reads the entry's ROM and checks its SHA-1. It may block on the
// network; use [Library.LoadAsync] from the game loop.
func (l *Library) Load(ctx context.Context, e Entry) ([]byte, error) {
	var (
		rom []byte
		err error
	)

	switch e.Group {
	case Games:
		rom, err = assets.ROMs.ReadFile(path.Join("roms", e.File))
	case TestSuite:
		rom, err = l.loadTest(ctx, e.File)
	default:
		err = ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("load %s: %w", e.ID, err)
	}

	if e.SHA1 != "" {
		sum := sha1.Sum(rom) //nolint:gosec // G401: see import
		if hex.EncodeToString(sum[:]) != e.SHA1 {
			return nil, fmt.Errorf("load %s: %w", e.ID, ErrHashMismatch)
		}
	}

	return rom, nil
}

func (l *Library) loadTest(ctx context.Context, file string) ([]byte, error) {
	if !strings.HasPrefix(l.testBase, "http://") && !strings.HasPrefix(l.testBase, "https://") {
		rom, err := os.ReadFile(filepath.Join(l.testBase, file))
		if err != nil {
			return nil, fmt.Errorf("read test ROM: %w", err)
		}

		return rom, nil
	}

	rom, err := platform.Fetch(ctx, l.testBase+file, maxROMSize)
	if err != nil {
		return nil, fmt.Errorf("download test ROM: %w", err)
	}

	return rom, nil
}

// ReadFile reads a ROM from the local file system (native -rom flag).
func ReadFile(name string) (Entry, []byte, error) {
	rom, err := os.ReadFile(name)
	if err != nil {
		return Entry{}, nil, fmt.Errorf("read ROM: %w", err)
	}

	if len(rom) > maxROMSize {
		return Entry{}, nil, ErrTooLarge
	}

	return Entry{ID: "file/" + filepath.Base(name), Title: filepath.Base(name), Platform: chip8.ModernChip8}, rom, nil
}
