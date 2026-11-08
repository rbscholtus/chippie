package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestImportMatchesCommittedGoldens re-imports every picture and checks the
// result is byte-identical to testdata/golden, so the goldens can't drift
// from the suite's pictures.
func TestImportMatchesCommittedGoldens(t *testing.T) {
	t.Parallel()

	suite := filepath.Join("..", "..", "third_party", "chip8-test-suite", "pictures")
	if _, err := os.Stat(suite); err != nil {
		t.Skipf("test suite not checked out: %v", err)
	}

	goldens := filepath.Join("..", "..", "testdata", "golden")

	out := t.TempDir()
	if err := run(suite, out); err != nil {
		t.Fatal(err)
	}

	for _, p := range pictures {
		for _, ext := range []string{".png", ".txt"} {
			name := p.golden + ext

			got, err := os.ReadFile(filepath.Join(out, name))
			if err != nil {
				t.Fatal(err)
			}

			want, err := os.ReadFile(filepath.Join(goldens, name))
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(got, want) {
				t.Errorf("%s differs from the committed golden; rerun go run ./cmd/golden-import", name)
			}
		}
	}
}
