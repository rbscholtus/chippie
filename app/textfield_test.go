package app

import "testing"

func TestTextFieldEditing(t *testing.T) {
	t.Parallel()

	f := newTextField("search")
	f.insert("héllo")

	if f.text != "héllo" || f.caret != len("héllo") || !f.changed {
		t.Fatalf("insert: %q caret %d changed %v", f.text, f.caret, f.changed)
	}

	f.changed = false
	f.moveCaret(-1)
	f.moveCaret(-1)
	f.moveCaret(-1)
	f.moveCaret(-1) // before "é"
	f.backspace()   // deletes "h"

	if f.text != "éllo" || f.caret != 0 || !f.changed {
		t.Errorf("backspace: %q caret %d", f.text, f.caret)
	}

	f.moveCaret(1) // over the two-byte "é"

	if f.caret != len("é") {
		t.Errorf("caret %d, want %d", f.caret, len("é"))
	}

	f.backspace()
	f.backspace() // nothing left before the caret

	if f.text != "llo" || f.caret != 0 {
		t.Errorf("after backspaces: %q caret %d", f.text, f.caret)
	}

	f.moveCaret(-1) // already at the start
	f.setText("abc", 99)

	if f.caret != 3 {
		t.Errorf("caret clamped to %d, want 3", f.caret)
	}
}

func TestTextFieldValueIncludesComposition(t *testing.T) {
	t.Parallel()

	f := newTextField("search")
	f.insert("ab")
	f.moveCaret(-1)
	f.composition = "XY"

	if got := f.value(); got != "aXYb" {
		t.Errorf("value %q, want aXYb", got)
	}

	f.focus()

	if !f.focused {
		t.Error("focus")
	}

	f.blur()

	if f.focused {
		t.Error("blur")
	}
}
