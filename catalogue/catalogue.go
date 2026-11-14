package catalogue

import "fmt"

// Catalogue is everything the library needs: the database (with
// chip8Archive's metadata merged in) and where each ROM can be downloaded.
type Catalogue struct {
	DB        *DB
	Sources   *Sources
	Conflicts []Conflict      // chip8Archive vs chip-8-database, for reports
	Overlay   []OverlayStatus // Chippie's pending additions (overlay.json)
}

// Load builds a catalogue from database files with Chippie's overlay
// applied, chip8Archive's programs.json and name → hash map, and a sources
// manifest.
func Load(f Files, archivePrograms, archiveHashes, sources []byte) (*Catalogue, error) {
	db, err := Parse(f)
	if err != nil {
		return nil, err
	}

	overlay, err := EmbeddedOverlay()
	if err != nil {
		return nil, err
	}

	status, err := db.ApplyOverlay(overlay)
	if err != nil {
		return nil, err
	}

	conflicts, err := db.MergeArchive(archivePrograms, archiveHashes)
	if err != nil {
		return nil, err
	}

	src, err := ParseSources(sources)
	if err != nil {
		return nil, err
	}

	return &Catalogue{DB: db, Sources: src, Conflicts: conflicts, Overlay: status}, nil
}

// LoadSnapshot builds the catalogue from the embedded snapshot.
func LoadSnapshot() (*Catalogue, error) {
	f, err := SnapshotFiles()
	if err != nil {
		return nil, err
	}

	return LoadWith(f)
}

// LoadWith builds a catalogue from (newer) database files, with the
// embedded chip8Archive metadata and sources manifest.
func LoadWith(f Files) (*Catalogue, error) {
	extra := make([][]byte, 3)

	for i, name := range []string{"chip8archive-programs.json", "chip8archive-hashes.json", "sources.json"} {
		data, err := snapshot.ReadFile("snapshot/" + name)
		if err != nil {
			return nil, fmt.Errorf("read snapshot: %w", err)
		}

		extra[i] = data
	}

	return Load(f, extra[0], extra[1], extra[2])
}
