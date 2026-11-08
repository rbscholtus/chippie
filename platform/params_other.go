//go:build !(js && wasm)

package platform

import (
	"flag"
	"fmt"
	"io"
)

// defaultTestROMs is the test suite's location in a repository checkout.
const defaultTestROMs = "third_party/chip8-test-suite/bin"

// ReadParams parses the command-line arguments (without the program name).
func ReadParams(args []string) (Params, error) {
	fs := flag.NewFlagSet("chippie", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var p Params

	fs.StringVar(&p.ROM, "rom", "", "library ID (game/…, test/…) or path to a ROM file")
	fs.StringVar(&p.Platform, "platform", "", "platform for a ROM file, e.g. originalChip8")
	fs.StringVar(&p.Shader, "shader", ShaderCRT, "crt or clean")
	fs.StringVar(&p.TestROMs, "testroms", defaultTestROMs, "directory with the test-suite ROMs")
	fs.BoolVar(&p.Debug, "debug", false, "show the TPS/FPS overlay")

	if err := fs.Parse(args); err != nil {
		return Params{}, fmt.Errorf("parse flags: %w", err)
	}

	return p, nil
}
