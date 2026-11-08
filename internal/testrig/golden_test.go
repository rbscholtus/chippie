package testrig

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func frameOf(pix ...byte) Frame { return Frame{Width: len(pix), Height: 1, Pix: pix} }

func newChecker(t *testing.T, update bool) Checker {
	t.Helper()

	dir := t.TempDir()

	return Checker{GoldenDir: filepath.Join(dir, "golden"), FailureDir: filepath.Join(dir, "failures"), Update: update}
}

func writeTestGolden(t *testing.T, c Checker, name string, f Frame) {
	t.Helper()

	if err := os.MkdirAll(c.GoldenDir, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := WriteGolden(filepath.Join(c.GoldenDir, name), f); err != nil {
		t.Fatal(err)
	}
}

func TestCheckMatch(t *testing.T) {
	t.Parallel()

	c := newChecker(t, false)
	writeTestGolden(t, c, "a", frameOf(0, 1, 0))

	if err := c.Check(Golden{Name: "a", Imported: true}, frameOf(0, 1, 0)); err != nil {
		t.Errorf("matching frame: %v", err)
	}

	if _, err := os.Stat(c.FailureDir); !errors.Is(err, os.ErrNotExist) {
		t.Error("a match wrote failure files")
	}
}

func TestCheckMismatchWritesFailureFiles(t *testing.T) {
	t.Parallel()

	c := newChecker(t, false)
	want, got := frameOf(0, 1, 1, 0), frameOf(1, 1, 0, 0)
	writeTestGolden(t, c, "a", want)

	err := c.Check(Golden{Name: "a", Imported: true}, got)
	if !errors.Is(err, ErrGoldenMismatch) || !strings.Contains(err.Error(), "2 pixels differ") {
		t.Fatalf("err=%v, want a mismatch with an ASCII diff", err)
	}

	dir := filepath.Join(c.FailureDir, "a")

	actual, err := LoadGolden(filepath.Join(dir, "actual.png"))
	if err != nil {
		t.Fatal(err)
	}

	if d, _ := Diff(got, actual); d != "" {
		t.Errorf("actual.png isn't the frame that ran:\n%s", d)
	}

	fh, err := os.Open(filepath.Join(dir, "diff.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()

	img, err := png.Decode(fh)
	if err != nil {
		t.Fatal(err)
	}

	if b := img.Bounds(); b.Dx() != 4*diffScale || b.Dy() != diffScale {
		t.Errorf("diff.png is %v, want %d×%d", b, 4*diffScale, diffScale)
	}

	txt, err := os.ReadFile(filepath.Join(dir, "diff.txt"))
	if err != nil || !bytes.Contains(txt, []byte(">")) || !bytes.Contains(txt, []byte("X X")) {
		t.Errorf("diff.txt = %q, %v", txt, err)
	}
}

func TestCheckUpdateRewritesGeneratedOnly(t *testing.T) {
	t.Parallel()

	c := newChecker(t, true)
	old, current := frameOf(0, 0), frameOf(1, 1)

	writeTestGolden(t, c, "gen", old)
	writeTestGolden(t, c, "imp", old)

	if err := c.Check(Golden{Name: "gen"}, current); err != nil {
		t.Fatalf("-update on a generated golden: %v", err)
	}

	c.Update = false
	if err := c.Check(Golden{Name: "gen"}, current); err != nil {
		t.Errorf("generated golden wasn't rewritten: %v", err)
	}

	c.Update = true
	if err := c.Check(Golden{Name: "imp", Imported: true}, current); !errors.Is(err, ErrGoldenMismatch) {
		t.Errorf("-update overwrote an imported golden: err=%v", err)
	}

	stored, err := LoadGolden(filepath.Join(c.GoldenDir, "imp.png"))
	if err != nil {
		t.Fatal(err)
	}

	if d, _ := Diff(old, stored); d != "" {
		t.Error("imported golden changed on disk")
	}
}

func TestCheckMissingGolden(t *testing.T) {
	t.Parallel()

	c := newChecker(t, false)

	err := c.Check(Golden{Name: "gen"}, frameOf(1))
	if !errors.Is(err, ErrGoldenMissing) || !strings.Contains(err.Error(), "-update") {
		t.Errorf("missing generated golden: err=%v", err)
	}

	if err := c.Check(Golden{Name: "imp", Imported: true}, frameOf(1)); !strings.Contains(fmtErr(err), "golden-import") {
		t.Errorf("missing imported golden: err=%v", err)
	}

	c.Update = true

	if err := os.MkdirAll(c.GoldenDir, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := c.Check(Golden{Name: "gen"}, frameOf(1)); err != nil {
		t.Errorf("-update didn't create a missing generated golden: %v", err)
	}

	if _, err := os.Stat(filepath.Join(c.GoldenDir, "gen.txt")); err != nil {
		t.Errorf("gen.txt not written: %v", err)
	}
}

func fmtErr(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}

func TestDiffImageColours(t *testing.T) {
	t.Parallel()

	img := DiffImage(frameOf(1, 1, 0, 0), frameOf(1, 0, 1, 0), 1)

	for x, want := range []any{diffLit, diffMissing, diffWrong, diffOff} {
		if got := img.RGBAAt(x, 0); got != want {
			t.Errorf("pixel %d = %v, want %v", x, got, want)
		}
	}
}

func TestClearFailures(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "failures")
	if err := os.MkdirAll(filepath.Join(dir, "old"), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := ClearFailures(dir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Error("failure directory still there")
	}
}
