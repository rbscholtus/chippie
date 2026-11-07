# Chippie prototype (throwaway)

Proved that a pure-Go Chippie works in the browser: ebitenui library with
search, Kage CRT shader, IndexedDB from Go. Findings are in the main README §3.

Run natively: `go run .`
Run in a browser: `./build.sh`, then serve `web/` and open `local.html`.

Reusable in the real project: `crt.kage`, `phosphor.kage`, `field.go`,
`storage_js.go`, `web/page.html`. Everything else is throwaway; don't copy its
structure.
