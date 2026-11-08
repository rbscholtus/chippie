package catalogue

import (
	"embed"
	"fmt"
)

// snapshot holds the embedded data: chip-8-database's four files,
// chip8Archive's programs.json with its name → SHA-1 map, and the sources
// manifest. Their source commits are in snapshot/SNAPSHOT.md.
//
//go:embed snapshot/*.json
var snapshot embed.FS

// SnapshotFiles returns the embedded chip-8-database files.
func SnapshotFiles() (Files, error) {
	var f Files

	for _, part := range f.parts() {
		data, err := snapshot.ReadFile("snapshot/" + part.name)
		if err != nil {
			return Files{}, fmt.Errorf("read snapshot: %w", err)
		}

		*part.data = data
	}

	return f, nil
}

// Snapshot parses the embedded database.
func Snapshot() (*DB, error) {
	f, err := SnapshotFiles()
	if err != nil {
		return nil, err
	}

	return Parse(f)
}
