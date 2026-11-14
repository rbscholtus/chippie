// Command skin-gen turns the TV artwork into the files the app loads
// (README §12). It reads the generated master image from assets/source/
// (not committed: it's large) and the measurements in
// assets/tv/recipe.json, and writes into assets/tv/:
//
//	cabinet@1x.jpg … @3x.jpg       the cabinet, cropped, edges faded into the background
//	cabinet-mask@1x.png … @3x.png  black where the picture tube shows through
//	<control>@1x.png … @3x.png     each control's sprite, cut from the cabinet
//	skin.json                      where everything is, at @3x
//
// @3x is the cropped master's full size; @2x and @1x are two thirds and one
// third of it. Run it again whenever the art or the recipe changes:
//
//	go run ./cmd/skin-gen
//
// To check the alignment, also write a preview with the controls turned:
//
//	go run ./cmd/skin-gen -preview /tmp/tv.png -angle 30
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/image/font/gofont/gomedium"

	"github.com/rbscholtus/chippie/app/skin"
)

const (
	jpegQuality  = 85
	backdropFile = "backdrop.jpg" // one size: it's blurred anyway
)

var errRecipe = errors.New("skin-gen: bad recipe")

// recipe is assets/tv/recipe.json, in the master image's pixels.
type recipe struct {
	Source     string     `json:"source"`
	Crop       rectJSON   `json:"crop"`
	PaintOut   []rectJSON `json:"paintOut"`
	ScreenSeed pointJSON  `json:"screenSeed"`
	Feather    int        `json:"feather"`
	// Backdrop is how far the backdrop reaches beyond the crop, and how
	// much it is shrunk.
	Backdrop struct {
		Pad    int `json:"pad"`
		Shrink int `json:"shrink"`
	} `json:"backdrop"`
	Controls []controlRec `json:"controls"`
}

type rectJSON struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

func (r rectJSON) rect() image.Rectangle { return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H) }

type pointJSON struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// controlRec is one control: a circle (cx, cy, r) or a rectangle.
type controlRec struct {
	ID       string    `json:"id"`
	Kind     skin.Kind `json:"kind"`
	CX       int       `json:"cx"`
	CY       int       `json:"cy"`
	R        int       `json:"r"`
	X        int       `json:"x"`
	Y        int       `json:"y"`
	W        int       `json:"w"`
	H        int       `json:"h"`
	Steps    int       `json:"steps"`
	MinAngle float64   `json:"minAngle"`
	MaxAngle float64   `json:"maxAngle"`
	Erase    *struct {
		Inner float64 `json:"inner"`
		Outer float64 `json:"outer"`
		// Keep are rectangles, relative to the centre, left untouched.
		Keep []rectJSON `json:"keep"`
	} `json:"erase"`
	Labels      []string `json:"labels"`
	LabelRadius float64  `json:"labelRadius"`
	LabelSize   float64  `json:"labelSize"`
	LabelColour string   `json:"labelColour"`
}

func main() {
	src := flag.String("source", "assets/source", "directory of the master images")
	dir := flag.String("dir", "assets/tv", "recipe in, outputs out")
	preview := flag.String("preview", "", "also write a composed preview here, e.g. /tmp/tv.png")
	angle := flag.Float64("angle", 0, "turn the round controls this many degrees in the preview")

	flag.Parse()

	if err := run(*src, *dir, *preview, *angle); err != nil {
		log.Fatal(err)
	}
}

// outputs are everything skin-gen writes, at @3x.
type outputs struct {
	skin     skin.Skin
	cabinet  *image.NRGBA
	mask     *image.Gray
	backdrop *image.NRGBA
	sprites  map[string]*image.NRGBA
}

func run(srcDir, dir, preview string, angle float64) error {
	var rc recipe
	if err := readJSON(filepath.Join(dir, "recipe.json"), &rc); err != nil {
		return err
	}

	master, err := readPNG(filepath.Join(srcDir, rc.Source))
	if err != nil {
		return err
	}

	out, err := build(&rc, master)
	if err != nil {
		return err
	}

	if err := writeScaled(dir, &out); err != nil {
		return err
	}

	if preview != "" {
		if err := writePreview(preview, &out, angle); err != nil {
			return err
		}
	}

	return writeSkin(filepath.Join(dir, "skin.json"), &out.skin)
}

// build makes the outputs from the master image, following the recipe.
func build(rc *recipe, master *image.NRGBA) (outputs, error) {
	for _, r := range rc.PaintOut {
		paintOut(master, r.rect())
	}

	crop := rc.Crop.rect()
	if !crop.In(master.Bounds()) {
		return outputs{}, fmt.Errorf("%w: crop %v outside the %v image", errRecipe, crop, master.Bounds())
	}

	mask, tube := tubeMask(master, image.Pt(rc.ScreenSeed.X, rc.ScreenSeed.Y))

	out := outputs{
		cabinet: toNRGBA(master.SubImage(crop)),
		mask:    toGray(mask, crop),
		sprites: map[string]*image.NRGBA{},
	}
	fill := borderColour(out.cabinet)
	fadeMask(out.mask, rc.Feather)

	pad, shrink := rc.Backdrop.Pad, rc.Backdrop.Shrink
	if pad <= 0 || shrink <= 0 {
		return outputs{}, fmt.Errorf("%w: backdrop pad and shrink must be positive", errRecipe)
	}

	out.backdrop = makeBackdrop(master, crop, pad, shrink, fill)

	out.skin = skin.Skin{
		Width: float64(crop.Dx()), Height: float64(crop.Dy()),
		Background: fmt.Sprintf("#%02x%02x%02x", fill.R, fill.G, fill.B),
		Cabinet:    "cabinet", Mask: "cabinet-mask",
		Backdrop: backdropFile,
		BackdropRect: skin.Rect{
			X: float64(-pad), Y: float64(-pad), W: float64(crop.Dx() + 2*pad), H: float64(crop.Dy() + 2*pad),
		},
		Screen: skin.Rect{
			X: float64(tube.Min.X - crop.Min.X), Y: float64(tube.Min.Y - crop.Min.Y),
			W: float64(tube.Dx()), H: float64(tube.Dy()),
		},
	}

	for _, c := range rc.Controls {
		sprite, ctl, err := cutControl(master, c, crop.Min)
		if err != nil {
			return outputs{}, err
		}

		out.sprites[c.ID] = sprite
		out.skin.Controls = append(out.skin.Controls, ctl)
	}

	if err := out.skin.Validate(); err != nil {
		return outputs{}, err //nolint:wrapcheck // ErrInvalid says what's wrong
	}

	return out, nil
}

// cutControl cuts one control's sprite out of the master and describes it
// relative to the crop.
func cutControl(master *image.NRGBA, c controlRec, origin image.Point) (*image.NRGBA, skin.Control, error) {
	ctl := skin.Control{
		ID: c.ID, Kind: c.Kind, Image: c.ID, Steps: c.Steps, MinAngle: c.MinAngle, MaxAngle: c.MaxAngle,
	}

	if c.Kind == skin.Rocker {
		r := image.Rect(c.X, c.Y, c.X+c.W, c.Y+c.H)
		ctl.X, ctl.Y = float64(c.X-origin.X), float64(c.Y-origin.Y)
		ctl.W, ctl.H = float64(c.W), float64(c.H)

		return toNRGBA(master.SubImage(r)), ctl, nil
	}

	ctl.CX, ctl.CY, ctl.R = float64(c.CX-origin.X), float64(c.CY-origin.Y), float64(c.R)
	sprite := roundSprite(master, c.CX, c.CY, c.R)

	if c.Erase != nil {
		keep := make([]image.Rectangle, 0, len(c.Erase.Keep))
		for _, k := range c.Erase.Keep {
			keep = append(keep, k.rect())
		}

		eraseRing(sprite, c.Erase.Inner, c.Erase.Outer, keep)
	}

	if len(c.Labels) > 0 {
		face, err := newFace(gomedium.TTF, c.LabelSize)
		if err != nil {
			return nil, ctl, fmt.Errorf("%s: font: %w", c.ID, err)
		}

		ink, err := parseColour(c.LabelColour)
		if err != nil {
			return nil, ctl, fmt.Errorf("%s: %w", c.ID, err)
		}

		drawLabels(sprite, c.Labels, c.LabelRadius, face, ink)
	}

	return sprite, ctl, nil
}

// toGray crops the mask to the cabinet.
func toGray(m *image.Gray, crop image.Rectangle) *image.Gray {
	out := image.NewGray(image.Rect(0, 0, crop.Dx(), crop.Dy()))

	for y := range crop.Dy() {
		copy(out.Pix[y*out.Stride:y*out.Stride+crop.Dx()], m.Pix[m.PixOffset(crop.Min.X, crop.Min.Y+y):])
	}

	return out
}

// writeScaled writes every image at @1x, @2x and @3x.
func writeScaled(dir string, out *outputs) error {
	w, h := out.cabinet.Bounds().Dx(), out.cabinet.Bounds().Dy()

	if err := writeFile(filepath.Join(dir, backdropFile), func(f *os.File) error {
		return jpeg.Encode(f, out.backdrop, &jpeg.Options{Quality: jpegQuality})
	}); err != nil {
		return err
	}

	for n := 1; n <= skin.MaxScale; n++ {
		size := func(v int) int { return int(math.Round(float64(v*n) / skin.MaxScale)) }

		if err := writeFile(filepath.Join(dir, skin.FileName("cabinet", n, "jpg")), func(f *os.File) error {
			return jpeg.Encode(f, scaleTo(out.cabinet, size(w), size(h)), &jpeg.Options{Quality: jpegQuality})
		}); err != nil {
			return err
		}

		if err := writePNG(filepath.Join(dir, skin.FileName("cabinet-mask", n, "png")),
			scaleGray(out.mask, size(w), size(h))); err != nil {
			return err
		}

		for id, s := range out.sprites {
			b := s.Bounds()
			if err := writePNG(filepath.Join(dir, skin.FileName(id, n, "png")),
				scaleTo(s, size(b.Dx()), size(b.Dy()))); err != nil {
				return err
			}
		}
	}

	return nil
}

func parseColour(s string) (color.NRGBA, error) {
	if len(s) != len("#rrggbb") || s[0] != '#' {
		return color.NRGBA{}, fmt.Errorf("%w: colour %q", errRecipe, s)
	}

	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("%w: colour %q", errRecipe, s)
	}

	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xFF}, nil //nolint:gosec // G115: 24-bit value
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read recipe: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%w: %s: %w", errRecipe, path, err)
	}

	return nil
}

func readPNG(path string) (*image.NRGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open master image: %w", err)
	}
	defer f.Close()

	m, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	return toNRGBA(m), nil
}

func writePNG(path string, m image.Image) error {
	return writeFile(path, func(f *os.File) error {
		enc := png.Encoder{CompressionLevel: png.BestCompression}

		return enc.Encode(f, m)
	})
}

func writeSkin(path string, s *skin.Skin) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode skin: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write skin: %w", err)
	}

	return nil
}

func writeFile(path string, encode func(*os.File) error) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}

	if err := encode(f); err != nil {
		_ = f.Close() // the encode error is the one to report

		return fmt.Errorf("encode %s: %w", path, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}
