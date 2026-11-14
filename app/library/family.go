package library

import "github.com/rbscholtus/chippie/chip8"

// Family groups platforms for the library's filter: the variants a player
// thinks of as one kind of game.
type Family uint8

// Families, in the filter's order. AllFamilies means no filter.
const (
	AllFamilies Family = iota
	Chip8Family
	SuperChipFamily
	XOChipFamily
	OtherFamily // e.g. MEGA-CHIP: only under All
)

// Filters are the families the library offers as filters, in order.
//
//nolint:gochecknoglobals // read-only list
var Filters = []Family{AllFamilies, Chip8Family, SuperChipFamily, XOChipFamily}

// FamilyOf returns the platform's family.
func FamilyOf(p chip8.Platform) Family {
	switch p {
	case chip8.OriginalChip8, chip8.HybridVIP, chip8.ModernChip8, chip8.Chip8X:
		return Chip8Family
	case chip8.Chip48, chip8.SuperChip1, chip8.SuperChip:
		return SuperChipFamily
	case chip8.XOChip:
		return XOChipFamily
	case chip8.MegaChip8:
		return OtherFamily
	default:
		return OtherFamily
	}
}

// Name is the family's label in the filter.
func (f Family) Name() string {
	switch f {
	case AllFamilies:
		return "All"
	case Chip8Family:
		return "CHIP-8"
	case SuperChipFamily:
		return "SUPER-CHIP"
	case XOChipFamily:
		return "XO-CHIP"
	case OtherFamily:
		return "Other"
	default:
		return "Other"
	}
}

// ShortName is the family's label where space is tight: SUPER-CHIP is
// commonly called SCHIP.
func (f Family) ShortName() string {
	if f == SuperChipFamily {
		return "SCHIP"
	}

	return f.Name()
}

// Key is how a family is saved in preferences.
func (f Family) Key() string {
	switch f {
	case Chip8Family:
		return "chip8"
	case SuperChipFamily:
		return "superchip"
	case XOChipFamily:
		return "xochip"
	case AllFamilies, OtherFamily:
		return ""
	default:
		return ""
	}
}

// FamilyFromKey is the family saved as key, or AllFamilies.
func FamilyFromKey(key string) Family {
	for _, f := range Filters {
		if f.Key() == key && key != "" {
			return f
		}
	}

	return AllFamilies
}

// searchWords are extra words the search matches for a family, so short
// names find a platform: "chip8", "schip", "xochip".
func (f Family) searchWords() string {
	switch f {
	case Chip8Family:
		return "chip8 chip-8"
	case SuperChipFamily:
		return "schip superchip super-chip"
	case XOChipFamily:
		return "xochip xo-chip"
	case AllFamilies, OtherFamily:
		return ""
	default:
		return ""
	}
}

// InFamily returns the entries of family f; AllFamilies returns all.
func InFamily(es []Entry, f Family) []Entry {
	if f == AllFamilies {
		return es
	}

	out := make([]Entry, 0, len(es))

	for _, e := range es {
		if e.Family == f {
			out = append(out, e)
		}
	}

	return out
}

// FamilyCounts counts the entries per family; AllFamilies holds the total.
func FamilyCounts(es []Entry) map[Family]int {
	n := map[Family]int{AllFamilies: len(es)}
	for _, e := range es {
		n[e.Family]++
	}

	return n
}
