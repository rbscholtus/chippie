// Package library is the list the library drawer shows: every ROM that can
// be played or downloaded, with search. It holds no Ebitengine state, so it
// can be tested directly.
package library

import (
	"path"
	"sort"
	"strings"

	"github.com/rbscholtus/chippie/catalogue"
	"github.com/rbscholtus/chippie/chip8"
)

// Group orders the list: what can be played now, then what needs a platform
// that's still to come.
type Group uint8

// Groups, in display order.
const (
	Playable Group = iota
	ComingSoon
)

// Name is the group's heading.
func (g Group) Name() string {
	if g == ComingSoon {
		return "COMING SOON"
	}

	return "PLAY"
}

// Entry is one ROM in the list.
type Entry struct {
	SHA1        string
	Title       string
	Authors     []string
	Release     string
	Description string
	Platform    chip8.Platform // the recommended platform
	Group       Group
	Offline     bool           // plays without a download: bundled, beside the app, or cached
	Keys        map[string]int // the database's keys hint, e.g. "up": 5; often empty
	Rotation    int            // degrees clockwise the picture is turned (screenRotation)

	search string // lower-case text the search matches against
}

// Playable reports whether Chippie can run the entry's platform yet.
func (e Entry) Playable() bool { return e.Group == Playable }

// Subtitle is a short line about the entry: authors, year and platform.
func (e Entry) Subtitle() string {
	parts := make([]string, 0, 3)
	if len(e.Authors) > 0 {
		parts = append(parts, strings.Join(e.Authors, ", "))
	}

	if y := year(e.Release); y != "" {
		parts = append(parts, y)
	}

	parts = append(parts, e.Platform.Name())

	return strings.Join(parts, " · ")
}

// year returns the year of a release date like "2015-10-31" or "19xx".
func year(release string) string {
	if len(release) >= 4 {
		return release[:4]
	}

	return release
}

// Build lists every ROM in the catalogue that can be downloaded or is
// available offline (offline[sha1]). Programs with several such ROMs get one
// entry per ROM, told apart by file name.
func Build(c *catalogue.Catalogue, offline map[string]bool) []Entry {
	var out []Entry

	for _, prog := range c.DB.Programs {
		available := make([]*catalogue.ROM, 0, len(prog.ROMs))

		for _, rom := range prog.ROMs {
			if offline[rom.SHA1] || c.Sources.Available(rom.SHA1) {
				available = append(available, rom)
			}
		}

		for _, rom := range available {
			title := prog.Title
			if len(available) > 1 && rom.File != "" {
				title += " (" + strings.TrimSuffix(rom.File, path.Ext(rom.File)) + ")"
			}

			out = append(out, newEntry(c, rom, title, offline[rom.SHA1]))
		}
	}

	sortEntries(out)

	return out
}

func newEntry(c *catalogue.Catalogue, rom *catalogue.ROM, title string, offline bool) Entry {
	p := c.DB.Resolve(rom.SHA1, catalogue.Overrides{}).Profile.Platform

	e := Entry{
		SHA1: rom.SHA1, Title: title, Authors: rom.Program.Authors, Release: rom.Program.Release,
		Description: Reflow(rom.Program.Description), Platform: p, Offline: offline, Keys: rom.Keys,
		Rotation: rom.ScreenRotation,
	}
	if !p.Implemented() {
		e.Group = ComingSoon
	}

	e.search = strings.ToLower(e.Title + " " + strings.Join(e.Authors, " ") + " " + e.Release + " " + p.Name())

	return e
}

// LocalEntry describes a ROM from the user's device (file picker, drag and
// drop, -rom), resolved against the catalogue (which may be nil) when it's a
// known ROM.
func LocalEntry(c *catalogue.Catalogue, sha1, fileName string) Entry {
	if c != nil {
		if rom, ok := c.DB.ROM(sha1); ok {
			return newEntry(c, rom, rom.Program.Title, true)
		}
	}

	e := Entry{SHA1: sha1, Title: fileName, Platform: chip8.ModernChip8, Offline: true}
	e.search = strings.ToLower(fileName)

	return e
}

func sortEntries(es []Entry) {
	sort.SliceStable(es, func(i, j int) bool {
		if es[i].Group != es[j].Group {
			return es[i].Group < es[j].Group
		}

		return strings.ToLower(es[i].Title) < strings.ToLower(es[j].Title)
	})
}

// MarkOffline marks the entries whose SHA-1 is in cached as playable
// offline (after a download, or found in the cache at start-up).
func MarkOffline(es []Entry, cached map[string]bool) {
	for i := range es {
		if cached[es[i].SHA1] {
			es[i].Offline = true
		}
	}
}

// Filter returns the entries whose title, authors, year or platform contain
// every word of the query, ignoring case. An empty query returns all.
func Filter(es []Entry, query string) []Entry {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return es
	}

	out := make([]Entry, 0, len(es))

	for _, e := range es {
		if matchesAll(e.search, words) {
			out = append(out, e)
		}
	}

	return out
}

func matchesAll(s string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(s, w) {
			return false
		}
	}

	return true
}

// Find returns the entry with the given SHA-1.
func Find(es []Entry, sha1 string) (Entry, bool) {
	sha1 = strings.ToLower(sha1)
	for _, e := range es {
		if e.SHA1 == sha1 {
			return e, true
		}
	}

	return Entry{}, false
}
