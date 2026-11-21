package library

import (
	"strings"
	"testing"
)

func TestReflowJoinsWrappedProse(t *testing.T) {
	t.Parallel()

	in := "Blinky has two lives. Finishing a screen increases the intelligence\n" +
		"of Packlett and Heward, the two ghosts. The game is at its most difficult\n" +
		"on the fourth screen.\n" +
		"\n" +
		"My personal highscore is 1575 so far."
	want := "Blinky has two lives. Finishing a screen increases the intelligence " +
		"of Packlett and Heward, the two ghosts. The game is at its most difficult " +
		"on the fourth screen.\n" +
		"\n" +
		"My personal highscore is 1575 so far."

	if got := Reflow(in); got != want {
		t.Errorf("Reflow:\n%s\nwant:\n%s", got, want)
	}
}

func TestReflowKeepsLayout(t *testing.T) {
	t.Parallel()

	keep := []string{
		// Short lines are deliberate.
		"WONDER as you watch a ball bounce back and forth\n\nAMAZE at the pixel perfect collision detection",
		// A heading and its underline.
		"HOW TO PLAY AND WIN AT THIS VERY LONG GAME, WHICH TAKES SOME EXPLAINING\n===========",
		// A table.
		" HEX key   PC key*   Use\n---------------------------\n 5         W         up",
		// List items, even long ones.
		"- the first thing to know about this game is that it is really very long indeed\n- second",
		`"5","7","8","9" - move up, left, down, and right respectively, which is long enough` + "\nnext",
		// An indented continuation.
		`"6" - haunt! if there is a person on the screen then they will run away from you.` +
			"\n      you can move while the person is running.",
	}

	for _, in := range keep {
		if got := Reflow(in); got != in {
			t.Errorf("changed layout:\n%q\nbecame\n%q", in, got)
		}
	}
}

func TestReflowTrimsAndNormalises(t *testing.T) {
	t.Parallel()

	in := "This line is long enough to be wrapped prose, and it ends with a space \r\n" +
		"which the join removes."
	if got := Reflow(in); strings.Contains(got, "  ") || strings.Contains(got, "\r") {
		t.Errorf("Reflow = %q", got)
	}
}

func TestPreformatted(t *testing.T) {
	t.Parallel()

	for line, want := range map[string]bool{
		" 5         W         up": true,
		"-----":                   true,
		"      indented":          true,
		"plain prose":             false,
		"":                        false,
		"a\tb":                    true,
	} {
		if got := Preformatted(line); got != want {
			t.Errorf("Preformatted(%q) = %v, want %v", line, got, want)
		}
	}
}
