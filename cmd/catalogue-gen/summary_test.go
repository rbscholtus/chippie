package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSummary(t *testing.T) {
	t.Parallel()

	before := snapshotState{
		counts: counts{programs: 300, roms: 543, downloadable: 204, conflicts: 2},
		pins:   map[string]string{"a/db": "1111111aaaa", "b/roms": "2222222bbbb"},
	}
	after := snapshotState{
		counts: counts{programs: 302, roms: 545, downloadable: 204, conflicts: 1},
		pins:   map[string]string{"a/db": "3333333cccc", "b/roms": "2222222bbbb"},
	}

	got := summary(before, true, after)

	for _, want := range []string{
		"| Programs in chip-8-database | 300 | 302 | +2 |",
		"| Downloadable ROMs | 204 | 204 |  |",
		"| chip8Archive conflicts | 2 | 1 | -1 |",
		"| a/db | 1111111 | 3333333 |",
		"| b/roms | 2222222 | unchanged |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("summary lacks %q:\n%s", want, got)
		}
	}
}

func TestSummaryFirstRun(t *testing.T) {
	t.Parallel()

	after := snapshotState{counts: counts{programs: 1}, pins: map[string]string{"a/db": "3333333cccc"}}
	got := summary(snapshotState{}, false, after)

	for _, want := range []string{"| Programs in chip-8-database | – | 1 | |", "| a/db | – | 3333333 |"} {
		if !strings.Contains(got, want) {
			t.Errorf("summary lacks %q:\n%s", want, got)
		}
	}
}

// TestReadState reads the committed snapshot, and reports a missing one.
func TestReadState(t *testing.T) {
	t.Parallel()

	st, ok := readState(filepath.Join("..", "..", "catalogue", "snapshot"))
	if !ok {
		t.Fatal("committed snapshot unreadable")
	}

	if st.counts.programs == 0 || st.counts.roms == 0 || st.counts.downloadable == 0 || len(st.pins) == 0 {
		t.Errorf("state = %+v, want non-zero counts and pins", st)
	}

	if _, ok := readState(t.TempDir()); ok {
		t.Error("empty directory read as a snapshot")
	}
}
