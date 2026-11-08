// Command chippie is the CHIP-8 emulator. The same code runs natively in a
// desktop window (go run ./cmd/chippie) and in the browser as WebAssembly
// (scripts/build.sh).
//
// Native flags:
//
//	-rom <sha1> | game/glitchGhost | test/2-ibm-logo | path/to/file.ch8
//	-platform originalChip8   platform for a ROM file
//	-shader clean             no CRT effect
//	-testroms DIR             test-suite ROMs (default third_party/chip8-test-suite/bin)
//	-debug                    show TPS/FPS
//
// In the browser the same options are URL query parameters, e.g.
// ?rom=test/2-ibm-logo&shader=clean.
package main

import (
	"context"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/rbscholtus/chippie/app"
	"github.com/rbscholtus/chippie/app/roms"
	"github.com/rbscholtus/chippie/catalogue"
	"github.com/rbscholtus/chippie/platform"
	"github.com/rbscholtus/chippie/romstore"
)

const (
	windowWidth  = 1024
	windowHeight = 640
)

func main() {
	params, err := platform.ReadParams(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	builtin, err := roms.NewLibrary(params.TestROMs)
	if err != nil {
		log.Fatal(err)
	}

	cat, err := catalogue.LoadSnapshot()
	if err != nil {
		log.Fatal(err)
	}

	store := romstore.New(cat.Sources.URLs, platform.NewCache(platform.StoreROMs), params.Base).
		WithLocal(builtin.LoadHash)

	game, err := app.New(app.Deps{
		Params:    params,
		Catalogue: cat,
		Builtin:   builtin,
		Store:     store,
		Prefs:     platform.NewPrefs(),
		Refresh: catalogue.Refresh(context.Background(), platform.Fetch,
			platform.NewCache(platform.StoreMetadata), catalogue.DatabaseURL),
	})
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowTitle("Chippie")
	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
