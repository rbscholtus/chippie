# Catalogue snapshot

Embedded into the app (`go:embed`), so the library works offline and on first
load. Regenerate with `go run ./cmd/catalogue-gen`; the weekly refresh workflow
runs it with `-update-pins` and opens a PR when anything changed. The commits
used are in `pins.json`.

| File | From | Licence |
|---|---|---|
| `platforms.json`, `quirks.json`, `programs.json`, `sha1-hashes.json` | [chip-8-database](https://github.com/chip-8/chip-8-database) `database/` | MIT (below); program descriptions belong to their authors |
| `chip8archive-programs.json` | [chip8Archive](https://github.com/JohnEarnest/chip8Archive) `programs.json` | CC0 1.0 |
| `chip8archive-hashes.json` | generated: chip8Archive program id → ROM SHA-1 | — |
| `sources.json` | generated: where each ROM can be downloaded (README §9.3) | — |

## chip-8-database licence

The JSON files are released by the CHIP-8 database authors under the MIT
licence. The program descriptions in `programs.json` were mostly published by
the original authors; the database republishes them in good faith, credited,
with a takedown procedure. Chippie shows them on the same terms.

```
Copyright 2023 The CHIP-8 database authors

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```
