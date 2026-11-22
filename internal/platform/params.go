// Package platform holds everything that differs between the browser and
// native builds. It is the only package that imports syscall/js; every
// _js.go file has a native counterpart in an _other.go file, so native
// builds and tests work.
package platform

// Params are the start-up options: command-line flags natively, URL query
// parameters in the browser.
type Params struct {
	// ROM is a library ID ("game/glitchGhost", "test/2-ibm-logo") or,
	// natively, a path to a ROM file. Empty means show the picker.
	ROM string
	// Platform overrides the platform for a ROM file (platforms.json id).
	Platform string
	// Shader is "crt" (default) or "clean" (no CRT effect, for tests).
	Shader string
	// TestROMs is where the test-suite ROMs live: a directory natively, or
	// a URL ending in "/" in the browser.
	TestROMs string
	// Debug shows the TPS/FPS overlay from the start.
	Debug bool
	// Calibrate outlines the TV's controls and screen over the art, for
	// checking skin.json (-calibrate natively, ?skin=debug in the browser).
	Calibrate bool
	// Base is the page's base URL ("" natively): relative download URLs,
	// for ROMs Chippie hosts itself, resolve against it.
	Base string
}

// Shader modes.
const (
	ShaderCRT   = "crt"
	ShaderClean = "clean"
)
