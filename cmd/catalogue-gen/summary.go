package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rbscholtus/chippie/catalogue"
)

// counts are a snapshot's headline numbers, compared in the refresh summary.
type counts struct {
	programs, roms, downloadable, conflicts int
}

// snapshotState is what the summary compares: the numbers and the pins.
type snapshotState struct {
	counts      counts
	pins        map[string]string
	overlayDone []string // overlay entries upstream now has: delete them
}

// readState reads the snapshot in dir as it is before a refresh. ok is
// false if it is missing or doesn't parse, as on a first run.
func readState(dir string) (snapshotState, bool) {
	read := func(name string) []byte {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil
		}

		return data
	}

	db, err := catalogue.Parse(catalogue.Files{
		Platforms: read("platforms.json"), Quirks: read("quirks.json"),
		Programs: read(programsFile), Hashes: read("sha1-hashes.json"),
	})
	if err != nil {
		return snapshotState{}, false
	}

	conflicts, err := db.MergeArchive(read("chip8archive-programs.json"), read("chip8archive-hashes.json"))
	if err != nil {
		return snapshotState{}, false
	}

	sources, err := catalogue.ParseSources(read("sources.json"))
	if err != nil {
		return snapshotState{}, false
	}

	pins, err := readPins(filepath.Join(dir, "pins.json"))
	if err != nil {
		return snapshotState{}, false
	}

	return snapshotState{
		counts: counts{
			programs: len(db.Programs), roms: db.ROMCount(),
			downloadable: len(sources.ROMs), conflicts: len(conflicts),
		},
		pins: pins,
	}, true
}

// summary renders the before/after comparison as Markdown, for the
// workflow's run page and the refresh PR. hadBefore is false when there was
// no readable snapshot to compare with.
func summary(before snapshotState, hadBefore bool, after snapshotState) string {
	var b strings.Builder

	b.WriteString("## Catalogue refresh\n\n| | Before | After | Change |\n|---|---|---|---|\n")

	for _, row := range []struct {
		name          string
		before, after int
	}{
		{"Programs in chip-8-database", before.counts.programs, after.counts.programs},
		{"ROMs in chip-8-database", before.counts.roms, after.counts.roms},
		{"Downloadable ROMs", before.counts.downloadable, after.counts.downloadable},
		{"chip8Archive conflicts", before.counts.conflicts, after.counts.conflicts},
	} {
		if !hadBefore {
			fmt.Fprintf(&b, "| %s | – | %d | |\n", row.name, row.after)

			continue
		}

		change := ""
		if d := row.after - row.before; d != 0 {
			change = fmt.Sprintf("%+d", d)
		}

		fmt.Fprintf(&b, "| %s | %d | %d | %s |\n", row.name, row.before, row.after, change)
	}

	b.WriteString("\n| Repository | Before | After |\n|---|---|---|\n")

	repos := make([]string, 0, len(after.pins))
	for r := range after.pins {
		repos = append(repos, r)
	}

	sort.Strings(repos)

	for _, r := range repos {
		old, now := short(before.pins[r]), short(after.pins[r])
		if old == now {
			now = "unchanged"
		}

		if old == "" {
			old = "–"
		}

		fmt.Fprintf(&b, "| %s | %s | %s |\n", r, old, now)
	}

	if len(after.overlayDone) > 0 {
		b.WriteString("\n**Upstream now has these overlay entries; delete them from `catalogue/overlay.json`:**\n\n")

		for _, d := range after.overlayDone {
			fmt.Fprintf(&b, "- %s\n", d)
		}
	}

	return b.String()
}

// short is a commit's 7-character abbreviation.
func short(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}

	return commit
}
