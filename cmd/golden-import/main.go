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
	"maps"
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

// strayWhite is three stray white pixels in quirks.png, next to "DISP.WAIT".
var strayWhite = color.RGBA{R: 255, G: 255, B: 255, A: 255} //nolint:gochecknoglobals // read-only colour

// picture is one reference picture to import.
type picture struct {
	file   string // in the suite's pictures/ directory
	golden string // base name in testdata/golden/
	width  int
	height int
	strays bool // ignore stray white pixels
}

// pictures lists every 64×32 reference picture. The scrolling ones are used
// from M5 (SUPER-CHIP), but are imported now with the rest.
var pictures = []picture{ //nolint:gochecknoglobals // read-only import list
	{file: "chip-8-logo.png", golden: "1-chip8-logo", width: 64, height: 32},
	{file: "ibm-logo.png", golden: "2-ibm-logo", width: 64, height: 32},
	{file: "corax+.png", golden: "3-corax+", width: 64, height: 32},
	{file: "flags.png", golden: "4-flags", width: 64, height: 32},
	{file: "quirks-platform.png", golden: "5-quirks-menu", width: 64, height: 32},
	{file: "quirks.png", golden: "5-quirks-chip8", width: 64, height: 32, strays: true},
	{file: "keypad-menu.png", golden: "6-keypad-menu", width: 64, height: 32},
	{file: "keypad-down.png", golden: "6-keypad-down", width: 64, height: 32},
	{file: "keypad-up.png", golden: "6-keypad-up", width: 64, height: 32},
	{file: "keypad-getkey.png", golden: "6-keypad-getkey", width: 64, height: 32},
	{file: "scrolling-platform.png", golden: "8-scrolling-menu", width: 64, height: 32},
	{file: "lores-scrolling.png", golden: "8-scrolling-lores", width: 64, height: 32},
	{file: "lores-no-scrolling.png", golden: "8-scrolling-lores-none", width: 64, height: 32},
	{file: "lores-no-half-scrolling.png", golden: "8-scrolling-lores-no-half", width: 64, height: 32},
}

// skipped lists the suite's other pictures and why they aren't imported.
var skipped = []struct{ file, reason string }{ //nolint:gochecknoglobals // read-only list
	{"beep.png", "anti-aliased; the beep test checks the buzzer state instead"},
	{"Cosmac-VIP-quirks.png", "screenshot of the Emma 02 emulator; quirks.png shows the same result"},
	{"hires-scrolling.png, hires-no-scrolling.png", "128×64; imported with the SUPER-CHIP tests in M5"},
	{"HP48SX-*.JPG, broken-on-hp48.jpg", "photos of a real HP48"},
	{"ibm.png, splash*.png, version.png, testimonial*.png", "not test screens"},
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
		pal := suitePalette
		if p.strays {
			pal = maps.Clone(suitePalette)
			pal[strayWhite] = testrig.Ignore
		}

		if err := importOne(filepath.Join(suite, p.file), filepath.Join(out, p.golden), p, pal); err != nil {
			return fmt.Errorf("%s: %w", p.file, err)
		}

		log.Printf("%s → %s", p.file, p.golden)
	}

	for _, s := range skipped {
		log.Printf("skipped %s: %s", s.file, s.reason)
	}

	return nil
}

func importOne(src, dst string, p picture, pal testrig.Palette) error {
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
