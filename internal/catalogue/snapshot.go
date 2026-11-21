package catalogue

import (
	"fmt"

	"github.com/rbscholtus/chippie/assets"
)

// snapshotDir is where the embedded data is in assets.Catalogue:
// chip-8-database's four files, chip8Archive's programs.json with its
// name → SHA-1 map, and the sources manifest. Their source commits are in
// assets/catalogue/snapshot/SNAPSHOT.md.
const snapshotDir = "catalogue/snapshot/"

// SnapshotFiles returns the embedded chip-8-database files.
func SnapshotFiles() (Files, error) {
	var f Files

	for _, part := range f.parts() {
		data, err := assets.Catalogue.ReadFile(snapshotDir + part.name)
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
