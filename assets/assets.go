// Package assets embeds the files compiled into the app: the bundled ROMs
// and, from M6, the TV artwork. Every file's origin and licence is recorded
// in LICENSES.md.
package assets

import "embed"

// ROMs holds roms/bundled.json and the ROM files it lists. All are CC0,
// from John Earnest's chip8Archive.
//
//go:embed roms/bundled.json roms/*.ch8
var ROMs embed.FS
