#!/usr/bin/env bash
# Builds the browser version into web/dist/:
#   index.html              the loader page, with wasm_exec.js inlined
#   chippie-gz.wasm         the app, gzip-compressed (the loader unpacks it)
#   roms/test-suite/        the Timendus test ROMs (GPLv3), shipped as
#                           separate files with their licence, never compiled in
#   roms/hosted/            the ROMs only on Zophar or David Winter's page, with attribution
#   tv/<skin>/              each TV skin's artwork (cmd/skin-gen), fetched at runtime
#
# Serve it with:  cd web/dist && python3 -m http.server 8080
set -euo pipefail
cd "$(dirname "$0")/.."

out=web/dist
suite=third_party/chip8-test-suite

if [ ! -f "$suite/bin/1-chip8-logo.ch8" ]; then
  echo "test suite missing: run git submodule update --init" >&2
  exit 1
fi

rm -rf "$out"
mkdir -p "$out/roms/test-suite" "$out/roms/hosted" "$out/tv"

GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o "$out/chippie.wasm" ./cmd/chippie
gzip -9 -n -c "$out/chippie.wasm" > "$out/chippie-gz.wasm"
rm "$out/chippie.wasm"

gz_size=$(wc -c < "$out/chippie-gz.wasm" | tr -d ' ')

python3 - "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$gz_size" "$out/index.html" <<'PY'
import sys
wasm_exec, gz_size, dst = sys.argv[1:]
page = open("web/page.html").read()
for marker in ("/*__WASM_EXEC__*/", "__WASM_GZ_SIZE__"):
    if marker not in page:
        sys.exit(f"web/page.html: marker {marker} missing")
page = page.replace("/*__WASM_EXEC__*/", open(wasm_exec).read()).replace("__WASM_GZ_SIZE__", gz_size)
open(dst, "w").write(page)
PY

cp "$suite"/bin/*.ch8 "$suite/LICENSE" "$out/roms/test-suite/"
cat > "$out/roms/test-suite/README.txt" <<'TXT'
CHIP-8 test suite by Timendus, tag v4.2, licensed under GPLv3 (see LICENSE).
Source: https://github.com/Timendus/chip8-test-suite/tree/v4.2
These files are distributed unmodified, separately from Chippie.
TXT

cp web/hosted-roms/* "$out/roms/hosted/"
for skin in assets/tv/*/; do
  name=$(basename "$skin")
  mkdir -p "$out/tv/$name"
  cp "$skin"*@[123]x.* "$skin"backdrop.jpg "$out/tv/$name/"
done

echo "built $out: chippie-gz.wasm $((gz_size / 1024)) KB"
