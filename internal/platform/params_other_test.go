//go:build !(js && wasm)

package platform

import "testing"

func TestReadParams(t *testing.T) {
	t.Parallel()

	p, err := ReadParams(nil)
	if err != nil {
		t.Fatal(err)
	}

	if p != (Params{Shader: ShaderCRT, TestROMs: defaultTestROMs}) {
		t.Errorf("defaults: %+v", p)
	}

	p, err = ReadParams([]string{"-rom", "game/danm8ku", "-shader", "clean", "-platform", "originalChip8", "-debug"})
	if err != nil {
		t.Fatal(err)
	}

	want := Params{
		ROM: "game/danm8ku", Platform: "originalChip8", Shader: ShaderClean,
		TestROMs: defaultTestROMs, Debug: true,
	}
	if p != want {
		t.Errorf("got %+v, want %+v", p, want)
	}

	if _, err := ReadParams([]string{"-nope"}); err == nil {
		t.Error("unknown flag accepted")
	}
}
