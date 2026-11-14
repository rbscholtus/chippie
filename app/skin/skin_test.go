package skin

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

func testSkin() Skin {
	return Skin{
		Name: "Test set", Width: 1000, Height: 800, Background: "#d0cecf", Cabinet: "cabinet", Mask: "cabinet-mask",
		Screen: Rect{X: 50, Y: 50, W: 600, H: 500},
		Controls: []Control{
			{ID: "channel", Kind: Dial, Image: "channel", CX: 800, CY: 150, R: 80, Steps: 13},
			{ID: "volume", Kind: Knob, Image: "volume", CX: 760, CY: 400, R: 20, MinAngle: -135, MaxAngle: 135},
			{ID: "tint", Kind: Knob, Image: "tint", CX: 840, CY: 400, R: 20, MinAngle: -60, MaxAngle: 60},
			{ID: "power", Kind: Rocker, Image: "power", X: 900, Y: 380, W: 20, H: 40},
		},
	}
}

func TestParseValid(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(testSkin())
	if err != nil {
		t.Fatal(err)
	}

	s, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	if c, ok := s.Control("tint"); !ok || c.R != 20 {
		t.Errorf("Control(tint) = %+v, %v", c, ok)
	}

	if c := s.Colour(); c != [3]uint8{0xd0, 0xce, 0xcf} {
		t.Errorf("Colour = %x", c)
	}
}

func TestValidateRejects(t *testing.T) {
	t.Parallel()

	for name, change := range map[string]func(*Skin){
		"no size":         func(s *Skin) { s.Width = 0 },
		"no mask":         func(s *Skin) { s.Mask = "" },
		"bad colour":      func(s *Skin) { s.Background = "grey" },
		"screen outside":  func(s *Skin) { s.Screen.W = 2000 },
		"control outside": func(s *Skin) { s.Controls[0].CX = 990 },
		"over the screen": func(s *Skin) { s.Controls[1].CX, s.Controls[1].CY = 600, 300 },
		"overlapping":     func(s *Skin) { s.Controls[2].CX = 780 },
		"duplicate id":    func(s *Skin) { s.Controls[2].ID = "volume" },
		"dial no steps":   func(s *Skin) { s.Controls[0].Steps = 0 },
		"knob no range":   func(s *Skin) { s.Controls[1].MaxAngle = -135 },
		"rocker no size":  func(s *Skin) { s.Controls[3].W = 0 },
		"unknown kind":    func(s *Skin) { s.Controls[3].Kind = "slider" },
		"no image":        func(s *Skin) { s.Controls[0].Image = "" },
	} {
		s := testSkin()
		change(&s)

		if err := s.Validate(); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: Validate = %v, want ErrInvalid", name, err)
		}
	}

	if _, err := Parse([]byte("{")); !errors.Is(err, ErrInvalid) {
		t.Errorf("Parse(bad JSON) = %v", err)
	}
}

func TestScaleFor(t *testing.T) {
	t.Parallel()

	s := testSkin() // 1000 wide at @3x: @1x is 333, @2x 667

	for _, tc := range []struct {
		width float64
		want  int
	}{{100, 1}, {333, 1}, {334, 2}, {666, 2}, {667, 3}, {5000, 3}} {
		if got := s.ScaleFor(tc.width); got != tc.want {
			t.Errorf("ScaleFor(%g) = %d, want %d", tc.width, got, tc.want)
		}
	}

	if got := FileName("cabinet", 2, "jpg"); got != "cabinet@2x.jpg" {
		t.Errorf("FileName = %q", got)
	}
}

func TestHitTest(t *testing.T) {
	t.Parallel()

	s := testSkin()

	for _, tc := range []struct {
		name   string
		x, y   float64
		minR   float64
		want   string
		wantOK bool
	}{
		{"dial centre", 800, 150, 0, "channel", true},
		{"dial edge", 870, 150, 0, "channel", true},
		{"outside all", 500, 700, 0, "", false},
		{"just outside volume", 783, 400, 0, "", false},
		{"grown volume", 735, 400, 30, "volume", true},
		{"overlap: nearest wins", 795, 400, 40, "volume", true},
		{"overlap: other side", 805, 400, 40, "tint", true},
		{"rocker", 910, 400, 0, "power", true},
		{"grown rocker", 925, 400, 20, "power", true},
	} {
		c, ok := s.HitTest(tc.x, tc.y, tc.minR)
		if ok != tc.wantOK || c.ID != tc.want {
			t.Errorf("%s: HitTest = %q, %v; want %q, %v", tc.name, c.ID, ok, tc.want, tc.wantOK)
		}
	}
}

func TestAngles(t *testing.T) {
	t.Parallel()

	c := Control{Kind: Dial, CX: 100, CY: 100, R: 50, Steps: 13}

	for _, tc := range []struct {
		x, y, want float64
	}{{100, 50, 0}, {150, 100, 90}, {100, 150, 180}, {50, 100, -90}} {
		if got := Angle(c, tc.x, tc.y); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("Angle(%g, %g) = %g, want %g", tc.x, tc.y, got, tc.want)
		}
	}

	for _, tc := range []struct{ a, b, want float64 }{
		{10, 20, 10}, {170, -170, 20}, {-170, 170, -20}, {0, 180, 180}, {90, -90, 180},
	} {
		if got := AngleDelta(tc.a, tc.b); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("AngleDelta(%g, %g) = %g, want %g", tc.a, tc.b, got, tc.want)
		}
	}

	if ClickDirection(c, 99) != -1 || ClickDirection(c, 101) != 1 {
		t.Error("ClickDirection: left half should step down, right half up")
	}

	if got := DialAngle(c, 14); math.Abs(got+360.0/13) > 1e-9 {
		t.Errorf("DialAngle(14) = %g", got)
	}

	k := Control{Kind: Knob, MinAngle: -135, MaxAngle: 135}
	if KnobAngle(k, 0) != -135 || KnobAngle(k, 0.5) != 0 || KnobAngle(k, 2) != 135 {
		t.Error("KnobAngle: wrong mapping or no clamping")
	}
}

func TestTurn(t *testing.T) {
	t.Parallel()

	var turn Turn

	per := 360.0 / 12 // 30° per stop

	if n := turn.Add(20, 12); n != 0 {
		t.Errorf("20°: %d steps", n)
	}

	if n := turn.Add(20, 12); n != 1 { // 40° in total
		t.Errorf("40°: %d steps", n)
	}

	if n := turn.Add(-2.5*per, 12); n != -2 { // back to -35°
		t.Errorf("back 75°: %d steps", n)
	}

	turn.Reset()

	if n := turn.Add(per*3, 12); n != 3 {
		t.Errorf("after reset, 90°: %d steps", n)
	}
}
