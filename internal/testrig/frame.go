package testrig

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"strings"

	"github.com/rbscholtus/chippie/chip8"
)

// Frame is a captured display: one byte of plane bits per pixel (0 = off).
type Frame struct {
	Width  int
	Height int
	Pix    []byte
}

// Errors from parsing and comparing frames.
var (
	ErrBadFrame     = errors.New("testrig: malformed frame")
	ErrSizeMismatch = errors.New("testrig: frame sizes differ")
)

// FromDisplay copies the display into a Frame.
func FromDisplay(v chip8.DisplayView) Frame {
	f := Frame{Width: v.Width(), Height: v.Height(), Pix: make([]byte, v.Width()*v.Height())}
	for y := range f.Height {
		for x := range f.Width {
			f.Pix[y*f.Width+x] = v.Pixel(x, y)
		}
	}

	return f
}

// At returns the plane bits at (x, y).
func (f Frame) At(x, y int) byte { return f.Pix[y*f.Width+x] }

// textGlyphs maps plane bits to characters in the .txt format.
const textGlyphs = ".#+@" // off, plane 1, plane 2, both

// Text returns the frame as ASCII art, one line per row.
func (f Frame) Text() string {
	var b strings.Builder

	for y := range f.Height {
		for x := range f.Width {
			b.WriteByte(textGlyphs[f.At(x, y)&3])
		}

		b.WriteByte('\n')
	}

	return b.String()
}

// ParseText reads a frame in the format written by [Frame.Text].
func ParseText(s string) (Frame, error) {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return Frame{}, fmt.Errorf("%w: empty", ErrBadFrame)
	}

	f := Frame{Width: len(lines[0]), Height: len(lines)}
	f.Pix = make([]byte, 0, f.Width*f.Height)

	for y, line := range lines {
		if len(line) != f.Width {
			return Frame{}, fmt.Errorf("%w: line %d has %d characters, want %d", ErrBadFrame, y+1, len(line), f.Width)
		}

		for _, c := range []byte(line) {
			i := strings.IndexByte(textGlyphs, c)
			if i < 0 {
				return Frame{}, fmt.Errorf("%w: line %d: unexpected %q", ErrBadFrame, y+1, c)
			}

			f.Pix = append(f.Pix, byte(i)) //nolint:gosec // G115: i < len(textGlyphs) == 4
		}
	}

	return f, nil
}

// palette is used for golden PNGs: the palette index is the plane bits.
var palette = color.Palette{ //nolint:gochecknoglobals // read-only colour table
	color.Gray{Y: 0x00}, color.Gray{Y: 0xFF}, color.Gray{Y: 0x80}, color.Gray{Y: 0xC0},
}

// EncodePNG writes the frame as a paletted PNG at native resolution.
func (f Frame) EncodePNG(w io.Writer) error {
	img := image.NewPaletted(image.Rect(0, 0, f.Width, f.Height), palette)
	copy(img.Pix, f.Pix)

	if err := png.Encode(w, img); err != nil {
		return fmt.Errorf("encode PNG: %w", err)
	}

	return nil
}

// DecodePNG reads a frame written by [Frame.EncodePNG].
func DecodePNG(r io.Reader) (Frame, error) {
	img, err := png.Decode(r)
	if err != nil {
		return Frame{}, fmt.Errorf("decode PNG: %w", err)
	}

	p, ok := img.(*image.Paletted)
	if !ok {
		return Frame{}, fmt.Errorf("%w: PNG is not paletted", ErrBadFrame)
	}

	b := p.Bounds()
	f := Frame{Width: b.Dx(), Height: b.Dy(), Pix: make([]byte, b.Dx()*b.Dy())}

	for y := range f.Height {
		copy(f.Pix[y*f.Width:(y+1)*f.Width], p.Pix[y*p.Stride:])
	}

	return f, nil
}

// LoadGolden reads the golden PNG at path.
func LoadGolden(path string) (Frame, error) {
	fh, err := os.Open(path)
	if err != nil {
		return Frame{}, fmt.Errorf("open golden: %w", err)
	}
	defer fh.Close()

	return DecodePNG(fh)
}

// WriteGolden writes base+".png" and base+".txt".
func WriteGolden(base string, f Frame) error {
	fh, err := os.Create(base + ".png")
	if err != nil {
		return fmt.Errorf("create golden: %w", err)
	}

	if err := f.EncodePNG(fh); err != nil {
		_ = fh.Close()

		return err
	}

	if err := fh.Close(); err != nil {
		return fmt.Errorf("close golden: %w", err)
	}

	if err := os.WriteFile(base+".txt", []byte(f.Text()), 0o600); err != nil {
		return fmt.Errorf("write golden text: %w", err)
	}

	return nil
}

// Diff compares two frames. It returns "" when they are equal, and
// otherwise a readable report: the number of differing pixels and the rows
// that differ, showing want and got.
func Diff(want, got Frame) (string, error) {
	if want.Width != got.Width || want.Height != got.Height {
		return "", fmt.Errorf("%w: want %d×%d, got %d×%d",
			ErrSizeMismatch, want.Width, want.Height, got.Width, got.Height)
	}

	var b strings.Builder

	bad := 0
	wantLines := strings.Split(want.Text(), "\n")
	gotLines := strings.Split(got.Text(), "\n")

	for y := range want.Height {
		if wantLines[y] == gotLines[y] {
			continue
		}

		for x := range want.Width {
			if want.At(x, y) != got.At(x, y) {
				bad++
			}
		}

		fmt.Fprintf(&b, "row %2d want %s\n       got  %s\n", y, wantLines[y], gotLines[y])
	}

	if bad == 0 {
		return "", nil
	}

	return fmt.Sprintf("%d pixels differ:\n%s", bad, b.String()), nil
}
