// Command catalogue-gen refreshes the catalogue snapshot (README §9–10). It
// downloads chip-8-database and the ROM archives at pinned commits, hashes
// every ROM, and writes into catalogue/snapshot/:
//
//	platforms.json quirks.json programs.json sha1-hashes.json  chip-8-database
//	chip8archive-programs.json chip8archive-hashes.json        chip8Archive metadata
//	sources.json                                               where each ROM can be downloaded,
//	                                                           including web/hosted-roms
//	pins.json                                                  the commits used
//
// and the coverage report to docs/coverage.md. It needs the network; the
// refresh workflow runs it weekly with -update-pins.
//
// Usage, from the repository root:
//
//	go run ./cmd/catalogue-gen               # regenerate at the pinned commits
//	go run ./cmd/catalogue-gen -update-pins  # follow each repository's branch or tag
//
// -summary writes a Markdown comparison with the previous snapshot (counts
// and pins), which the refresh workflow shows on its run page and in its PR.
package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/rbscholtus/chippie/catalogue"
	"github.com/rbscholtus/chippie/platform"
)

const (
	databaseRepo = "chip-8/chip-8-database"
	databaseRef  = master

	master     = "master" // default branch names
	mainBranch = "main"
	gpl3       = "GPL-3.0" // licence of Timendus's repositories

	programsFile = "programs.json"

	maxJSON    = 16 << 20  // a database file
	maxTarball = 512 << 20 // a repository download
	maxROMFile = 1 << 20   // one file inside it
)

var errBadTarball = errors.New("bad tarball")

func main() {
	snapshotDir := flag.String("snapshot", "catalogue/snapshot", "snapshot directory")
	reportPath := flag.String("report", "docs/coverage.md", "coverage report")
	updatePins := flag.Bool("update-pins", false, "follow each repository's branch or tag instead of the pins")
	hostedDir := flag.String("hosted", "web/hosted-roms", "directory of the ROMs Chippie hosts itself")
	summaryPath := flag.String("summary", "", "write a before/after summary here (Markdown)")

	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	err := run(ctx, options{
		dir: *snapshotDir, reportPath: *reportPath, hostedDir: *hostedDir,
		summaryPath: *summaryPath, updatePins: *updatePins,
	})

	cancel()

	if err != nil {
		log.Fatal(err)
	}
}

// options are the command-line flags.
type options struct {
	dir, reportPath, hostedDir, summaryPath string
	updatePins                              bool
}

func run(ctx context.Context, o options) error {
	dir := o.dir

	pins, err := readPins(filepath.Join(dir, "pins.json"))
	if err != nil {
		return err
	}

	before, hadBefore := readState(dir)

	if o.updatePins {
		if err := resolvePins(ctx, pins); err != nil {
			return err
		}
	}

	db, err := fetchDatabase(ctx, dir, pins[databaseRepo])
	if err != nil {
		return err
	}

	trees := make([]tree, 0, len(archives))

	for _, a := range archives {
		files, err := fetchTree(ctx, a.repo, pins[a.repo], a.include)
		if err != nil {
			return fmt.Errorf("%s: %w", a.repo, err)
		}

		trees = append(trees, tree{archive: a, commit: pins[a.repo], files: files})
		log.Printf("%s@%.7s: %d files", a.repo, pins[a.repo], len(files))
	}

	hosted, err := readHosted(o.hostedDir)
	if err != nil {
		return err
	}

	trees = append(trees, tree{archive: hostedArchive, files: hosted})

	res := generate(db, trees, time.Now().UTC().Format(time.RFC3339))
	keepTimestamp(filepath.Join(dir, "sources.json"), &res.sources)

	conflicts, err := writeOutputs(dir, o.reportPath, db, trees[0].files[programsFile], res, pins)
	if err != nil || o.summaryPath == "" {
		return err
	}

	after := snapshotState{
		counts: counts{
			programs: len(db.Programs), roms: db.ROMCount(),
			downloadable: len(res.sources.ROMs), conflicts: conflicts,
		},
		pins: pins,
	}

	if err := os.WriteFile(o.summaryPath, []byte(summary(before, hadBefore, after)), 0o600); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}

	return nil
}

// readHosted reads the files directly inside dir, by name.
func readHosted(dir string) (map[string][]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("hosted ROMs: %w", err)
	}

	files := make(map[string][]byte, len(entries))

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("hosted ROMs: %w", err)
		}

		files[e.Name()] = data
	}

	return files, nil
}

// writeOutputs merges chip8Archive's metadata (to find conflicts) and
// writes the snapshot files and the report. It returns the number of
// conflicts.
func writeOutputs(
	dir, reportPath string, db *catalogue.DB, programs []byte, res result, pins map[string]string,
) (int, error) {
	if programs == nil {
		return 0, fmt.Errorf("chip8Archive: %s missing: %w", programsFile, errBadTarball)
	}

	hashes, err := json.MarshalIndent(res.archiveHashes, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("encode archive hashes: %w", err)
	}

	conflicts, err := db.MergeArchive(programs, hashes)
	if err != nil {
		return 0, fmt.Errorf("merge chip8Archive: %w", err)
	}

	sources, err := jsonMarshal(res.sources)
	if err != nil {
		return 0, err
	}

	pinsJSON, err := json.MarshalIndent(pins, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("encode pins: %w", err)
	}

	for name, data := range map[string][]byte{
		"chip8archive-programs.json": programs,
		"chip8archive-hashes.json":   append(hashes, '\n'),
		"sources.json":               sources,
		"pins.json":                  append(pinsJSON, '\n'),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			return 0, fmt.Errorf("write %s: %w", name, err)
		}
	}

	if err := os.WriteFile(reportPath, []byte(report(res, db, conflicts)), 0o600); err != nil {
		return 0, fmt.Errorf("write report: %w", err)
	}

	log.Printf("%d of %d ROMs downloadable; %d conflicts; report in %s",
		len(res.sources.ROMs), db.ROMCount(), len(conflicts), reportPath)

	return len(conflicts), nil
}

// keepTimestamp reuses the previous manifest's timestamp when nothing else
// changed, so an unchanged refresh leaves the files untouched.
func keepTimestamp(path string, s *catalogue.Sources) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	old, err := catalogue.ParseSources(data)
	if err != nil {
		return
	}

	candidate := *s
	candidate.Generated = old.Generated

	if reflect.DeepEqual(old, &candidate) {
		s.Generated = old.Generated
	}
}

// readPins reads repo → commit; every repository must be pinned.
func readPins(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pins: %w", err)
	}

	var pins map[string]string
	if err := json.Unmarshal(data, &pins); err != nil {
		return nil, fmt.Errorf("parse pins: %w", err)
	}

	for _, repo := range append([]string{databaseRepo}, repos()...) {
		if pins[repo] == "" {
			return nil, fmt.Errorf("pins.json: %s not pinned: %w", repo, os.ErrNotExist)
		}
	}

	return pins, nil
}

func repos() []string {
	out := make([]string, len(archives))
	for i, a := range archives {
		out[i] = a.repo
	}

	return out
}

// resolvePins updates each pin to the current commit of its branch or tag.
func resolvePins(ctx context.Context, pins map[string]string) error {
	refs := map[string]string{databaseRepo: databaseRef}
	for _, a := range archives {
		refs[a.repo] = a.ref
	}

	for repo, ref := range refs {
		data, err := platform.Fetch(ctx, "https://api.github.com/repos/"+repo+"/commits/"+ref, maxJSON)
		if err != nil {
			return fmt.Errorf("resolve %s@%s: %w", repo, ref, err)
		}

		var c struct {
			SHA string `json:"sha"`
		}

		if err := json.Unmarshal(data, &c); err != nil || c.SHA == "" {
			return fmt.Errorf("resolve %s@%s: bad response: %w", repo, ref, err)
		}

		pins[repo] = c.SHA
	}

	return nil
}

// fetchDatabase downloads chip-8-database's files at the commit, checks
// they parse, and writes them into the snapshot.
func fetchDatabase(ctx context.Context, dir, commit string) (*catalogue.DB, error) {
	var f catalogue.Files

	for _, part := range []struct {
		name string
		dst  *[]byte
	}{
		{"platforms.json", &f.Platforms},
		{"quirks.json", &f.Quirks},
		{programsFile, &f.Programs},
		{"sha1-hashes.json", &f.Hashes},
	} {
		data, err := platform.Fetch(ctx, rawURL(databaseRepo, commit, "database/"+part.name), maxJSON)
		if err != nil {
			return nil, fmt.Errorf("chip-8-database: %w", err)
		}

		*part.dst = data
	}

	db, err := catalogue.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("chip-8-database@%.7s: %w", commit, err)
	}

	for name, data := range map[string][]byte{
		"platforms.json": f.Platforms, "quirks.json": f.Quirks,
		programsFile: f.Programs, "sha1-hashes.json": f.Hashes,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			return nil, fmt.Errorf("write %s: %w", name, err)
		}
	}

	return db, nil
}

// fetchTree downloads a repository at a commit and returns the files that
// include accepts, plus a top-level programs.json if there is one.
func fetchTree(ctx context.Context, repo, commit string, include func(string) bool) (map[string][]byte, error) {
	data, err := platform.Fetch(ctx, "https://codeload.github.com/"+repo+"/tar.gz/"+commit, maxTarball)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	return untar(data, func(name string) bool { return name == programsFile || include(name) })
}

// untar reads a GitHub tarball (everything under one top-level directory)
// and returns the regular files that keep accepts, by path below that
// directory.
func untar(data []byte, keep func(string) bool) (map[string][]byte, error) {
	zr, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errBadTarball, err)
	}

	tr := tar.NewReader(zr)
	files := map[string][]byte{}

	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return files, nil
		}

		if err != nil {
			return nil, fmt.Errorf("%w: %w", errBadTarball, err)
		}

		_, name, ok := strings.Cut(h.Name, "/")
		if !ok || h.Typeflag != tar.TypeReg || !keep(name) {
			continue
		}

		body, err := io.ReadAll(io.LimitReader(tr, maxROMFile+1))
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", errBadTarball, name, err)
		}

		if len(body) > maxROMFile {
			continue // not a ROM
		}

		files[name] = body
	}
}
