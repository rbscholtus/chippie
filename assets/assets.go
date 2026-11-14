// Package assets embeds the files compiled into the app: the bundled ROMs
// and the TV's skin.json. The TV images themselves are fetched at runtime
// (README §12.2), so they don't grow the WASM. Every file's origin and
// licence is recorded in LICENSES.md.
package assets

import "embed"

// ROMs holds roms/bundled.json and the ROM files it lists. All are CC0,
// from John Earnest's chip8Archive.
//
//go:embed roms/bundled.json roms/*.ch8
var ROMs embed.FS

// Skin is tv/skin.json: where the cabinet's picture tube and controls are.
// It is small and fixes the layout from the first frame; the images it
// names arrive later.
//
//go:embed tv/skin.json
var Skin []byte
