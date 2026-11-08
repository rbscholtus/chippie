// Command golden-import converts the Timendus test suite's reference
// pictures into golden frames at native resolution.
//
// The pictures are 768×384 with 12×12 blocks per CHIP-8 pixel. Each one is
// converted losslessly (every block must be a single colour) and written as
// testdata/golden/<name>.png and .txt.
//
// Usage, from the repository root:
//
//	go run ./cmd/golden-import
package main

import (
	"flag"
	"fmt"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/rbscholtus/chippie/internal/testrig"
)

// Suite picture colours. Off is drawn in two nearly identical shades.
var suitePalette = testrig.Palette{ //nolint:gochecknoglobals // read-only colour table
	{R: 153, G: 102, B: 1, A: 255}: 0,
	{R: 153, G: 103, B: 0, A: 255}: 0,
	{R: 255, G: 204, B: 1, A: 255}: 1,
}

// picture is one reference picture to import.
type picture struct {
	file   string // in the suite's pictures/ directory
	golden string // base name in testdata/golden/
	width  int
	height int
}

// pictures lists the reference pictures used by the tests. beep.png is
// anti-aliased and can't be converted; the beep test checks the buzzer
// state instead.
var pictures = []picture{ //nolint:gochecknoglobals // read-only import list
	{"chip-8-logo.png", "1-chip8-logo", 64, 32},
	{"ibm-logo.png", "2-ibm-logo", 64, 32},
	{"corax+.png", "3-corax+", 64, 32},
	{"flags.png", "4-flags", 64, 32},
	{"keypad-down.png", "6-keypad-down", 64, 32},
	{"keypad-up.png", "6-keypad-up", 64, 32},
	{"keypad-getkey.png", "6-keypad-getkey", 64, 32},
}

func main() {
	suite := flag.String("suite", "third_party/chip8-test-suite/pictures", "suite pictures directory")
	out := flag.String("out", "testdata/golden", "golden frames directory")

	flag.Parse()

	if err := run(*suite, *out); err != nil {
		log.Fatal(err)
	}
}

func run(suite, out string) error {
	if err := os.MkdirAll(out, 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	for _, p := range pictures {
		if err := importOne(filepath.Join(suite, p.file), filepath.Join(out, p.golden), p, suitePalette); err != nil {
			return fmt.Errorf("%s: %w", p.file, err)
		}

		log.Printf("%s → %s", p.file, p.golden)
	}

	return nil
}

func importOne(src, dst string, p picture, pal map[color.RGBA]byte) error {
	fh, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open picture: %w", err)
	}
	defer fh.Close()

	img, err := png.Decode(fh)
	if err != nil {
		return fmt.Errorf("decode picture: %w", err)
	}

	f, err := testrig.ImportPicture(img, p.width, p.height, pal)
	if err != nil {
		return fmt.Errorf("convert picture: %w", err)
	}

	if err := testrig.WriteGolden(dst, f); err != nil {
		return fmt.Errorf("write golden: %w", err)
	}

	return nil
}
