# Chippie: notes for Claude

A CHIP-8 emulator written entirely in Go with Ebitengine + ebitenui, compiled
to WebAssembly, presented as a mid-1970s CRT TV.

- **Design:** [README.md](README.md). Read the relevant section before
  changing architecture, the sources manifest, `skin.json` or the tests.
  §3 lists lessons from the prototype; don't repeat those mistakes.
- **Coding standards:** [CONTRIBUTING.md](CONTRIBUTING.md). Follow them
  strictly, especially the package rules, the Ebitengine rules and "the
  per-frame path never allocates".
- **Go 1.27**, golangci-lint v2 with all linters on except those disabled in
  `.golangci.yml`. Never add a bare `//nolint`.
- **Before finishing a change:** `golangci-lint run`, `go test ./...`,
  `GOOS=js GOARCH=wasm go build ./cmd/chippie` and `go build ./cmd/chippie`.
- **The `chip8/` package must stay dependency-free and clock-free.**
- **`syscall/js` only in `platform/`**, always with a native stub.
- **Never block `Update` or `Draw`.** Async work goes in goroutines with
  channels.
- **Test against the Timendus suite v4.2** (`third_party/chip8-test-suite`).
  Never weaken a golden frame to make a test pass. Fix the emulator, or
  explain in the PR why the golden was wrong.
- **The user prefers clarifying questions** before big or ambiguous changes.
