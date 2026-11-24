package library

import (
	"strings"
	"testing"

	"github.com/rbscholtus/chippie/internal/catalogue"
	"github.com/rbscholtus/chippie/internal/chip8"
)

const (
	glitchGhost = "2cdcb3c29a5f013a991db5909ca8e18e27b3c42b"
	clockVIP    = "016345d75eef34448840845a9590d41e6bfdf46a" // needs hybridVIP, not implemented
	ibmTest     = "b9bbc12cee3f7b9d3b1f69161f7d7a2d86953379"
	loopzHP48   = "3c444e43e5f02dac4324b7b24cd38ef4938a4b56" // Andreas Daumann's original, 1993
	loopzHap    = "a2788177b820a28cd27e6d2d180340cb7f4948fb" // hap's, with difficulty select
)

func snapshot(t *testing.T) *catalogue.Catalogue {
	t.Helper()

	c, err := catalogue.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestBuild(t *testing.T) {
	t.Parallel()

	c := snapshot(t)
	es := Build(c, map[string]bool{glitchGhost: true})

	if len(es) < 200 {
		t.Fatalf("%d entries, want the ~200 downloadable ROMs", len(es))
	}

	// Playable entries come first, each group sorted by title.
	seenComingSoon := false

	for i, e := range es {
		if e.Group == ComingSoon {
			seenComingSoon = true
		} else if seenComingSoon {
			t.Fatalf("entry %d (%s) is playable after the coming-soon group", i, e.Title)
		}

		if e.Playable() != e.Platform.Implemented() {
			t.Errorf("%s: playable %v but platform %s", e.Title, e.Playable(), e.Platform)
		}
	}

	for i := 1; i < len(es); i++ {
		if es[i].Group == es[i-1].Group && strings.ToLower(es[i].Title) < strings.ToLower(es[i-1].Title) {
			t.Errorf("not sorted: %q before %q", es[i-1].Title, es[i].Title)
		}
	}

	g, ok := Find(es, strings.ToUpper(glitchGhost))
	if !ok || !g.Offline || !g.Playable() || g.Title != "Glitch Ghost" {
		t.Errorf("Glitch Ghost: %+v %v", g, ok)
	}

	if cl, ok := Find(es, clockVIP); !ok || cl.Playable() || cl.Group != ComingSoon {
		t.Errorf("Clock Program should be coming soon: %+v", cl)
	}
}

func TestBuildLeavesOutUnavailable(t *testing.T) {
	t.Parallel()

	c := snapshot(t)
	es := Build(c, nil)

	for _, e := range es {
		if !c.Sources.Available(e.SHA1) {
			t.Errorf("%s isn't downloadable or offline but is listed", e.Title)
		}
	}
}

func TestFilter(t *testing.T) {
	t.Parallel()

	es := Build(snapshot(t), nil)

	tests := []struct {
		query string
		want  string // a title that must be in the result
		max   int
	}{
		{"", "Glitch Ghost", len(es)},
		{"glitch", "Glitch Ghost", 3},
		{"GLITCH ghost", "Glitch Ghost", 3},
		{"kircher", "Glitch Ghost", 10}, // author
		{"ibm logo", "CHIP-8 test suite: IBM logo", 10},
		{"xo-chip", "Red October V", len(es)}, // platform name
	}

	for _, tt := range tests {
		got := Filter(es, tt.query)
		if len(got) == 0 || len(got) > tt.max {
			t.Errorf("Filter(%q): %d results", tt.query, len(got))

			continue
		}

		if _, ok := findTitle(got, tt.want); !ok {
			t.Errorf("Filter(%q) lacks %q", tt.query, tt.want)
		}
	}

	if got := Filter(es, "no such game anywhere"); len(got) != 0 {
		t.Errorf("nonsense query matched %d", len(got))
	}
}

func findTitle(es []Entry, title string) (Entry, bool) {
	for _, e := range es {
		if e.Title == title {
			return e, true
		}
	}

	return Entry{}, false
}

func TestMarkOfflineAndKeys(t *testing.T) {
	t.Parallel()

	es := Build(snapshot(t), nil)
	MarkOffline(es, map[string]bool{glitchGhost: true})

	if g, _ := Find(es, glitchGhost); !g.Offline {
		t.Error("Glitch Ghost should be offline after MarkOffline")
	}

	withKeys := 0

	for _, e := range es {
		if len(e.Keys) > 0 {
			withKeys++
		}
	}

	if withKeys < 20 {
		t.Errorf("only %d entries carry a keys hint", withKeys)
	}
}

func TestSubtitleAndLocal(t *testing.T) {
	t.Parallel()

	c := snapshot(t)

	ibm := LocalEntry(c, ibmTest, "whatever.ch8")
	if ibm.Title != "CHIP-8 test suite: IBM logo" || !strings.Contains(ibm.Subtitle(), "Timendus · 2023") {
		t.Errorf("known local ROM: %q / %q", ibm.Title, ibm.Subtitle())
	}

	unknown := LocalEntry(c, strings.Repeat("0", 40), "mine.ch8")
	if unknown.Title != "mine.ch8" || unknown.Platform != chip8.ModernChip8 || !unknown.Playable() {
		t.Errorf("unknown local ROM: %+v", unknown)
	}

	if got := Filter([]Entry{unknown}, "MINE"); len(got) != 1 {
		t.Error("local entries should be searchable by file name")
	}

	if Playable.Name() == ComingSoon.Name() {
		t.Error("group names should differ")
	}
}

func TestOtherVersion(t *testing.T) {
	t.Parallel()

	es := Build(snapshot(t), nil)

	hp48, ok := Find(es, loopzHP48)
	if !ok {
		t.Fatal("Loopz (1993) isn't in the library")
	}

	if other, ok := OtherVersion(es, hp48); !ok || other.SHA1 != loopzHap {
		t.Errorf("Loopz (1993): other version %q, %v; want hap's", other.SHA1, ok)
	}

	ibm, _ := Find(es, ibmTest)
	if other, ok := OtherVersion(es, ibm); ok {
		t.Errorf("IBM logo has one ROM, yet another version: %s", other.Title)
	}

	if _, ok := OtherVersion(es, LocalEntry(nil, strings.Repeat("0", 40), "mine.ch8")); ok {
		t.Error("an unknown local ROM has no other versions")
	}
}

func TestDescriptionHasTheROMsNotesFirst(t *testing.T) {
	t.Parallel()

	hap, ok := Find(Build(snapshot(t), nil), loopzHap)
	if !ok {
		t.Fatal("hap's Loopz isn't in the library")
	}

	if !strings.HasPrefix(hap.Description, "difficulty select for Loopz, by hap") {
		t.Errorf("hap's Loopz: description starts %.40q, want the ROM's own notes", hap.Description)
	}
}
