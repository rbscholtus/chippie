#!/usr/bin/env bash
# Assembles Chippie's own games (games/<name>/<name>.8o) with Octo into
# assets/roms/<name>.ch8, which is committed and compiled into the app
# (docs/plans/M9-game.md §3). CI runs this and fails if a ROM is stale.
#
# When a ROM changes, its SHA-1 changes too: the old one is replaced with the
# new one in assets/roms/bundled.json and catalogue/overlay.json.
#
# Needs Node: the first run installs the pinned assembler into games/node_modules.
set -euo pipefail
cd "$(dirname "$0")/.."

if [ ! -x games/node_modules/.bin/octo ]; then
  (cd games && npm ci --no-audit --no-fund)
fi

for src in games/*/*.8o; do
  name=$(basename "$src" .8o)
  if [ "$(basename "$(dirname "$src")")" != "$name" ]; then
    continue # only games/<name>/<name>.8o is a game; other files are includes
  fi
  out="assets/roms/$name.ch8"
  old=""
  [ -f "$out" ] && old=$(shasum "$out" | cut -d' ' -f1)
  games/node_modules/.bin/octo "$src" "$out"
  new=$(shasum "$out" | cut -d' ' -f1)
  echo "$out: $(wc -c < "$out" | tr -d ' ') bytes, SHA-1 $new"
  if [ -n "$old" ] && [ "$old" != "$new" ]; then
    for meta in assets/roms/bundled.json catalogue/overlay.json; do
      if grep -q "$old" "$meta"; then
        sed -i.bak "s/$old/$new/g" "$meta" && rm "$meta.bak"
        echo "  updated the SHA-1 in $meta"
      fi
    done
  fi
done
