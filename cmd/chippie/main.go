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
//	-calibrate                outline the TV's controls (F3 toggles, F5 reloads skin.json)
//
// In the browser the same options are URL query parameters, e.g.
// ?rom=test/2-ibm-logo&shader=clean.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
	hostedROMs   = "web/hosted-roms" // from the repository root
	tvArtDir     = "assets/tv"       // from the repository root
	tvArtURL     = "tv/"             // next to the page, from scripts/build.sh
	maxArtFile   = 4 << 20           // the largest TV image is well under this
)

// artFetcher reads the TV's images: from the repository natively, from next
// to the page in the browser.
func artFetcher(base string) app.ArtFetcher {
	if base == "" {
		return func(_ context.Context, name string) ([]byte, error) {
			data, err := os.ReadFile(filepath.Join(tvArtDir, name))
			if err != nil {
				return nil, fmt.Errorf("TV art: %w", err)
			}

			return data, nil
		}
	}

	return func(ctx context.Context, name string) ([]byte, error) {
		data, err := platform.Fetch(ctx, base+tvArtURL+name, maxArtFile)
		if err != nil {
			return nil, fmt.Errorf("TV art: %w", err)
		}

		return data, nil
	}
}

func main() {
	params, err := platform.ReadParams(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	builtin, err := roms.NewLibrary(params.TestROMs)
	if err != nil {
		log.Fatal(err)
	}

	if params.Base == "" { // native: relative download URLs can't work
		builtin.SetHostedDir(hostedROMs)
	}

	cat, err := catalogue.LoadSnapshot()
	if err != nil {
		log.Fatal(err)
	}

	romCache := platform.NewCache(platform.StoreROMs)
	store := romstore.New(cat.Sources.URLs, romCache, params.Base).WithLocal(builtin.LoadHash)

	var skinFile func() ([]byte, error)
	if params.Base == "" {
		skinFile = func() ([]byte, error) {
			data, err := os.ReadFile(filepath.Join(tvArtDir, "skin.json"))
			if err != nil {
				return nil, fmt.Errorf("reload skin: %w", err)
			}

			return data, nil
		}
	}

	game, err := app.New(app.Deps{
		Params:     params,
		Catalogue:  cat,
		Builtin:    builtin,
		Store:      store,
		Prefs:      platform.NewPrefs(),
		CachedROMs: romCache.Keys,
		Art:        artFetcher(params.Base),
		SkinFile:   skinFile,
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
