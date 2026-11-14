// Package assets embeds the files compiled into the app: the bundled ROMs
// and each TV skin's skin.json. The TV images themselves are fetched at
// runtime (README §12.2), so they don't grow the WASM. Every file's origin
// and licence is recorded in LICENSES.md.
package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
)

// ROMs holds roms/bundled.json and the ROM files it lists. All are CC0,
// from John Earnest's chip8Archive.
//
//go:embed roms/bundled.json roms/*.ch8
var ROMs embed.FS

// skinFiles holds tv/<skin>/skin.json for every skin: small, and it fixes
// the layout from the first frame; the images it names arrive later.
//
//go:embed tv/*/skin.json
var skinFiles embed.FS

// DefaultSkin is the skin shown until the user picks another.
const DefaultSkin = "classic"

// SkinFile is one skin's folder name (also where its images are) and its
// skin.json.
type SkinFile struct {
	Dir  string
	JSON []byte
}

// Skins returns every skin, the default first, then by folder name.
func Skins() ([]SkinFile, error) {
	dirs, err := fs.Glob(skinFiles, "tv/*/skin.json")
	if err != nil {
		return nil, fmt.Errorf("list skins: %w", err)
	}

	out := make([]SkinFile, 0, len(dirs))

	for _, path := range dirs {
		data, err := skinFiles.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		out = append(out, SkinFile{Dir: path[len("tv/") : len(path)-len("/skin.json")], JSON: data})
	}

	slices.SortStableFunc(out, func(a, b SkinFile) int {
		switch {
		case a.Dir == DefaultSkin:
			return -1
		case b.Dir == DefaultSkin:
			return 1
		case a.Dir < b.Dir:
			return -1
		case a.Dir > b.Dir:
			return 1
		default:
			return 0
		}
	})

	return out, nil
}
