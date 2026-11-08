package testrig

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Errors from checking goldens.
var (
	ErrGoldenMissing  = errors.New("testrig: golden frame missing")
	ErrGoldenMismatch = errors.New("testrig: frame differs from golden")
)

// Golden names an expected frame in the golden directory.
type Golden struct {
	Name string
	// Imported goldens come from the test suite's own pictures (written by
	// cmd/golden-import). They are never overwritten by -update.
	Imported bool
}

// Checker compares frames with goldens, writes failure files for
// mismatches, and with Update rewrites generated (not imported) goldens.
type Checker struct {
	GoldenDir  string
	FailureDir string
	Update     bool
}

// Check compares got with the golden. It returns nil when they match or
// when a generated golden was (re)written with Update; otherwise an error
// that includes an ASCII diff and where the failure files are.
func (c Checker) Check(g Golden, got Frame) error {
	base := filepath.Join(c.GoldenDir, g.Name)

	want, err := LoadGolden(base + ".png")

	switch {
	case errors.Is(err, fs.ErrNotExist):
		if c.Update && !g.Imported {
			return WriteGolden(base, got)
		}

		hint := "run go test ./... -update, check it by eye, and commit it"
		if g.Imported {
			hint = "run go run ./cmd/golden-import"
		}

		return fmt.Errorf("%w: %s (%s)", ErrGoldenMissing, g.Name, hint)
	case err != nil:
		return err
	}

	diff, err := Diff(want, got)
	if err == nil && diff == "" {
		return nil
	}

	if c.Update && !g.Imported {
		return WriteGolden(base, got)
	}

	dir := filepath.Join(c.FailureDir, g.Name)
	if werr := WriteFailure(dir, want, got); werr != nil {
		return fmt.Errorf("%w: %s; also writing failure files: %w", ErrGoldenMismatch, g.Name, werr)
	}

	if err != nil {
		return fmt.Errorf("%w: %s: %w; see %s", ErrGoldenMismatch, g.Name, err, dir)
	}

	return fmt.Errorf("%w: %s; see %s\n%s", ErrGoldenMismatch, g.Name, dir, diff)
}

// diffScale enlarges diff.png so single pixels are visible.
const diffScale = 8

// WriteFailure writes actual.png, diff.png and diff.txt into dir.
func WriteFailure(dir string, want, got Frame) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create failure directory: %w", err)
	}

	if err := writePNG(filepath.Join(dir, "actual.png"), got.EncodePNG); err != nil {
		return err
	}

	diffImg := DiffImage(want, got, diffScale)

	if err := writePNG(filepath.Join(dir, "diff.png"), func(w io.Writer) error { return encode(w, diffImg) }); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, "diff.txt"), []byte(SideBySide(want, got)), 0o600); err != nil {
		return fmt.Errorf("write diff.txt: %w", err)
	}

	return nil
}

func writePNG(path string, write func(io.Writer) error) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}

	if err := write(f); err != nil {
		_ = f.Close()

		return err
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}

	return nil
}

func encode(w io.Writer, img image.Image) error {
	if err := png.Encode(w, img); err != nil {
		return fmt.Errorf("encode PNG: %w", err)
	}

	return nil
}

// Diff image colours.
var (
	diffOff     = color.RGBA{A: 0xFF}                            //nolint:gochecknoglobals // read-only colour
	diffLit     = color.RGBA{R: 0x50, G: 0x50, B: 0x50, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	diffWrong   = color.RGBA{R: 0xFF, G: 0x20, B: 0x20, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	diffMissing = color.RGBA{R: 0xFF, G: 0xA0, B: 0x00, A: 0xFF} //nolint:gochecknoglobals // read-only colour
)

// DiffImage shows the expected frame dimmed, with pixels that are lit but
// shouldn't be in red and pixels that should be lit but aren't in orange.
// Frames of different sizes are compared over their common area.
func DiffImage(want, got Frame, scale int) *image.RGBA {
	w, h := max(want.Width, got.Width), max(want.Height, got.Height)
	img := image.NewRGBA(image.Rect(0, 0, w*scale, h*scale))

	at := func(f Frame, x, y int) byte {
		if x >= f.Width || y >= f.Height {
			return 0
		}

		return f.At(x, y)
	}

	for y := range h {
		for x := range w {
			wv, gv := at(want, x, y), at(got, x, y)

			c := diffOff

			switch {
			case wv != gv && gv != 0:
				c = diffWrong
			case wv != gv:
				c = diffMissing
			case wv != 0:
				c = diffLit
			}

			for dy := range scale {
				for dx := range scale {
					img.SetRGBA(x*scale+dx, y*scale+dy, c)
				}
			}
		}
	}

	return img
}

// SideBySide shows want and got next to each other, with a marker column
// for rows that differ and an X under each differing pixel.
func SideBySide(want, got Frame) string {
	wl := strings.Split(strings.TrimRight(want.Text(), "\n"), "\n")
	gl := strings.Split(strings.TrimRight(got.Text(), "\n"), "\n")

	var b strings.Builder

	fmt.Fprintf(&b, "   %-*s   %s\n", want.Width, "want", "got")

	for y := range max(len(wl), len(gl)) {
		var w, g string
		if y < len(wl) {
			w = wl[y]
		}

		if y < len(gl) {
			g = gl[y]
		}

		mark := " "
		if w != g {
			mark = ">"
		}

		fmt.Fprintf(&b, "%s%2d %-*s   %s\n", mark, y, want.Width, w, g)

		if w != g {
			b.WriteString("   " + strings.Repeat(" ", want.Width) + "   " + markDiff(w, g) + "\n")
		}
	}

	return b.String()
}

func markDiff(a, b string) string {
	n := max(len(a), len(b))
	out := make([]byte, n)

	for i := range n {
		out[i] = ' '
		if i >= len(a) || i >= len(b) || a[i] != b[i] {
			out[i] = 'X'
		}
	}

	return string(out)
}

// ClearFailures removes old failure files, so only this run's are left.
func ClearFailures(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clear failures: %w", err)
	}

	return nil
}
