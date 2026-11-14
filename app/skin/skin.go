// Package skin describes the TV artwork: where the cabinet's picture tube
// and controls are, in the cabinet image's own pixels (README §12.1). It
// parses and checks skin.json and does the geometry (which control is under
// a point, how a turn maps to steps). It holds no Ebitengine state, so
// cmd/skin-gen can write skin.json with the same types and the maths can be
// tested directly.
package skin

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
)

// Kind is how a control moves.
type Kind string

// Control kinds.
const (
	Dial   Kind = "dial"   // turns in click-stops, forever (the channel selector)
	Knob   Kind = "knob"   // turns between two end stops
	Rocker Kind = "rocker" // flips between two positions (the power switch)
)

// MaxScale is the largest pre-scaled size: @3x is the cabinet at full size.
const MaxScale = 3

// ErrInvalid reports a skin.json that doesn't describe a usable TV.
var ErrInvalid = errors.New("skin: invalid skin.json")

// Rect is a rectangle in cabinet pixels (at @3x).
type Rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// Contains reports whether (x, y) is inside r.
func (r Rect) Contains(x, y float64) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Inside reports whether r lies within o.
func (r Rect) Inside(o Rect) bool {
	return r.X >= o.X && r.Y >= o.Y && r.X+r.W <= o.X+o.W && r.Y+r.H <= o.Y+o.H
}

// Overlaps reports whether r and o share any area.
func (r Rect) Overlaps(o Rect) bool {
	return r.X < o.X+o.W && o.X < r.X+r.W && r.Y < o.Y+o.H && o.Y < r.Y+r.H
}

// Control is one knob, dial or switch. Round controls (dials, knobs) use
// the centre and radius; rockers use the rectangle.
type Control struct {
	ID    string  `json:"id"`
	Kind  Kind    `json:"kind"`
	Image string  `json:"image"` // base name of its sprite: <image>@<n>x.png
	CX    float64 `json:"cx,omitempty"`
	CY    float64 `json:"cy,omitempty"`
	R     float64 `json:"r,omitempty"`
	X     float64 `json:"x,omitempty"`
	Y     float64 `json:"y,omitempty"`
	W     float64 `json:"w,omitempty"`
	H     float64 `json:"h,omitempty"`
	// Steps is a dial's click-stops per turn.
	Steps int `json:"steps,omitempty"`
	// MinAngle and MaxAngle are a knob's end stops, in degrees clockwise
	// from the position in the art (pointing up).
	MinAngle float64 `json:"minAngle,omitempty"`
	MaxAngle float64 `json:"maxAngle,omitempty"`
}

// Round reports whether the control is a circle.
func (c Control) Round() bool { return c.Kind == Dial || c.Kind == Knob }

// Bounds is the control's sprite rectangle.
func (c Control) Bounds() Rect {
	if c.Round() {
		return Rect{X: c.CX - c.R, Y: c.CY - c.R, W: 2 * c.R, H: 2 * c.R}
	}

	return Rect{X: c.X, Y: c.Y, W: c.W, H: c.H}
}

// Centre is the middle of the control.
func (c Control) Centre() (float64, float64) {
	b := c.Bounds()

	return b.X + b.W/2, b.Y + b.H/2
}

// Skin is a parsed skin.json.
type Skin struct {
	Width  float64 `json:"width"`  // cabinet image size at @3x
	Height float64 `json:"height"` //
	// Background fills the page around the cabinet: "#rrggbb", the colour
	// the cabinet image's edges fade into.
	Background string `json:"background"`
	// Cabinet and Mask are base names: <cabinet>@<n>x.jpg and
	// <mask>@<n>x.png, a greyscale image that is black where the picture
	// tube shows through.
	Cabinet string `json:"cabinet"`
	Mask    string `json:"mask"`
	// Backdrop is a small blurred image of the background around the
	// cabinet, drawn behind it at BackdropRect (which reaches beyond the
	// cabinet), so the cabinet's faded edges meet what they were cut from.
	Backdrop     string `json:"backdrop"`
	BackdropRect Rect   `json:"backdropRect"`
	// Screen is the picture tube's bounding box; the mask gives its shape.
	Screen   Rect      `json:"screen"`
	Controls []Control `json:"controls"`
}

// Parse reads and checks skin.json.
func Parse(data []byte) (*Skin, error) {
	var s Skin
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalid, err)
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}

	return &s, nil
}

// Validate checks that everything lies within the cabinet, controls don't
// overlap each other or the screen, and each control is complete.
func (s *Skin) Validate() error {
	cabinet := Rect{W: s.Width, H: s.Height}

	switch {
	case s.Width <= 0 || s.Height <= 0:
		return fmt.Errorf("%w: cabinet size %gx%g", ErrInvalid, s.Width, s.Height)
	case s.Cabinet == "" || s.Mask == "":
		return fmt.Errorf("%w: cabinet or mask image missing", ErrInvalid)
	case !validColour(s.Background):
		return fmt.Errorf("%w: background %q is not #rrggbb", ErrInvalid, s.Background)
	case s.Screen.W <= 0 || s.Screen.H <= 0 || !s.Screen.Inside(cabinet):
		return fmt.Errorf("%w: screen %+v not inside the cabinet", ErrInvalid, s.Screen)
	}

	seen := map[string]bool{}

	for i, c := range s.Controls {
		if err := c.validate(); err != nil {
			return err
		}

		b := c.Bounds()

		switch {
		case seen[c.ID]:
			return fmt.Errorf("%w: control %q twice", ErrInvalid, c.ID)
		case !b.Inside(cabinet):
			return fmt.Errorf("%w: control %q outside the cabinet", ErrInvalid, c.ID)
		case b.Overlaps(s.Screen):
			return fmt.Errorf("%w: control %q overlaps the screen", ErrInvalid, c.ID)
		}

		seen[c.ID] = true

		for _, o := range s.Controls[:i] {
			if b.Overlaps(o.Bounds()) {
				return fmt.Errorf("%w: controls %q and %q overlap", ErrInvalid, o.ID, c.ID)
			}
		}
	}

	return nil
}

func (c Control) validate() error {
	switch {
	case c.ID == "" || c.Image == "":
		return fmt.Errorf("%w: control without id or image", ErrInvalid)
	case c.Round() && c.R <= 0:
		return fmt.Errorf("%w: control %q has no radius", ErrInvalid, c.ID)
	case c.Kind == Rocker && (c.W <= 0 || c.H <= 0):
		return fmt.Errorf("%w: control %q has no size", ErrInvalid, c.ID)
	case c.Kind == Dial && c.Steps <= 0:
		return fmt.Errorf("%w: dial %q has no steps", ErrInvalid, c.ID)
	case c.Kind == Knob && c.MaxAngle <= c.MinAngle:
		return fmt.Errorf("%w: knob %q has no range", ErrInvalid, c.ID)
	case !c.Round() && c.Kind != Rocker:
		return fmt.Errorf("%w: control %q has unknown kind %q", ErrInvalid, c.ID, c.Kind)
	}

	return nil
}

// Control returns the control with the id.
func (s *Skin) Control(id string) (Control, bool) {
	for _, c := range s.Controls {
		if c.ID == id {
			return c, true
		}
	}

	return Control{}, false
}

// validColour reports whether s is "#rrggbb".
func validColour(s string) bool {
	if len(s) != len("#rrggbb") || s[0] != '#' {
		return false
	}

	_, err := strconv.ParseUint(s[1:], 16, 32)

	return err == nil
}

// Colour returns the background as red, green and blue bytes; Validate has
// checked it.
func (s *Skin) Colour() [3]uint8 {
	v, _ := strconv.ParseUint(s.Background[1:], 16, 32) // checked by Validate

	return [3]uint8{uint8(v >> 16), uint8(v >> 8), uint8(v)} //nolint:gosec // G115: six hex digits fit 24 bits
}

// ScaleFor picks the pre-scaled size (1, 2 or 3) closest to drawing the
// cabinet width wide, preferring to scale down: the smallest one at least
// that wide, else the largest.
func (s *Skin) ScaleFor(width float64) int {
	for n := 1; n < MaxScale; n++ {
		if s.Width*float64(n)/MaxScale >= width {
			return n
		}
	}

	return MaxScale
}

// FileName is a pre-scaled file's name: "cabinet", 2, "jpg" → cabinet@2x.jpg.
func FileName(base string, scale int, ext string) string {
	return fmt.Sprintf("%s@%dx.%s", base, scale, ext)
}

// HitTest returns the control under (x, y), in cabinet pixels. Every
// control's hit area is at least minR from its centre (round) or edges
// (rocker), so small knobs stay easy to touch; where grown areas overlap,
// the nearest centre wins.
func (s *Skin) HitTest(x, y, minR float64) (Control, bool) {
	best, bestD := -1, math.Inf(1)

	for i, c := range s.Controls {
		cx, cy := c.Centre()
		d := math.Hypot(x-cx, y-cy)

		var hit bool

		if c.Round() {
			hit = d <= max(c.R, minR)
		} else {
			b := c.Bounds()
			gx, gy := max(0, minR-b.W/2), max(0, minR-b.H/2)
			hit = (Rect{X: b.X - gx, Y: b.Y - gy, W: b.W + 2*gx, H: b.H + 2*gy}).Contains(x, y)
		}

		if hit && d < bestD {
			best, bestD = i, d
		}
	}

	if best < 0 {
		return Control{}, false
	}

	return s.Controls[best], true
}

// Angle is the direction from a control's centre to (x, y), in degrees
// clockwise from straight up, in (-180, 180].
func Angle(c Control, x, y float64) float64 {
	cx, cy := c.Centre()

	return math.Atan2(x-cx, cy-y) * 180 / math.Pi
}

// AngleDelta is the shortest turn from a to b, in degrees, in (-180, 180].
func AngleDelta(a, b float64) float64 {
	d := math.Mod(b-a, 360)

	switch {
	case d > 180:
		d -= 360
	case d <= -180:
		d += 360
	}

	return d
}

// ClickDirection is +1 for a click on a control's right half and -1 for
// its left half: one step up or down.
func ClickDirection(c Control, x float64) int {
	if cx, _ := c.Centre(); x < cx {
		return -1
	}

	return 1
}

// Turn accumulates a drag around a dial into whole click-stops: it adds
// delta degrees and returns how many stops were passed (negative
// anticlockwise), keeping the remainder for the next call.
type Turn struct {
	rest float64
}

// Add adds a turn of delta degrees on a dial with steps stops per turn.
func (t *Turn) Add(delta float64, steps int) int {
	per := 360 / float64(steps)
	t.rest += delta
	n := math.Trunc(t.rest / per)
	t.rest -= n * per

	return int(n)
}

// Reset forgets any partial step.
func (t *Turn) Reset() { t.rest = 0 }

// KnobAngle maps a knob's value v in [0, 1] to its rotation in degrees.
func KnobAngle(c Control, v float64) float64 {
	return c.MinAngle + min(max(v, 0), 1)*(c.MaxAngle-c.MinAngle)
}

// DialAngle is a dial's rotation in degrees at click-stop n: each step turns
// it one stop anticlockwise, so the next number comes under the marker at
// the top.
func DialAngle(c Control, n int) float64 {
	return -float64(n%c.Steps) * 360 / float64(c.Steps)
}
