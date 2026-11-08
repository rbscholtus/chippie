package testrig

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rbscholtus/chippie/chip8"
)

func testFrame() Frame {
	f := Frame{Width: 4, Height: 2, Pix: []byte{0, 1, 2, 3, 1, 0, 0, 1}}

	return f
}

func TestTextRoundTrip(t *testing.T) {
	t.Parallel()

	f := testFrame()
	if got, want := f.Text(), ".#+@\n#..#\n"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}

	back, err := ParseText(f.Text())
	if err != nil {
		t.Fatal(err)
	}

	if d, _ := Diff(f, back); d != "" {
		t.Errorf("round trip differs:\n%s", d)
	}
}

func TestParseTextErrors(t *testing.T) {
	t.Parallel()

	for _, s := range []string{"", "..\n...\n", "..\n.x\n"} {
		if _, err := ParseText(s); !errors.Is(err, ErrBadFrame) {
			t.Errorf("ParseText(%q): err=%v, want ErrBadFrame", s, err)
		}
	}
}

func TestGoldenRoundTrip(t *testing.T) {
	t.Parallel()

	base := filepath.Join(t.TempDir(), "frame")
	f := testFrame()

	if err := WriteGolden(base, f); err != nil {
		t.Fatal(err)
	}

	back, err := LoadGolden(base + ".png")
	if err != nil {
		t.Fatal(err)
	}

	if d, _ := Diff(f, back); d != "" {
		t.Errorf("PNG round trip differs:\n%s", d)
	}

	txt, err := os.ReadFile(base + ".txt")
	if err != nil || string(txt) != f.Text() {
		t.Errorf(".txt = %q, %v; want %q", txt, err, f.Text())
	}
}

func TestDecodePNGRejectsNonPaletted(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}

	if _, err := DecodePNG(&buf); !errors.Is(err, ErrBadFrame) {
		t.Errorf("err=%v, want ErrBadFrame", err)
	}
}

func TestDiff(t *testing.T) {
	t.Parallel()

	a := testFrame()
	b := testFrame()

	b.Pix = append([]byte(nil), a.Pix...)
	b.Pix[0], b.Pix[5] = 1, 1

	d, err := Diff(a, b)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(d, "2 pixels differ") || !strings.Contains(d, "row  0") || !strings.Contains(d, "row  1") {
		t.Errorf("unexpected diff:\n%s", d)
	}

	if _, err := Diff(a, Frame{Width: 1, Height: 1, Pix: []byte{0}}); !errors.Is(err, ErrSizeMismatch) {
		t.Errorf("size mismatch: err=%v", err)
	}
}

// scaled builds a picture of the frame with each pixel drawn as a
// scale×scale block in the given colours.
func scaled(f Frame, scale int, colours []color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, f.Width*scale, f.Height*scale))
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			img.SetRGBA(x, y, colours[f.At(x/scale, y/scale)])
		}
	}

	return img
}

var (
	off   = color.RGBA{R: 153, G: 102, B: 1, A: 255}
	off2  = color.RGBA{R: 153, G: 103, B: 0, A: 255}
	onCol = color.RGBA{R: 255, G: 204, B: 1, A: 255}
	pal   = Palette{off: 0, off2: 0, onCol: 1}
)

func TestImportPicture(t *testing.T) {
	t.Parallel()

	f := Frame{Width: 3, Height: 2, Pix: []byte{0, 1, 0, 1, 1, 0}}
	img := scaled(f, 12, []color.RGBA{off, onCol})

	got, err := ImportPicture(img, 3, 2, pal)
	if err != nil {
		t.Fatal(err)
	}

	if d, _ := Diff(f, got); d != "" {
		t.Errorf("import differs:\n%s", d)
	}

	// Two shades of "off" in one block are fine: they map to the same value.
	img.SetRGBA(0, 0, off2)

	if _, err := ImportPicture(img, 3, 2, pal); err != nil {
		t.Errorf("two off shades: %v", err)
	}
}

func TestImportPictureIgnoresStrayMarks(t *testing.T) {
	t.Parallel()

	f := Frame{Width: 2, Height: 1, Pix: []byte{0, 1}}
	img := scaled(f, 4, []color.RGBA{off, onCol})
	stray := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	img.SetRGBA(1, 1, stray) // in the off block
	img.SetRGBA(5, 1, stray) // in the lit block

	withIgnore := Palette{off: 0, onCol: 1, stray: Ignore}

	got, err := ImportPicture(img, 2, 1, withIgnore)
	if err != nil {
		t.Fatal(err)
	}

	if d, _ := Diff(f, got); d != "" {
		t.Errorf("stray marks changed the frame:\n%s", d)
	}

	// A block of nothing but stray marks can't be decided.
	allStray := scaled(f, 1, []color.RGBA{stray, onCol})
	if _, err := ImportPicture(allStray, 2, 1, withIgnore); !errors.Is(err, ErrMixedBlock) {
		t.Errorf("all-ignored block: err=%v, want ErrMixedBlock", err)
	}
}

func TestImportPictureErrors(t *testing.T) {
	t.Parallel()

	f := Frame{Width: 2, Height: 2, Pix: []byte{0, 1, 1, 0}}

	mixed := scaled(f, 4, []color.RGBA{off, onCol})
	mixed.SetRGBA(1, 1, onCol)

	unknown := scaled(f, 4, []color.RGBA{off, onCol})
	unknown.SetRGBA(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})

	tests := []struct {
		name string
		img  image.Image
		w, h int
		want error
	}{
		{"mixed block", mixed, 2, 2, ErrMixedBlock},
		{"unknown colour", unknown, 2, 2, ErrUnknownColour},
		{"not a multiple", image.NewRGBA(image.Rect(0, 0, 9, 8)), 2, 2, ErrNotScaled},
		{"different x and y scale", image.NewRGBA(image.Rect(0, 0, 8, 4)), 2, 2, ErrNotScaled},
	}

	for _, tt := range tests {
		if _, err := ImportPicture(tt.img, tt.w, tt.h, pal); !errors.Is(err, tt.want) {
			t.Errorf("%s: err=%v, want %v", tt.name, err, tt.want)
		}
	}
}

func TestRunAppliesScript(t *testing.T) {
	t.Parallel()

	// V0 = mem[0x1FF]; then copy key state: if key 5 down, V1 = 1.
	rom := []byte{
		0xA1, 0xFF, // I = 0x1FF
		0xF0, 0x65, // V0 = mem[I]
		0x62, 0x05, // V2 = 5
		0xE2, 0xA1, // skip if key V2 up
		0x61, 0x01, // V1 = 1
		0x12, 0x0A, // loop
	}

	m, err := Run(rom, chip8.DefaultProfile(chip8.ModernChip8), Script{
		Pokes:  []Poke{{Addr: 0x1FF, Value: 7}},
		Keys:   []KeyEvent{{Key: 5, Press: 0}},
		Frames: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if s := m.State(); s.V[0] != 7 || s.V[1] != 1 {
		t.Errorf("V0=%d V1=%d, want 7 1", s.V[0], s.V[1])
	}

	_, err = Run([]byte{0x00, 0xEE}, chip8.DefaultProfile(chip8.ModernChip8), Script{Frames: 1})
	if !errors.Is(err, chip8.ErrStackUnderflow) {
		t.Errorf("halting ROM: err=%v, want ErrStackUnderflow", err)
	}

	if _, err := Run(nil, chip8.DefaultProfile(chip8.SuperChip), Script{}); !errors.Is(err, chip8.ErrUnsupportedPlatform) {
		t.Errorf("unsupported platform: err=%v", err)
	}
}

func TestKeyEvents(t *testing.T) {
	t.Parallel()

	m := chip8.New(chip8.DefaultProfile(chip8.ModernChip8))
	events := []KeyEvent{{Key: 3, Press: 2, Release: 4}}

	for f, want := range []bool{false, false, true, true, false, false} {
		applyKeys(m, events, f)

		if got := m.KeyDown(3); got != want {
			t.Errorf("frame %d: key down=%v, want %v", f, got, want)
		}
	}
}
