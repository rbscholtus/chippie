#!/usr/bin/env bash
# Assembles Chippie's own games (games/<name>/<name>.8o) with Octo into
# assets/roms/<name>.ch8, which is committed and compiled into the app
# (docs/plans/M9-game.md §3). CI runs this and fails if a ROM is stale.
#
# When a ROM changes, its SHA-1 changes too: whatever SHA-1 the metadata
# (assets/roms/bundled.json and catalogue/overlay.json) holds for the ROM's
# file is replaced with the new one. It's read from the metadata rather than
# the old ROM, so an interrupted run can't leave them out of step.
#
# Needs Node: the first run installs the pinned assembler into games/node_modules.
set -euo pipefail
cd "$(dirname "$0")/.."

if [ ! -x games/node_modules/.bin/octo ]; then
  (cd games && npm ci --no-audit --no-fund)
fi

# rom_hashes prints the SHA-1s that metadata file $1 holds for ROM file $2:
# bundled.json entries' "sha1", and overlay.json "roms" keys.
rom_hashes() {
  node -e '
    const [meta, file] = process.argv.slice(1);
    const found = new Set();
    (function walk(v) {
      if (Array.isArray(v)) return v.forEach(walk);
      if (!v || typeof v !== "object") return;
      if (v.file === file && v.sha1) found.add(v.sha1);
      if (v.roms) for (const [sha, r] of Object.entries(v.roms)) if (r.file === file) found.add(sha);
      Object.values(v).forEach(walk);
    })(require(require("path").resolve(meta)));
    console.log([...found].join("\n"));
  ' "$1" "$2"
}

for src in games/*/*.8o; do
  name=$(basename "$src" .8o)
  if [ "$(basename "$(dirname "$src")")" != "$name" ]; then
    continue # only games/<name>/<name>.8o is a game; other files are includes
  fi
  out="assets/roms/$name.ch8"
  games/node_modules/.bin/octo "$src" "$out"
  new=$(shasum "$out" | cut -d' ' -f1)
  echo "$out: $(wc -c < "$out" | tr -d ' ') bytes, SHA-1 $new"
  for meta in assets/roms/bundled.json catalogue/overlay.json; do
    for old in $(rom_hashes "$meta" "$name.ch8"); do
      if [ "$old" != "$new" ]; then
        sed -i.bak "s/$old/$new/g" "$meta" && rm "$meta.bak"
        echo "  updated the SHA-1 in $meta"
      fi
    done
  done
done
