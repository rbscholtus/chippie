# Chippie: notes for Claude

A CHIP-8 emulator written entirely in Go with Ebitengine (own widgets), compiled
to WebAssembly, presented as a mid-1970s CRT TV.

- **Design:** [docs/DESIGN.md](docs/DESIGN.md) (cited as "DESIGN §N"). Read the
  relevant section before changing architecture, the sources manifest,
  `skin.json` or the tests. §3 lists lessons from the prototype; don't repeat
  those mistakes.
- **README.md is for users:** what Chippie is, how to play, platforms,
  sources, roadmap summary. Technical detail goes in DESIGN.md, milestones and
  ideas in [docs/ROADMAP.md](docs/ROADMAP.md), upstream work in
  [docs/upstream/](docs/upstream/README.md).
- **Coding standards:** [CONTRIBUTING.md](CONTRIBUTING.md). Follow them
  strictly, especially the package rules and the Ebitengine rules. Keep
  per-frame allocations few, but don't make code awkward for it; optimise
  only when a benchmark or profile shows the need.
- **Go 1.27**, golangci-lint v2 with all linters on except those disabled in
  `.golangci.yml`. Never add a bare `//nolint`.
- **Before finishing a change:** `golangci-lint run` (also with
  `GOOS=js GOARCH=wasm`), `go test ./...`, `go build ./cmd/chippie`, and
  `./scripts/build.sh && (cd e2e && npx playwright test)` for anything the
  browser sees.
- **No `net/http` outside `platform/`:** downloads go through
  `platform.Fetch` (it keeps ~1.7 MB gzip out of the WASM).
- **The `chip8/` package must stay dependency-free and clock-free.**
- **`syscall/js` only in `platform/`**, always with a native stub.
- **Never block `Update` or `Draw`.** Async work goes in goroutines with
  channels.
- **Test against the Timendus suite v4.2** (`third_party/chip8-test-suite`).
  Never weaken a golden frame to make a test pass. Fix the emulator, or
  explain in the PR why the golden was wrong.
- **The user prefers clarifying questions** before big or ambiguous changes.
- **Name issues and PRs in full** when writing to the user, in commits and
  in PRs: say whether it's an issue or a PR, and name the repository when
  it isn't Chippie's own, e.g. "chip-8-database PR #34", "chip-8-database
  issue #33", "Chippie PR #20". A bare "#34" is ambiguous here, because
  Chippie and the upstream repositories have overlapping numbers.
