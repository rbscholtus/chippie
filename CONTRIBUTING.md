# Contributing to Chippie

These are the coding standards for Chippie. The design is described in
[README.md](README.md).

## Toolchain

- **Go 1.27**, latest patch release. `go.mod` declares `go 1.27`, and the
  `toolchain` line is kept on the latest 1.27.x.
- **Ebitengine v2.10+** and **ebitenui v0.7+**. Upgrades are their own PRs.
- **golangci-lint v2.** The config is in `.golangci.yml`.
- **Node LTS** only for the Playwright browser tests in `e2e/`. No JavaScript
  in the app, apart from the small loader page.

## Linting

- **All linters on, minus a short list.** The config enables everything
  (`linters.default: all`) and disables only the very noisy or opinionated
  ones, as listed in `.golangci.yml`.
- **Disabling a linter for the whole project needs a one-line reason**, as a
  comment next to it in `.golangci.yml`.
- **`//nolint` must name the linter and give a reason:**
  `//nolint:gosec // G404: CXNN needs a fast, seedable RNG, not crypto`.
  A bare `//nolint` is not allowed.
- **Lint must pass in CI** for both `GOOS=js GOARCH=wasm` and the native
  build. Run `golangci-lint run` before pushing.
- **Formatting:** `gofumpt` + `goimports` (through golangci-lint formatters).
  Kage files (`*.kage`) follow gofmt layout.

## Package rules

- **`chip8/` (the core):**
  - No imports outside the standard library.
  - No `syscall/js`, `time`, `math/rand` global state, or I/O.
  - The core never reads the clock. It advances only through `RunFrame`.
- **`catalogue/`, `romstore/`:** standard library + `platform.Fetch` for
  downloads (never `net/http` directly: it would bloat the WASM); no Ebitengine,
  no `syscall/js`.
- **`platform/`:** the only package with `syscall/js`. Every file that uses it
  is build-tagged `//go:build js && wasm` and has a `!(js && wasm)` stub, so
  native builds and tests work.
- **`app/`:** Ebitengine and ebitenui. Keeps game logic out: it calls into
  `chip8`, `catalogue` and `romstore`.
- **`internal/testrig/`:** test helpers only, never imported by production
  code.
- **`cmd/`:** entry points and developer/CI tools. Tools may use the network
  and the file system.

## Ebitengine rules

- **Never block `Update` or `Draw`.** Network, storage and file-picker work
  runs in goroutines and reports back on channels that `Update` polls without
  blocking (`select` with `default`).
- **No image or shader creation per frame.** Create `ebiten.Image`s and
  shaders at start-up or on resize, then reuse them.
- **Reuse draw options and uniforms.** Keep `DrawImageOptions`,
  `DrawRectShaderOptions` and their `Uniforms` maps as fields; update values
  in place.
- **Kage:** use `//kage:unit pixels`. Work in source coordinates
  (`srcPos - imageSrc0Origin()`), not `imageDstSize()`, which is the whole
  destination image.
- **HiDPI:** `Layout` returns real pixels (outside size × device scale
  factor). Sizes and fonts are scaled by the same factor, never hard-coded in
  CSS pixels.
- **Text input:** ebitenui `TextInput` by default. The `exp/textinput` field
  in `app/ui` is the fallback for IME-heavy input.

## Memory and performance

- **Keep allocations in the per-frame path few, without making the code
  awkward.** The per-frame path is `RunFrame`, every instruction, display
  operations, trace recording, the framebuffer → texture copy and shader
  uniform updates. Prefer the plain, readable version. Optimise later, when a
  benchmark or profile shows the need (README §16).
- **Size buffers for the platform when it's set up.** `New`/`Reset` allocate
  memory and display buffers for the platform in use (4 KB for CHIP-8), and
  reuse them when the size doesn't change. Don't size everything for the
  largest platform.
- **Sensible defaults in hot code:** create images, shaders, draw options and
  uniform maps once and reuse them; avoid closures and interface dispatch per
  instruction.
- **`sync.Pool` is not used for now.** Add one only when a profile shows
  repeated allocations that can't be preallocated, and include benchmark
  results in the PR.
- **Avoid `fmt`, `reflect` and growing strings in hot code.** Disassembly,
  trace formatting and OSD text are built only when they change.
- **PRs that touch `chip8/` or `app/crt/`** include `benchstat` output
  against `main` if they could affect speed.

## Code style

- **Names:** follow Effective Go and the Go Code Review Comments. Opcode
  handlers are named after the instruction, e.g. `opDXYN`, `op8XY6`.
- **Errors:** wrap with `%w` and context. Sentinel errors for conditions
  callers check (`ErrROMTooLarge`, `ErrUnknownOpcode`, `ErrHashMismatch`).
  Never `panic` on bad ROM input or bad network data.
- **Quirks are data:** instruction code checks `m.quirks.X`, never "if
  platform == ...", except to decide whether an opcode exists at all.
- **Comments:** every exported identifier has a doc comment. Each opcode
  handler cites its behaviour per platform and links the reference when the
  behaviour is unusual.
- **Constants:** no unexplained numbers. Memory addresses, font locations and
  sizes are named. Layout positions for the TV come from `skin.json`, not
  code.

## Tests

- **Every opcode has table-driven tests** covering each quirk that affects it.
- **Screenshot tests** use the Timendus suite (submodule pinned to `v4.2`) and
  the golden frames in `testdata/golden/`.
  - `go test ./... -update` regenerates golden frames that have no reference
    picture.
  - Regenerated goldens must be checked by eye in the PR (the ASCII `.txt`
    diff is shown in the review).
- **Deterministic:** fixed RNG seed, frame counts instead of time, no network
  in `go test` (use `httptest`; real-network tools under `cmd/` use an
  explicit build tag).
- **Browser tests** (`e2e/`, Playwright): allow for software rendering in CI
  (README §15.3).
- **Coverage:** `chip8/` aims for 90%+ line coverage. Coverage is reported
  in CI, not enforced.
- **Before pushing:** `go test -race ./...` (native), the WASM build, and the
  native build all pass.

## Assets

- **Every asset has a recorded origin and licence** in `assets/LICENSES.md`:
  generator, model and prompt for art, licence for fonts, source for hosted
  ROMs.
- **Fonts:** OFL or similarly free licences only.
- **Art sizes:** keep within the budgets in README §12.2. Pre-scaled `@1x`,
  `@2x`, `@3x` files are generated by the build script from one master.

## Git

- **Branches:** short-lived branches off `main`. PRs need green CI.
- **Commit messages:** Conventional Commits, e.g. `feat(chip8): add SCHIP
  scrolling`, `fix(catalogue): ...`, `test: ...`, `perf: ...`.
- **Submodule bumps** (test suite, database snapshot) are their own commits.
- **ROM files:** never commit any, except `web/hosted-roms/` (the 12
  Zophar-only ROMs, with attribution) and the test suite submodule.
