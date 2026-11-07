#!/usr/bin/env bash
# Builds the prototype: WASM (stripped) + gzip + the host page with wasm_exec.js inlined.
set -euo pipefail
cd "$(dirname "$0")"
export GOTOOLCHAIN=go1.27.0
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o web/chippie.wasm .
gzip -9 -k -f web/chippie.wasm
python3 - "$(go env GOROOT)/lib/wasm/wasm_exec.js" <<'P'
import sys
page = open("web/page.html").read().replace("/*__WASM_EXEC__*/", open(sys.argv[1]).read())
open("web/index.html", "w").write(page)
open("web/local.html", "w").write("<!doctype html><meta charset=utf-8><meta name=viewport content='width=device-width,initial-scale=1'>" + page)
P
ls -la web
