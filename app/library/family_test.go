package library

import (
	"strings"
	"testing"

	"github.com/rbscholtus/chippie/chip8"
)

func TestFamilyOf(t *testing.T) {
	t.Parallel()

	for p, want := range map[chip8.Platform]Family{
		chip8.OriginalChip8: Chip8Family, chip8.ModernChip8: Chip8Family, chip8.HybridVIP: Chip8Family,
		chip8.Chip48: SuperChipFamily, chip8.SuperChip1: SuperChipFamily, chip8.SuperChip: SuperChipFamily,
		chip8.XOChip: XOChipFamily, chip8.MegaChip8: OtherFamily,
	} {
		if got := FamilyOf(p); got != want {
			t.Errorf("FamilyOf(%s) = %s, want %s", p, got.Name(), want.Name())
		}
	}

	for _, f := range Filters {
		if got := FamilyFromKey(f.Key()); got != f {
			t.Errorf("FamilyFromKey(%q) = %s, want %s", f.Key(), got.Name(), f.Name())
		}
	}

	if FamilyFromKey("nonsense") != AllFamilies {
		t.Error("an unknown saved key should mean All")
	}
}

func TestFamilySearchAndFilter(t *testing.T) {
	t.Parallel()

	c := snapshot(t)
	all := Build(c, nil)

	// Short words find a family even where the platform's name differs.
	for word, want := range map[string]Family{"schip": SuperChipFamily, "chip8": Chip8Family, "xo": XOChipFamily} {
		found := Filter(all, word)
		if len(found) == 0 {
			t.Errorf("%q finds nothing", word)
		}

		for _, e := range found {
			if e.Family != want && !strings.Contains(strings.ToLower(e.Title+" "+strings.Join(e.Authors, " ")), word) {
				t.Errorf("%q found %q (%s)", word, e.Title, e.Family.Name())
			}
		}
	}

	counts := FamilyCounts(all)
	if counts[AllFamilies] != len(all) ||
		counts[Chip8Family]+counts[SuperChipFamily]+counts[XOChipFamily]+counts[OtherFamily] != len(all) {
		t.Errorf("counts %v don't add up to %d", counts, len(all))
	}

	for _, f := range Filters[1:] {
		got := InFamily(all, f)
		if len(got) != counts[f] {
			t.Errorf("%s: %d entries, counted %d", f.Name(), len(got), counts[f])
		}
	}

	if len(InFamily(all, AllFamilies)) != len(all) {
		t.Error("All should keep every entry")
	}
}
