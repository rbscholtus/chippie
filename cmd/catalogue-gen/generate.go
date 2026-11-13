package main

import (
	"crypto/sha1" //nolint:gosec // G505: SHA-1 identifies ROMs, as in chip-8-database; not for security
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/rbscholtus/chippie/catalogue"
)

// archive is one ROM collection that catalogue-gen hashes.
type archive struct {
	id      string // source id in sources.json
	info    catalogue.SourceInfo
	repo    string // GitHub owner/name
	ref     string // branch or tag to follow with -update-pins
	include func(file string) bool
}

// archives are the collections in README §9.1 that browsers can download
// from (GitHub raw, CORS *). Zophar's packs and David Winter's page aren't:
// the ROMs only found there are hosted by Chippie; see hostedArchive.
var archives = []archive{ //nolint:gochecknoglobals // read-only configuration
	{
		id: "chip8archive",
		info: catalogue.SourceInfo{
			Name: "chip8Archive", License: "CC0-1.0", Home: "https://johnearnest.github.io/chip8Archive/",
		},
		repo: "JohnEarnest/chip8Archive", ref: master,
		include: func(f string) bool { return strings.HasPrefix(f, "roms/") },
	},
	{
		id:   "kripod",
		info: catalogue.SourceInfo{Name: "kripod/chip8-roms", Home: "https://github.com/kripod/chip8-roms"},
		repo: "kripod/chip8-roms", ref: master,
		include: func(f string) bool {
			ext := strings.ToLower(path.Ext(f))

			return strings.Contains(f, "/") && ext != ".txt" && ext != ".md" && !strings.HasPrefix(path.Base(f), ".")
		},
	},
	{
		id: "timendus",
		info: catalogue.SourceInfo{
			Name: "Timendus CHIP-8 test suite", License: gpl3, Home: "https://github.com/Timendus/chip8-test-suite",
		},
		repo: "Timendus/chip8-test-suite", ref: "v4.2",
		include: func(f string) bool { return strings.HasPrefix(f, "bin/") && strings.HasSuffix(f, ".ch8") },
	},
	{
		id: "mattmikolay",
		info: catalogue.SourceInfo{
			Name: "Matthew Mikolay's CHIP-8 programs", License: "MIT", Home: "https://github.com/mattmikolay/chip-8",
		},
		repo: "mattmikolay/chip-8", ref: master,
		include: func(f string) bool { return strings.HasSuffix(f, ".ch8") },
	},
	{
		id: "3dvipermaze",
		info: catalogue.SourceInfo{
			Name: "3D Viper Maze (Timendus)", License: gpl3, Home: "https://github.com/Timendus/3d-viper-maze",
		},
		repo: "Timendus/3d-viper-maze", ref: mainBranch,
		include: func(f string) bool { return f == "3dvipermaze.ch8" },
	},
	{
		id: "3dviprmaze",
		info: catalogue.SourceInfo{
			Name: "3D VIP'r Maze (Timendus)", License: gpl3, Home: "https://github.com/Timendus/3d-vipr-maze",
		},
		repo: "Timendus/3d-vipr-maze", ref: mainBranch,
		include: func(f string) bool { return strings.HasPrefix(f, "bin/") && strings.HasSuffix(f, ".ch8") },
	},
	{
		// Octojam 6-10 entries, extracted from their itch.io pages (which
		// browsers can't download from). No licence: each game's belongs
		// to its author. We only link, as with kripod.
		id: "octojam-itch",
		info: catalogue.SourceInfo{
			Name: "Octojam games from itch.io (via maxisimonazzi/chip8-emulator)",
			Home: "https://github.com/maxisimonazzi/chip8-emulator/tree/main/roms/octojam",
		},
		repo: "maxisimonazzi/chip8-emulator", ref: mainBranch,
		include: func(f string) bool { return strings.HasPrefix(f, "roms/octojam/") && strings.HasSuffix(f, ".ch8") },
	},
}

// hostedArchive is web/hosted-roms: ROMs Chippie serves beside the app, at
// relative URLs, because browsers can't download them anywhere else (Zophar's
// packs and David Winter's page; see its README.md for each file's origin).
// Only the ROMs count, not the notes and attribution next to them, which have
// file extensions.
var hostedArchive = archive{ //nolint:gochecknoglobals // read-only configuration
	id: catalogue.HostedSource,
	info: catalogue.SourceInfo{
		Name: "Hosted by Chippie (from Zophar and David Winter)",
		Home: "https://github.com/rbscholtus/chippie/tree/main/web/hosted-roms",
	},
	include: func(f string) bool { return path.Ext(f) == "" },
}

// hostedURLPrefix is where build.sh puts the hosted ROMs, relative to the app.
const hostedURLPrefix = "roms/hosted/"

// locationURL is where a file of an archive can be downloaded.
func locationURL(t tree, file string) string {
	if t.archive.id == catalogue.HostedSource {
		return hostedURLPrefix + url.PathEscape(file)
	}

	return rawURL(t.archive.repo, t.commit, file)
}

// rawURL is a file's GitHub raw URL, pinned to a commit.
func rawURL(repo, commit, file string) string {
	parts := strings.Split(file, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}

	return "https://raw.githubusercontent.com/" + repo + "/" + commit + "/" + strings.Join(parts, "/")
}

// tree is the files of one archive at a commit, by path inside the repo.
type tree struct {
	archive archive
	commit  string
	files   map[string][]byte
}

// sourceStats is one row of the coverage report.
type sourceStats struct {
	name    string
	unique  int
	inDB    int
	missing []string // files whose hash isn't in the database
}

// result is everything generate produces.
type result struct {
	sources       catalogue.Sources
	stats         []sourceStats
	archiveHashes map[string]string // chip8Archive id → SHA-1
	allUnique     int
	allInDB       int
}

func sha1Hex(b []byte) string {
	sum := sha1.Sum(b) //nolint:gosec // G401: see import

	return hex.EncodeToString(sum[:])
}

// generate hashes every ROM, matches it against the database, and builds
// the sources manifest and coverage numbers. It does no I/O.
func generate(db *catalogue.DB, trees []tree, generated string) result {
	res := result{
		sources: catalogue.Sources{
			Version: catalogue.SourcesVersion, Generated: generated,
			Sources: map[string]catalogue.SourceInfo{}, ROMs: map[string][]catalogue.Location{},
		},
		archiveHashes: map[string]string{},
	}

	all := map[string]bool{}

	for _, t := range trees {
		res.sources.Sources[t.archive.id] = t.archive.info
		st := sourceStats{name: t.archive.info.Name}
		seen := map[string]bool{}

		for _, file := range sortedKeys(t.files) {
			if !t.archive.include(file) {
				continue
			}

			hash := sha1Hex(t.files[file])
			if t.archive.id == "chip8archive" {
				res.archiveHashes[strings.TrimSuffix(path.Base(file), path.Ext(file))] = hash
			}

			if seen[hash] {
				continue
			}

			seen[hash] = true
			all[hash] = true
			st.unique++

			if _, ok := db.ROM(hash); !ok {
				st.missing = append(st.missing, file)

				continue
			}

			st.inDB++

			res.sources.ROMs[hash] = append(res.sources.ROMs[hash],
				catalogue.Location{Source: t.archive.id, URL: locationURL(t, file)})
		}

		res.stats = append(res.stats, st)
	}

	res.allUnique = len(all)

	for h := range all {
		if _, ok := db.ROM(h); ok {
			res.allInDB++
		}
	}

	return res
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

// report renders the coverage report as Markdown (README §9.1).
func report(res result, db *catalogue.DB, conflicts []catalogue.Conflict) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# ROM coverage\n\nGenerated %s by `cmd/catalogue-gen`.\n\n", res.sources.Generated)
	fmt.Fprintf(&b, "chip-8-database: %d programs, %d ROMs.\n\n", len(db.Programs), db.ROMCount())
	b.WriteString("| Source | Unique ROMs | In database | Missing |\n|---|---|---|---|\n")

	for _, s := range res.stats {
		fmt.Fprintf(&b, "| %s | %d | %d | %d |\n", s.name, s.unique, s.inDB, len(s.missing))
	}

	fmt.Fprintf(&b, "| **All combined** | **%d** | **%d** | **%d** |\n",
		res.allUnique, res.allInDB, res.allUnique-res.allInDB)
	fmt.Fprintf(&b, "\nDownloadable ROMs in the manifest: %d of %d in the database.\n",
		len(res.sources.ROMs), db.ROMCount())

	b.WriteString("\n## Not in chip-8-database\n\n")

	for _, s := range res.stats {
		for _, m := range s.missing {
			fmt.Fprintf(&b, "- %s: `%s`\n", s.name, m)
		}
	}

	b.WriteString("\n## chip8Archive vs chip-8-database\n\n")
	b.WriteString("The database wins; these are candidates for upstream fixes.\n\n")

	if len(conflicts) == 0 {
		b.WriteString("None.\n")
	}

	for _, c := range conflicts {
		fmt.Fprintf(&b, "- %s\n", c)
	}

	return b.String()
}

// jsonMarshal encodes the manifest as written to sources.json.
func jsonMarshal(s catalogue.Sources) ([]byte, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode sources: %w", err)
	}

	return append(data, '\n'), nil
}
