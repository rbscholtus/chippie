# Chippie: design

How Chippie is built, and why. For what Chippie is and how to play it, see
the [README](../README.md); for coding standards,
[CONTRIBUTING.md](../CONTRIBUTING.md); for what's done and what's next,
[ROADMAP.md](ROADMAP.md).

Section numbers are stable: code comments cite them as "DESIGN §N".

---

## Contents

1. [Goals](#1-goals)
2. [Key decisions](#2-key-decisions)
3. [Prototype findings](#3-prototype-findings)
4. [Architecture](#4-architecture)
5. [Repository layout](#5-repository-layout)
6. [Emulator core](#6-emulator-core)
7. [Platforms, quirks and profiles](#7-platforms-quirks-and-profiles)
8. [Timing](#8-timing)
9. [Game library](#9-game-library)
10. [Metadata](#10-metadata)
11. [Presentation: the 1970s TV](#11-presentation-the-1970s-tv)
12. [TV artwork](#12-tv-artwork)
13. [Input and audio](#13-input-and-audio)
14. [Browser integration](#14-browser-integration)
15. [Testing](#15-testing)
16. [Performance](#16-performance)
17. [Build, CI and deployment](#17-build-ci-and-deployment)

Upstream contributions are tracked in [upstream/](upstream/README.md).

---

## 1. Goals

- **Correct emulation of several CHIP-8 platforms**, each with its own quirks,
  selected automatically per game and overridable by the user.
- **A large built-in game library.** Games are browsed, downloaded and played
  from public archives, with no local ROM file needed.
- **Browser first.** Nothing to install. Hosted on GitHub Pages. Works on
  phones and tablets as well as desktops.
- **Retro presentation.** The emulator is a mid-1970s CRT TV, and its knobs
  and buttons are the controls.
- **Verifiable.** Every platform is tested against the Timendus test suite
  (v4.2) with pixel-exact screenshot comparisons.
- **Instruction tracing.** An optional log of executed instructions,
  disassembled for the active platform.
- **One language.** Everything is Go, including the UI and the shaders
  (Kage).

Out of scope for now: save states, a step-debugger UI. A desktop build comes
almost for free with Ebitengine and is useful during development, but it isn't
a product goal.

## 2. Key decisions

| Topic | Decision |
|---|---|
| Language | Go 1.27 (latest patch), everything including UI. Shaders in Kage. |
| Engine | Ebitengine v2.10+ (game loop, rendering, input, audio, text input) |
| UI widgets | Our own, drawn with Ebitengine: one touch model (drag to scroll, tap on release, 44 px rows) for the library, settings, knobs and remote. ebitenui was used in the prototype; dropped in M4 (smaller WASM, touch scrolling) |
| Target | Browser (`GOOS=js GOARCH=wasm`). Native desktop builds for development. |
| TV look | Mid-1970s TVs from AI-generated artwork (Gemini), plus a CRT shader in Kage. Three skins to choose from (Walnut console, Portable, Slim walnut); the controls are cut from each cabinet image itself. |
| Metadata authority | [chip-8-database](https://github.com/chip-8/chip-8-database), plus Chippie's overlay (`catalogue/overlay.json`) of changes proposed upstream but not yet merged, each linked to its PR. The goal is an empty overlay; no long-lived fork. |
| ROM files | Downloaded from their original source at runtime. Only the 18 ROMs that can't be fetched from a browser are hosted by us. |
| Profiles | Platform defaults, then the database's per-ROM overrides, then user settings |
| SUPER-CHIP | Modern behaviour by default (what the database's quirks describe); the HP48's legacy behaviour is a per-ROM setting (`LegacySuperChip`) |
| Art pipeline | Generated masters stay outside the repository (`assets/source/`); `cmd/skin-gen` turns them into the committed `assets/tv/<skin>/` files |
| Upstream | Findings go to chip-8-database and the test suite as issues and PRs from the owner's fork, each drafted in `docs/upstream/` and approved by the owner before it's posted |
| Tests | Golden framebuffers made from the test suite's own reference pictures |
| Memory | Few allocations in the per-frame path, without awkward code; optimise when profiles show the need. No `sync.Pool` for now. |

**Why pure Go?** Three earlier options were considered: Go core + Svelte UI,
Go core + TypeScript loop, and all TypeScript. Pure Go with Ebitengine was
chosen once the prototype showed the three things in doubt all work in the
browser: a searchable ebitenui library, a Kage CRT shader, and IndexedDB from
Go. It also keeps the whole project in one language with one toolchain, and
`go run` gives a native window for fast iteration.

## 3. Prototype findings

A throwaway prototype (in `prototype/`: Ebitengine 2.10.6, ebitenui 0.7.3,
Go 1.27.0) was built and tested in headless Chromium, desktop Chrome/Safari on
a Mac, and Safari on an iPhone.

| Question | Result |
|---|---|
| Searchable library in ebitenui | ✅ 484 titles, filtering on every keystroke, selection works. Landscape and portrait layouts. |
| On-screen keyboard on iPhone | ✅ Appears for both ebitenui's `TextInput` and a field built on Ebitengine's `exp/textinput`. First letter is kept on real devices. |
| CRT effect in Kage | ✅ Curvature, scanlines, shadow mask, bloom, phosphor afterglow, power-off collapse, colour tint |
| IndexedDB from Go | ✅ Save and restore across reloads, through `syscall/js` wrapped in goroutines and channels |
| Timing | ✅ TPS 60 on Mac (120 Hz ProMotion) and iPhone. Emulation speed is independent of refresh rate. |
| Rendering speed | ⚠️ Mac: 60 FPS. iPhone: 50+ FPS. The CRT shader at 3× pixel density is the likely cost (see §16). |
| Download size | ⚠️ 21.9 MB raw, **5.1 MB gzip, 3.6 MB brotli**. Downloaded quickly on iPhone and Mac; judged acceptable. |
| Load time | Download + unpack ~0.13 s, compile ~0.06 s on a fast connection |

**Size breakdown** (measured on a native build as a stand-in): Go runtime
~4.9 MB, Ebitengine ~1.4 MB, text shaping (go-text/typesetting) ~1.2 MB,
ebitenui only ~0.2 MB. A bare Go WASM program is about 1.7 MB.

**Lessons that carry into the real code:**

- **Kage coordinates:** with `//kage:unit pixels`, `imageDstSize()` is the
  whole destination image, not the area being drawn. Work in source
  coordinates (`srcPos - imageSrc0Origin()`, `imageSrc0Size()`).
- **ebitenui panels** grow to fit their widest child unless the layout bounds
  them. Bound panels with anchor padding and stretching.
- **Lists need a height limit** in portrait layouts, or they grow to their
  full length.
- **The loader page** must remove its overlay completely (`[hidden]` with
  `display: none`), or it swallows clicks meant for the canvas.
- **The library list in the prototype stays open after a pick** because it
  was a permanent panel. The real library is a drawer that closes on
  selection (§11).
- **Headless Chromium uses software rendering** (~5 FPS) and drops the first
  key typed into `exp/textinput`. Neither happens on real devices. Browser
  tests must allow for this (§15.3).

Reusable pieces from the prototype: `crt.kage`, `phosphor.kage`, `field.go`
(the `exp/textinput` search field), `storage_js.go` (IndexedDB) and the loader
page in `web/page.html`.

## 4. Architecture

```
┌─────────────────────────────── Browser ───────────────────────────────┐
│  web/index.html   loader: download + unpack + start chippie.wasm      │
│                                                                       │
│  ┌─────────────────────────── chippie.wasm ────────────────────────┐  │
│  │                                                                 │  │
│  │  app/        Ebitengine Game: Update (60 TPS) / Draw (per frame)│  │
│  │   ├─ tv/       TV skin, knobs, buttons, power, OSD              │  │
│  │   ├─ crt/      Kage shaders: phosphor, CRT                      │  │
│  │   ├─ library/  drawer model: search, groups, details            │  │
│  │   ├─ (app)     drawer, settings panel, search field, layout     │  │
│  │   ├─ remote/   on-screen hex keypad for touch devices           │  │
│  │   └─ sound/    buzzer + XO-CHIP pattern audio                   │  │
│  │                                                                 │  │
│  │  catalogue/  database, sources manifest, profile resolution     │  │
│  │  romstore/   download (platform.Fetch) + SHA-1 check + cache    │  │
│  │  platform/   browser-only bits: IndexedDB, file picker,         │  │
│  │              localStorage (stubs on desktop)                    │  │
│  │  chip8/      CPU, memory, display, timers, quirks, trace        │  │
│  │              (no dependencies, no clock, no I/O)                │  │
│  └─────────────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────────────┘
```

**Dependency direction:** `app` → `catalogue`, `romstore`, `platform`,
`chip8`. `chip8` depends on nothing. Only `platform` imports `syscall/js`.

**The frame:**

1. Ebitengine calls `Update` 60 times per second (fixed TPS).
2. `Update` reads input, updates UI state, and calls `machine.RunFrame()` once
   when the TV is on and no menu has paused it.
3. Ebitengine calls `Draw` once per display refresh (60, 120, ... Hz).
4. `Draw` copies the CHIP-8 display into a preallocated texture, runs the
   phosphor and CRT shaders into the TV's screen area, then draws the TV skin
   and any open UI on top.

## 5. Repository layout

```
chippie/
├── go.mod                      # go 1.27, toolchain go1.27.x
├── cmd/
│   ├── chippie/                # main: ebiten.RunGame (browser + desktop)
│   ├── catalogue-gen/          # CI: hash archives → sources.json + coverage report
│   ├── golden-import/          # test-suite pictures → golden frames
│   └── skin-gen/               # TV master art + recipe → assets/tv/<skin>/ (§12.1)
├── chip8/                      # emulator core: stdlib only, no clock, no I/O
│   ├── machine.go  cpu.go  ops_base.go  ops_schip.go  ops_xochip.go  ops_hires.go
│   ├── display.go  quirks.go  platform.go  disasm.go  trace.go  fonts.go  rng.go
│   ├── testdata/               # disassembly goldens
│   └── *_test.go   suite_test.go   perf_test.go
├── catalogue/                  # metadata + sources + profile resolution
│   ├── db.go  sources.go  profile.go
│   ├── snapshot/               # embedded chip-8-database JSON (go:embed)
│   └── *_test.go
├── romstore/                   # fetch, verify, cache ROMs
├── platform/                   # syscall/js lives only here
│   ├── params_js.go  params_other.go   # URL query / command-line flags
│   ├── fetch_js.go  fetch_other.go     # fetch() / net/http
│   ├── storage_js.go  storage_other.go
│   ├── filepicker_js.go  filepicker_other.go
│   ├── prefs_js.go  prefs_other.go
│   └── layout_js.go  layout_other.go   # layout for the browser tests; touch screen?
├── app/
│   ├── game.go  game_flow.go   # ebiten.Game; states; layout
│   ├── tv.go  controls.go      # the TV: layout, art loading, drawing; what the controls do
│   ├── osd.go  remote.go       # on-screen display; touch remote (hex keypad)
│   ├── ui.go  picker.go  settings.go  textfield.go   # drawer, settings panel, search field
│   ├── skin/                   # skin.json: parsing, checks, hit testing, turning maths
│   ├── crt/                    # *.kage + Go wrappers
│   ├── input/                  # keyboard → hex keypad
│   ├── roms/                   # bundled games + test suite
│   ├── library/                # the library's entries
│   └── sound/                  # buzzer and XO-CHIP audio, volume
├── assets/
│   ├── assets.go               # go:embed of the bundled ROMs and every skin.json
│   ├── roms/                   # bundled CC0 games + bundled.json (M3)
│   ├── source/                 # generated master images (git-ignored)
│   ├── tv/<skin>/              # one folder per TV skin, made by cmd/skin-gen (see §12)
│   │   ├── recipe.json             # name and measurements in the master (hand-edited)
│   │   ├── skin.json               # where everything is, at @3x (generated)
│   │   ├── cabinet@{1,2,3}x.jpg  cabinet-mask@{1,2,3}x.png  backdrop.jpg
│   │   └── channel  speed  volume  tint  power @{1,2,3}x.png
│   └── LICENSES.md             # origin and licence of every asset
├── internal/testrig/           # headless runner, golden I/O, image diffs
├── testdata/
│   ├── golden/                 # golden frames (.png native res + .txt)
│   └── failures/               # written on failure (git-ignored)
├── third_party/
│   └── chip8-test-suite/       # git submodule pinned to tag v4.2
├── web/
│   ├── page.html               # loader page (from the prototype); build.sh → web/dist/
│   ├── hosted-roms/            # 18 ROMs only on Zophar / David Winter's page (+ attribution)
│   └── (sources.json is embedded: catalogue/snapshot/)
├── e2e/                        # Playwright tests (+ fixtures/: CC0 ROM for download tests)
├── scripts/
│   └── build.sh                # wasm + compression + page assembly
├── docs/
│   ├── DESIGN.md               # this document
│   ├── ROADMAP.md              # milestones, open questions, ideas
│   ├── crt-shader.md  tv-artwork.md   # the CRT effects; the TV art prompts
│   ├── plans/                  # milestone plans
│   ├── upstream/               # proposals to chip-8-database and the test suite
│   ├── images/                 # screenshots for the README
│   └── coverage.md             # ROM coverage report (catalogue-gen)
├── .github/workflows/
├── .golangci.yml               # lint config
├── CONTRIBUTING.md
└── CLAUDE.md
```

## 6. Emulator core

### 6.1 Principles

- **No dependencies, no clock, no randomness the caller can't control.** The
  core advances only when told to (`RunFrame`), which makes every run
  repeatable. The screenshot tests depend on that.
- **State sized per platform.** Memory and display planes are allocated by
  `New`/`Reset` for the platform in use (4 KB of memory for CHIP-8, 64 KB for
  XO-CHIP) and reused when the size doesn't change. Registers, stack and
  keypad are small arrays in the `Machine` struct.
- **Quirks are data.** Instruction code checks `m.quirks.Shift` and so on. It
  never checks "which platform am I", except for instructions that only exist
  on some platforms.

### 6.2 Public API (sketch)

```go
package chip8

type Platform uint8 // OriginalChip8, HybridVIP, ModernChip8, Chip48, SuperChip1, SuperChip, XOChip, ...

type Quirks struct {
    Shift                 bool // 8XY6/8XYE shift vX in place (ignore vY)
    MemoryIncrementByX    bool // FX55/FX65 increment I by X
    MemoryLeaveIUnchanged bool // FX55/FX65 leave I unchanged
    Wrap                  bool // sprites wrap at the screen edge instead of clipping
    Jump                  bool // BNNN behaves as BXNN (jump to XNN + vX)
    VBlank                bool // DXYN waits for vertical blank
    Logic                 bool // 8XY1/2/3 reset vF to 0
    LegacySuperChip       bool // the HP48's real SUPER-CHIP behaviour (§7); not in quirks.json
}

type Profile struct {
    Platform      Platform
    Quirks        Quirks
    TicksPerFrame int
    Seed          uint64 // for CXNN; tests use a fixed seed
}

type Machine struct { /* memory and display sized per platform */ }

func New(p Profile) *Machine
func (m *Machine) Reset(p Profile)
func (m *Machine) LoadROM(rom []byte) error   // copies into memory at 0x200
func (m *Machine) Poke(addr uint16, v byte)   // used by tests (e.g. 0x1FF)
func (m *Machine) SetKey(key uint8, down bool)
func (m *Machine) RunFrame()                  // timers tick once, then up to TicksPerFrame instructions
func (m *Machine) Display() DisplayView       // read-only view: width, height, planes
func (m *Machine) Sound() SoundState          // buzzer on/off, XO-CHIP pattern + pitch
func (m *Machine) State() CPUState            // PC, I, V, SP, timers (for tracing/UI)
func (m *Machine) Trace() *TraceBuffer        // nil unless enabled
func (m *Machine) EnableTrace(n int)          // allocates once, here
```

### 6.3 Platforms

Each platform from the database's `platforms.json` gets its own `Platform`
value. Defaults come straight from that file:

| Platform id | Resolutions | Default ticks/frame | Notes |
|---|---|---|---|
| `originalChip8` | 64×32 | 15 | COSMAC VIP. `vblank` and `logic` quirks on. |
| `hybridVIP` | 64×32 | 15 | VIP plus machine-code routines (later) |
| `modernChip8` | 64×32 | 12 | Behaviour most modern ROMs expect |
| `chip8x` | 64×32 | 15 | Colour extension. Not planned: no ROMs in the database ([roadmap](ROADMAP.md#not-planned)) |
| `chip48` | 64×32 | 30 | HP48 |
| `superchip1` | 64×32, 128×64 | 30 | SCHIP 1.0 |
| `superchip` | 64×32, 128×64 | 30 | SCHIP 1.1. Modern behaviour unless the per-ROM legacy setting is on (§7) |
| `megachip8` | up to 256×192 | 1000 | Not planned: no ROMs in the database ([roadmap](ROADMAP.md#not-planned)) |
| `xochip` | 64×32, 128×64 | 100 | 2 planes, 64 KB memory, audio patterns |
| `hiresChip8` (overlay) | 64×32, 64×64 | 15 | COSMAC VIP two-page 64×64 mode, VIP quirks. Not in the database: Chippie's overlay adds it (§10) |

**Hires CHIP-8** (M10) is VIP CHIP-8 plus two rules. A hires ROM starts at
0x200 with `1260`, a jump over a machine-code patch that switches the VIP
interpreter to 64×64. The core doesn't run the patch: when `1260` is the
instruction at 0x200, it switches to 64×64 and continues at 0x2C0, where the
CHIP-8 program starts. `0230` (the patch's clear routine) clears the screen.
Without `1260` a ROM runs exactly as on `originalChip8`, so the Timendus
suite runs on it with the same goldens. The picture stays 2:1: a 64×64
frame is stretched to fill it, with tall pixels.

**Order of implementation:** `modernChip8` → `originalChip8` → `chip48` /
`superchip1` / `superchip` → `xochip`. That covers about 98% of the ROMs in the
database (291 VIP, 105 SCHIP 1.1, 97 XO-CHIP, 42 modern).

The display is one byte per pixel at the active resolution, with the XO-CHIP
planes as bits of that byte. The buffer is sized for the platform's largest
resolution. Legacy SUPER-CHIP lo-res keeps the full 128×64 screen with 2×2
pixels, so scrolling can move half a lo-res pixel.

**Where platform behaviour comes from.** `quirks.json` and the Timendus
suite cover the quirks, but not which instructions each platform has or how
its sound behaves. For those the reference is
[CHIP-8 extensions and compatibility](https://chip-8.github.io/extensions/),
checked against Chippie on 2026-11-16. Nothing else checks what is taken
from it, so each such behaviour gets a unit test that pins it. From that
review:

- **SUPER-CHIP 1.0 font:** 1.0 has no `FX30`; its `FX29` points I at the
  big font when VX is `10`–`19`. SUPER-CHIP 1.1 added `FX30` and made `FX29`
  small-font only. Chippie does 1.0's `FX29` (`opsFontFX29`) and also keeps
  `FX30` on 1.0, a harmless extra for ROMs labelled 1.0 but written for 1.1.
- **COSMAC VIP sound timer:** the VIP doesn't respond to a value of 1; the
  minimum is 2 (Mikolay's
  [technical reference](https://github.com/mattmikolay/chip-8/wiki/CHIP%E2%80%908-Technical-Reference),
  citing the VIP manual). On the VIP platforms the buzzer sounds while the
  timer is at least 2, so `FX18` with 1 is silent (§13).
- **XO-CHIP pattern position** restarts only when the sound timer reaches 0
  (§13).
- **Hires load address:** the page says programs load at `0260`. Both that
  and the `0x2C0` entry above are right: `0x260`–`0x2BF` is CHIP-8 code that
  patches the interpreter's memory, and Chippie applies its effect instead.
- **Deliberate differences:** a 16-entry stack everywhere (the VIP has 12,
  in memory at `0xEA0`); a big font with all 16 glyphs (SUPER-CHIP has
  0–9; Octo has 0–F); the whole of `0x200`–`0xFFF` for programs, although
  the VIP keeps `0xEA0`–`0xFFF` for its stack, work area and screen.
- **Checked and matching:** VIP coordinate wrap with clipping and the
  display wait; CHIP-48's `BXNN`, `FX55`/`FX65` I+X (none for X = 0) and
  VX-only shifts; SUPER-CHIP 1.1's unchanged I, legacy row-counting `VF`,
  half-pixel lo-res scrolling and no clear on a mode switch; XO-CHIP's
  four-byte skip over `F000 NNNN`, clearing on a resolution change and the
  pitch formula.
- **Ideas not taken up:** see [the roadmap](ROADMAP.md) (a CDP1802 core for
  the VIP; Octo's `0000` halt).

### 6.4 Instruction trace

- **A fixed-size ring buffer** of `{PC uint16, Op uint16, I uint16, Frame uint32}`
  entries, allocated once when tracing is switched on.
- **Recording is cheap:** while running, the core stores raw values only.
- **Disassembly happens only when someone views the log**, using the active
  platform's opcode table. The same opcode can mean different things on
  different platforms (e.g. `00FE` on SCHIP).
- **Where to view it:** the settings panel (last instructions, and "Copy trace" to the clipboard).
- **Debugging aid:** a "dump trace on unknown opcode" option.

## 7. Platforms, quirks and profiles

The emulator works out a profile when a ROM is loaded:

```
1. Platform defaults        platforms.json (+ the overlay's) → quirks + default tickrate
2. ROM-specific overrides   programs.json → roms[sha1].platforms[0]
                                          → roms[sha1].quirkyPlatforms[platform]
                                          → roms[sha1].tickrate, colors, keys
3. User overrides           saved per ROM hash (platform/prefs: localStorage)
```

- **chip8Archive quirks:** SUPER-CHIP and XO-CHIP games from chip8Archive were
  written in Octo and run with the quirks recorded there (§10). A switch that
  isn't listed means Octo's default, off. chip-8-database wins where it sets a
  value of its own.
- **Legacy SUPER-CHIP** is a per-ROM setting, off by default. chip-8-database
  has one `superchip` entry, and its quirks are the modern behaviour almost all
  ROMs expect. The setting adds what the HP48 really did: lo-res display wait,
  2×2 lo-res pixels with half-pixel scrolling, 8×16 lo-res `DXY0`, row-counting
  `VF` in hi-res, and no clear on a resolution switch.
- **Unknown ROMs** (hash not in the database) start with `modernChip8`
  defaults. The settings panel lets the user pick a platform and toggle quirks.
- **"Reset to recommended"** drops the user's overrides for that ROM.
- **Where the logic lives:** `catalogue.Resolve(hash, userOverrides) Profile`,
  unit-tested against the real database snapshot.

## 8. Timing

The original COSMAC VIP ran at **60 frames per second**. Its RCA 1861 "Pixie"
video chip produced an NTSC picture: a 1.76 MHz CPU clock, 3,668 machine
cycles per frame. The interrupt at the start of each frame counted down the
delay and sound timers. The same interrupt causes the `vblank` quirk: sprite
drawing waited for the next frame. The VIP had no fixed "instructions per
frame", because each CHIP-8 instruction took a different number of real
cycles. The tick rate is the usual approximation, and the database's tickrates
are tuned so games feel right.

**In Chippie:**

- **Fixed 60 Hz updates:** `ebiten.SetTPS(60)` (the default). `Update` runs
  60 times per second regardless of the display's refresh rate. The prototype
  confirmed TPS 60 on a 120 Hz Mac and on an iPhone.
- **One emulated frame per update:** each `Update` calls `RunFrame()` once.
  Timers tick once, then up to `TicksPerFrame` instructions run.
- **Speed knob = ticks per frame.** It starts at the profile's tickrate.
- **The `vblank` quirk is handled inside the core.** On VIP platforms, `DXYN`
  ends the current frame's instruction loop.
- **Display sync:** drawing happens at the screen's refresh rate. On 60 Hz
  and 120 Hz displays motion is perfectly even. On 144/165 Hz displays each
  frame is shown for 2 or 3 refreshes; the phosphor afterglow hides this. If
  it's ever noticeable, `ebiten.SetTPS(ebiten.SyncWithFPS)` can lock updates
  to the refresh when it's a multiple of ~60 Hz.
- **Pausing:** the TV pauses when the tab is hidden, when it's switched off,
  and while the library or settings are open.

## 9. Game library

### 9.1 Sources and coverage (measured 2026-11-08)

Every ROM in each source was hashed (SHA-1) and looked up in chip-8-database:

| Source | Unique ROMs | In database | Missing | Browser can download? |
|---|---|---|---|---|
| Timendus test suite | 8 | 8 | 0 | Yes (GitHub raw, CORS `*`) |
| [chip8Archive](https://github.com/JohnEarnest/chip8Archive) (CC0) | 104 | 100 | 4 | Yes (GitHub Pages/raw, CORS `*`) |
| [kripod/chip8-roms](https://github.com/kripod/chip8-roms) | 106 | 96 | 10 | Yes (GitHub raw, CORS `*`) |
| [Zophar](https://www.zophar.net/pdroms/chip8.html) CHIP-8 + SCHIP packs | 34 | 33 | 1 | **No.** Zip-packed, redirect, no CORS. |
| c8games (local collection) | 23 | 22 | 1 | n/a, a subset of the Zophar CHIP-8 pack |
| [David Winter's page](https://www.pong-story.com/chip8/) (`chp8_220.zip`, `loopz.zip`) | 40 | 40 | 0 | **No.** Zip-packed, no CORS. 6 are new; the rest match kripod. |
| [mattmikolay/chip-8](https://github.com/mattmikolay/chip-8) (MIT) | 6 | 4 | 2 | Yes (GitHub raw, CORS `*`) |
| Timendus [3d-viper-maze](https://github.com/Timendus/3d-viper-maze), [3d-vipr-maze](https://github.com/Timendus/3d-vipr-maze) (GPL-3.0) | 3 | 3 | 0 | Yes (GitHub raw, CORS `*`) |
| Octojam 6–10 games from itch.io, via [maxisimonazzi/chip8-emulator](https://github.com/maxisimonazzi/chip8-emulator) (no licence) | 68 | 42 | 26 | Yes (GitHub raw, CORS `*`) |
| **All combined** | **~310** | **268** | **42** | |

Added 2026-11-14: David Winter's page, the Octojam games and the three GitHub repositories, found
by hashing candidate sources against the database. `docs/coverage.md` has the
current numbers.

**Findings:**

- **The database is larger than the archives.** It lists 543 ROMs (484
  programs), and about 330 of them are in none of the sources above. Finding
  files, not metadata, is the main limit on library size.
- **12 Super-CHIP ROMs exist only on Zophar.** They are ALIEN, ANT, BLINKY,
  CAR, FIELD, JOUST, PIPER, RACE, SPACEFIG, UBOAT and WORM3 from the SCHIP
  pack, plus KALEID from the CHIP-8 pack. The versions in kripod's repo are
  different files. All 12 are in the database, and **we host these ourselves**
  in `web/hosted-roms/`, with Zophar's attribution and `.DOC` files.
- **6 more exist only on David Winter's page:** MINES, SQUARE, MAZE (his own
  freeware), DRAGON1/DRAGON2 (Single Dragon) and LOOPZ. We host these too.
- **Octojam games on itch.io (2019–2023)** are only playable there as Octo
  HTML exports, with no CORS. A third-party repository
  ([maxisimonazzi/chip8-emulator](https://github.com/maxisimonazzi/chip8-emulator))
  has extracted them. We link to its copies at a pinned commit, as with
  kripod (no licence; each game belongs to its author), which adds 40
  playable ROMs. Its other 26 aren't in chip-8-database yet: candidates for
  M8. chip8Archive (also shown at beyondloom.com/octo) holds the earlier
  jams.
- **The 14 ROMs missing from the database** are `dinorun`, `mato8`, `mrworm`
  and `tick` (chip8Archive), `Guess (alt)` (same file as Zophar/c8games
  `GUESS`), `Rush Hour (alt)`, and 8 "hires" ROMs from kripod. Most of the
  hires ones are probably the 64×64 "Hires CHIP-8" variant, which the
  database has no platform for yet. (M10 confirmed all 8 are, and added
  them and the platform through the overlay.)

### 9.2 Approach: one shared metadata database, URLs of our own, no copies

| Option | Pros | Cons |
|---|---|---|
| A. Copy all ROMs into chip-8-database | One place for everything | Not what that database is for (it holds hashes). Licensing unclear. Depends on the maintainer. |
| B. Our own ROM + metadata database | Full control | Duplicates the database and the archives |
| **C. Sources manifest + live download (chosen)** | No copies, extensible, always up to date | Needs CORS. Sources can move. Mitigated by SHA-1 checks and caching. |

### 9.3 Sources manifest

`catalogue/snapshot/sources.json` is generated by `cmd/catalogue-gen` and
embedded in the app (no extra download at start-up). It maps each ROM hash to
every URL it can be downloaded from:

```json
{
  "version": 1,
  "generated": "2026-11-08T00:00:00Z",
  "sources": {
    "chip8archive": { "name": "chip8Archive", "license": "CC0-1.0",
                       "home": "https://johnearnest.github.io/chip8Archive/" },
    "kripod":       { "name": "kripod/chip8-roms",
                       "home": "https://github.com/kripod/chip8-roms" },
    "chippie":      { "name": "Hosted by Chippie (from Zophar and David Winter)",
                       "home": "https://github.com/rbscholtus/chippie/tree/main/web/hosted-roms" }
  },
  "roms": {
    "<sha1>": [
      { "source": "chip8archive", "url": "https://johnearnest.github.io/chip8Archive/roms/<file>.ch8" },
      { "source": "kripod",       "url": "https://raw.githubusercontent.com/kripod/chip8-roms/<commit>/games/<file>.ch8" }
    ]
  }
}
```

**`romstore` rules:**

- **Downloads use `platform.Fetch`.** In the browser it calls `fetch()`
  through `syscall/js`, so CORS applies; natively it uses `net/http`. Go's
  `net/http` isn't linked into the WASM because it adds about 1.7 MB gzip
  (measured in M3); CI fails if it creeps back in.
- **SHA-1 is checked after every download.** A mismatch means try the next
  URL. If all fail, the game is shown as unavailable.
- **Prefer the original source.** `chippie` (our hosted copies) is used only
  where no CORS-friendly source exists.
- **Pin GitHub URLs to a commit**, so files can't change under us. The CI job
  updates the pins.
- **Cache downloaded ROMs** in IndexedDB, keyed by hash, so they work offline
  after the first play.
- **Never block `Update`.** Downloads run in goroutines; results arrive on a
  channel that `Update` polls. The TV shows a "tuning…" static effect while it
  waits.
- **Extra sources:** an advanced setting lets users add their own manifest URL
  later (manifest format version 1, same shape).
- **Local files still work:** drag-and-drop (`ebiten.DroppedFiles`) and a
  "load from device" button (browser file picker, §14). Local ROMs are
  identified by hash like any other.

## 10. Metadata

- **Embedded snapshot:** chip-8-database (`platforms.json`, `quirks.json`,
  `programs.json`, `sha1-hashes.json`) is embedded with `go:embed` under
  `catalogue/snapshot/`. The app works offline and on first load.
- **Refreshed at startup:** in the background, `catalogue` fetches the latest
  JSON from GitHub (CORS works). It validates the files (they parse, the
  schema version is known, they cross-reference correctly), swaps them in,
  and caches them in IndexedDB. Anything wrong means the app keeps the
  snapshot.
- **Weekly CI job:** refreshes the embedded snapshot and `sources.json`, and
  opens a PR if anything changed.
- **Second metadata source:** chip8Archive's own `programs.json` (colours,
  tickrate, quirk options for its 104 programs) is merged in. chip-8-database
  wins on conflicts. Conflicts are reported by `catalogue-gen` and are
  candidates for upstream fixes.


**The overlay:** Chippie doesn't wait for upstream. `catalogue/overlay.json`
holds its pending additions and corrections to chip-8-database, in the
database's own format, each linked to the upstream PR that proposes it (or
"pending"). It can add platforms (in `platforms.json`'s shape, checked like
the database's, and only ids `chip8` knows), programs and ROMs, and change
ROMs; platforms are added first, so ROMs can use them. `catalogue` applies it on top of the snapshot and the runtime
refresh; `catalogue-gen` applies it too, so its ROMs get download locations.
Whatever upstream already has, an entry skips, and `catalogue-gen` deletes it
from `overlay.json`, so the weekly refresh PR carries the cleanup (a change
counts as done only when upstream has every field it sets; an addition, when
upstream has its ROMs). The PR's summary lists the upstream PRs concerned:
marking them merged in [upstream/README.md](upstream/README.md) stays a manual
step. The goal is an empty overlay: everything upstream.

## 11. Presentation: the 1970s TV

The whole page is the Ebitengine canvas. The TV fills most of it; UI panels
slide in over it.

**The set:** a mid-1970s tabletop colour TV with a wood-grain veneer cabinet,
a rounded-rectangle picture tube, a rotary channel selector, a fabric speaker
grille and a rocker power switch. Artwork is AI-generated (§12), around a
light grey studio background that fills the page.

**Controls:**

| Control | Function | Keys |
|---|---|---|
| Power rocker | Off: the CRT collapses and the game pauses, the pilot lamp goes out. Any click, tap or key switches on again. | P |
| Channel dial (VHF, `U 2–13`, 13 click-stops) | Next/previous game in the library's current list (its search), skipping games that can't play yet. The OSD shows the game's channel number: its place in that list. | PageUp / PageDown |
| Speed knob (UHF fine tuning) | Instructions per frame, the settings panel's speed steps; saved per ROM | `[` / `]` |
| Volume knob | Buzzer and XO-CHIP audio volume, 0 = mute; saved | `-` / `=` |
| Tint knob | ROM colours → white → amber → green → Octo; saved per ROM | T |
| "< ROM list" under the TV | Opens the game library | Esc |
| "Settings >" under the TV | Settings: platform, quirks, trace log | F2 |

- **Knobs are separate sprites** cut from the cabinet (`cmd/skin-gen`), drawn
  over it and rotated by code. Click or tap: one step (left half down, right
  half up). Drag: the dial follows the pointer around its centre; knobs follow
  vertical movement. Mouse wheel: one step per notch. Each control's touch
  area is at least 44 pt across; where those overlap, the nearest wins.
- **OSD:** on channel change, the channel number and game title appear in
  green in the picture's corner for two seconds, as on period sets; volume,
  tint and speed changes show too. It is drawn into the picture before the
  CRT pass, so it glows and curves with it.
- **CRT shader** (from the prototype): curvature, scanlines, shadow mask,
  bloom, phosphor afterglow, power-on/off collapse, tint. A "clean" setting
  gives pixel-exact output. Afterglow also fixes CHIP-8's sprite flicker, so
  it's functional, not just decoration. Every effect, its numbers and its
  known limits: [crt-shader.md](crt-shader.md).
- **"Tuning" static:** a noise shader shown while a ROM downloads.

**The library drawer** (`app/library` model, drawn by `app`):

- Covers the window. A search field (title, author, year, platform; every
  word must match; short words like `xo`, `schip`, `chip8` and `chip48`
  find a platform), "Load a ROM from this device", then the ROMs in two groups:
  PLAY and COMING SOON (platforms still to come), each A–Z.
- **Platform filter:** a row of chips under the search field (All · CHIP-8 ·
  SUPER-CHIP · XO-CHIP, with how many games each has for the current
  search) narrows the list; it combines with the search and is saved on the
  device. Where space is tight the counts go, then SUPER-CHIP becomes SCHIP.
  The channel dial steps through the filtered list, so it then stays within
  one platform.
- On wide screens a details pane beside the list: description, authors,
  year, platform, and whether it plays offline or downloads.
- **Picking a game closes the drawer**, shows "tuning in" static while it
  downloads, and starts it. (The prototype's list stayed open because it was
  a permanent panel.)
- Touch: drag to scroll, tap on release (a drag never picks), 44 px rows.
- **Settings** (F2, or "Settings" under the picture): platform, speed, the
  seven quirks, reset to recommended, instruction trace. Speed and quirks
  apply at once; a platform change restarts the ROM. Saved per ROM hash.

**Layout and scaling:**

- **Landscape:** the TV centred and scaled to fit, the status lines under
  it; drawers overlay it. With the remote, it takes a column on the right.
- **Portrait (phones):** the TV at the top, scaled to the width. The on-screen
  remote (hex keypad) below it. Drawers cover the remote.
- **The picture** is a 2:1 rectangle across the middle of the tube; the rest of
  the tube shows the game's background colour, as a TV would. Games made for
  a sideways screen (the database's `screenRotation`, e.g. Sub-Terr8nia, 270°)
  are turned clockwise by that much and get a tall 1:2 picture; the on-screen
  display, keys and remote stay upright.
- **Crisp on high-density screens:** `Layout` returns the real pixel size
  (`outside size × ebiten.Monitor().DeviceScaleFactor()`). All UI sizes and
  fonts are multiplied by the same factor. The prototype used logical pixels,
  which is slightly soft on Retina screens.

## 12. TV artwork

### 12.1 How the art is used

The cabinet is one image with an **empty, flat, black screen area**. The CRT
shader draws the picture into that area, so the generated screen content is
never shown. The controls are **separate sprites, cut from the cabinet
itself**, so they turn (or flip) with matching lighting and scale.

**`cmd/skin-gen`** makes everything the app loads from the generated master
(kept outside the repository in `assets/source/`) and
`assets/tv/<skin>/recipe.json`, which holds the skin's name and the
measurements in the master's pixels. **There can be several skins**, one
folder each (`classic` "Walnut console", the default; `portable` "Portable";
`slim` "Slim walnut"); Settings has a "TV" row to choose one, saved on the
device. Every skin's `skin.json` is compiled in; its images are fetched from
`tv/<skin>/`. A skin may have fewer controls: whatever it lacks still works by
keyboard and Settings.

- **Cabinet:** cropped to the TV, in three sizes (JPEG), plus a greyscale
  **mask** (PNG) that is black where the tube shows through. The app combines
  them, draws the picture first and the cabinet on top, so the bezel frames
  the picture exactly, curved edges included. The crop's edges fade to
  transparent in the mask.
- **Backdrop:** a small, blurred image of the studio around the TV, drawn
  behind the cabinet and fading into the page colour, so the crop leaves no
  visible edge.
- **Controls:** each cut out round (or as a rectangle for the rocker),
  centred on its spindle. Only what turns is cut: the channel dial's numbered
  disc (not its chrome bezel, whose shine would spin) and each knob's cap.
- **Channel numbers:** the generated dial's numbers were garbled, so
  skin-gen erases the printed ring and redraws `U 2 … 13` (6 and 9 with a dot)
  in Go Medium.
- **`skin.json`**, at `@3x` (the cropped master's size), written from the same
  recipe, so the two never disagree:

```json
{
  "width": 2100, "height": 1554, "background": "#d1cfd0",
  "cabinet": "cabinet", "mask": "cabinet-mask",
  "backdrop": "backdrop.jpg", "backdropRect": { "x": -700, "y": -700, "w": 3500, "h": 2954 },
  "screen": { "x": 190, "y": 165, "w": 1306, "h": 1046 },
  "controls": [
    { "id": "channel", "kind": "dial",   "image": "channel", "cx": 1767, "cy": 328, "r": 98, "steps": 13 },
    { "id": "speed",   "kind": "knob",   "image": "speed",   "cx": 1969, "cy": 324, "r": 27, "minAngle": -135, "maxAngle": 135 },
    { "id": "volume",  "kind": "knob",   "image": "volume",  "cx": 1708, "cy": 564, "r": 32, "minAngle": -135, "maxAngle": 135 },
    { "id": "tint",    "kind": "knob",   "image": "tint",    "cx": 1841, "cy": 567, "r": 29, "minAngle": -60,  "maxAngle": 60 },
    { "id": "power",   "kind": "rocker", "image": "power",   "x": 1947,   "y": 534, "w": 45, "h": 75 }
  ]
}
```

`skin.json` is compiled in (it fixes the layout from the first frame); the
images are fetched. `app/skin` parses and checks it (everything inside the
cabinet, nothing overlapping) and does the hit testing and turning maths.

**Checking alignment:** `go run ./cmd/skin-gen -preview /tmp/tv.png -angle 30`
writes the composed TV with the controls turned, the tube hole in green and
outlines. In the app, F3 (or `-calibrate`, `?skin=debug`) outlines every
control, its touch area, the tube and the picture; natively, F5 reloads
`skin.json` after editing it.

### 12.2 Scaling across devices

Scaling one image up or down works, but quality is best when the art is made
large and scaled **down**:

- **Generate large**, at least 2400 px wide (upscale if the generator
  outputs less).
- **Pre-scaled by `cmd/skin-gen`** to three sizes: `@3x` is the cropped master
  (2100 px wide), `@2x` two thirds and `@1x` one third. The app picks the
  smallest one at least as wide as the TV on screen in real pixels, then
  scales that smoothly (`ebiten.FilterLinear`) to the exact size, and loads a
  bigger set if the window grows.
- **Formats:** cabinet and backdrop as JPEG, the tube mask and controls as
  PNG.
- **Loading:** fetched with `platform.Fetch` from `tv/` next to the page
  (natively read from `assets/tv/<skin>/`) after the WASM starts, in a goroutine, so
  the art doesn't add to the WASM download. The CRT picture works before the
  cabinet arrives; the cabinet then fades in.
- **Sizes:** cabinet 51/189/443 KB at @1x/@2x/@3x, backdrop 9 KB, the channel
  dial up to 53 KB, other controls under 10 KB.

### 12.3 Image brief

The prompts used for each cabinet, and tips for making new skins, are in
[tv-artwork.md](tv-artwork.md).

## 13. Input and audio

**Keyboard:**

- The default layout is the usual `1234/QWER/ASDF/ZXCV` → `123C/456D/789E/A0BF`.
- Read in `Update` with `ebiten.IsKeyPressed` and passed to
  `machine.SetKey` before `RunFrame`.
- If the database has a `keys` hint for a ROM, the remote highlights only
  those keys, and extra keys work next to the usual layout: the arrow keys
  press the hinted `up`/`down`/`left`/`right` keys, Space presses `a` and
  Shift `b` (`input.Extras`). Hints for a second keypad (0x10 and up) and
  player 2 get no extra keys. The keys line says which extras work, e.g.
  "Keys: ↑ W · ↓ S · A E (or arrows, Space)".
- `Fx0A` follows the original behaviour: it waits for a key to be pressed and
  then released. Timers keep counting down while it waits.

**Touch: the remote control.**

- On touch screens (when the browser reports one, or after the first touch),
  a 1970s-style remote with the 16-key hex
  keypad appears below the TV in portrait, or beside it in landscape.
- Multi-touch is supported (`ebiten.AppendTouchIDs`), so two keys can be held
  at once.
- Keys in the database's `keys` hint are lighter. The `touchInputMode` hint
  (e.g. a gamepad layout) isn't used yet.

**Text input:**

- The search field (`app/textfield.go`, from the prototype's `field.go`) uses
  Ebitengine's `exp/textinput` on touch screens: it brings up the phone's
  keyboard, and the search includes words still being composed. With a
  physical keyboard it reads typed characters directly, because the input
  method's asynchronous session can lose the first keys typed after focusing.

**Audio (`app/sound`):**

- Ebitengine's `audio` package. The audio context is created on the first
  user gesture (the power button), as browsers require.
- **CHIP-8/SCHIP:** a square wave, gated by the sound timer. On the COSMAC
  VIP platforms it sounds only while the timer is at least 2, because the
  VIP ignores a value of 1 (§6.3); every VIP beep is a frame shorter.
- **XO-CHIP:** the same stream plays the 16-byte, 1-bit pattern buffer at the
  pitch register's rate, from the first `F002`. A ROM that never loads a
  pattern gets the plain square wave. The position in the pattern carries
  over between frames and pattern loads, and restarts at the first bit when
  the buzzer switches on, since XO-CHIP resets it only when the sound timer
  reaches 0.
- **Volume** from the TV's volume knob.

## 14. Browser integration

Everything browser-specific lives in `platform/`, behind small interfaces,
with desktop stubs so `go run ./cmd/chippie` works natively:

| Need | Browser implementation |
|---|---|
| ROM and metadata cache | IndexedDB, through `syscall/js`, wrapped in goroutines and channels (from the prototype) |
| Small preferences | `localStorage` (synchronous; wrapped so the app works when storage is blocked) |
| "Load ROM from device" | A hidden `<input type="file">` created from Go. Its `change` event reads the file with `arrayBuffer()` and sends the bytes on a channel. This is the async case that was hard in the Rust version; goroutines make it a few lines. |
| Drag-and-drop ROMs | `ebiten.DroppedFiles()`, no JS needed |
| Network | `platform.Fetch`: `fetch()` through `syscall/js` (Go's `net/http` would add ~1.7 MB gzip); `net/http` natively |
| Tab hidden | `ebiten.IsFocused()`, to pause |

**Rule:** no call into `platform/` ever blocks `Update` or `Draw`. Every
browser call either returns immediately or runs in a goroutine and reports
back on a channel.

**The loader page** (`web/page.html`, from the prototype):
- Shows a title and a progress bar while `chippie.wasm` downloads.
- Unpacks it only if it's still gzip-compressed.
- Starts Go, then removes itself completely.

## 15. Testing

### 15.1 Unit tests

- **Instructions:** table-driven tests for each opcode × each relevant quirk.
  Set up registers, run one instruction, check the result.
- **Profiles:** `catalogue.Resolve` tested against the embedded snapshot.
- **Disassembler:** golden text per platform.
- **romstore:** with an `httptest` server: SHA-1 mismatch → next URL; all
  fail → unavailable.
- **Skin:** `skin.json` loads, and every control lies inside the cabinet.

### 15.2 Screenshot tests with the Timendus suite (v4.2)

**Headless test rig** (`internal/testrig`), pure Go, no Ebitengine:

- Load the ROM with a profile.
- `Poke(0x1FF, n)` to skip the ROM's menu.
- Run N frames, feeding scripted key events (`press key K at frame F, release
  at frame G`).
- Snapshot `Display()` and compare it with the golden frame, pixel by pixel.

**Golden frames come from the suite's own pictures.** The reference images in
`pictures/` are 768×384 with exactly two colours: 12× for 64×32 and 6× for
128×64. Every 12×12 (or 6×6) block was checked and is a single colour, so they
convert losslessly into native-resolution 1-bit frames. `cmd/golden-import`
does this once and writes them to `testdata/golden/`.

- **Pictures with 3+ colours** (`quirks.png`, `chip-8-logo.png`, `beep.png`):
  `golden-import` maps the palette explicitly. For `beep`, the test checks the
  buzzer state instead.
- **Platforms with no reference picture:** generate with
  `go test ./... -update`, check by eye, then commit.

**Golden file format:**

- A `.png` at native resolution (one pixel per CHIP-8 pixel, palette index =
  plane bits).
- A `.txt` ASCII version (`#` / `.`), so diffs are readable in git and in PRs.

**On failure:** the rig writes `actual.png`, `diff.png` (8× scale: extra lit
pixels red, missing ones orange, the rest of the expected frame dimmed) and a
side-by-side ASCII `diff.txt` to `testdata/failures/<platform>/<golden>/`. CI
uploads that folder as an artifact. Each run starts by clearing it.

**Test matrix:**

| ROM | `0x1FF` | Platforms / modes | Keys | Golden from |
|---|---|---|---|---|
| 1-chip8-logo | — | all | — | `chip-8-logo.png` |
| 2-ibm-logo | — | all | — | `ibm-logo.png` |
| 3-corax+ | — | all | — | `corax+.png` |
| 4-flags | — | all | — | `flags.png` |
| 5-quirks | 1 / 2 / 3 / 4 | CHIP-8, SCHIP modern, XO-CHIP, SCHIP legacy | — | `quirks*.png` + generated |
| 6-keypad | 1 / 2 / 3 | Ex9E, ExA1, Fx0A | scripted presses of 1 and 6 | `keypad-*.png` |
| 7-beep | — | all | press `B` | buzzer-state assertion |
| 8-scrolling | 1–5 | SCHIP lo-res modern/legacy, SCHIP hi-res, XO lo/hi-res | — | `*-scrolling.png` (SCHIP, see below); generated (XO-CHIP) |

- **Corrected pictures:** `lores-scrolling.png` and `hires-scrolling.png` show
  two border pixels as off that the ROM can't turn off. `cmd/golden-import`
  corrects exactly those pixels and explains why.
- **Pinning:** the suite is a git submodule pinned to tag `v4.2`. The ROM
  hashes are also recorded in the test table, so an accidental submodule bump
  fails loudly.
- **Repeatability:** a fixed seed (CXNN) and a frame count instead of wall
  time make every run identical.

### 15.3 Browser tests (Playwright)

Same approach as the prototype's test:

- Serve `web/`, launch Chromium with software WebGL (`--use-angle=swiftshader`).
- Wait for the loader to finish (`window.chippieTimings`).
- Check: the TV renders, a test ROM (IBM logo) shows the right picture in
  "clean" shader mode, search filters the library, a pick closes the drawer,
  and IndexedDB restores state after a reload.
- **Allow for software rendering:** ~5 FPS, so type with ≥150 ms between
  keys. Ebitengine's `exp/textinput` drops the first synthetic key in headless
  Chromium; that's a test artefact (real devices are fine), so tests type a
  throwaway character first.
- Desktop and phone viewports (390×844, DPR 2, touch).

### 15.4 Native run

`go run ./cmd/chippie` opens the same app in a desktop window. That's the
fastest way to work on UI and shaders. CI builds it but doesn't ship it.

## 16. Performance

Rule: **keep allocations in the per-frame path few, without making the code
awkward.** Readable code comes first; optimisation happens when a benchmark or
profile shows the need. Allocation matters more than usual here, because Go's
garbage collector runs on the browser's single thread under WASM, so the
benchmarks report allocations and CI tracks them. `sync.Pool` is not used for
now. Download buffers are the likely first candidate if it's ever needed.

**Per-frame path:** `RunFrame`, every instruction, display operations, trace
recording, the framebuffer → texture copy (`WritePixels` from a reused byte
slice), and shader uniforms (reuse the `map[string]any` and slices; don't
build new ones each frame).

**Performance tests, in `chip8/perf_test.go` and `app/crt/perf_test.go`:**

| Test | Kind | Fails CI? |
|---|---|---|
| `BenchmarkRunFrame/<platform>/<ticks>` | ns/frame and allocs/frame (`b.ReportAllocs()`), each platform, default tickrate and 1000 ticks, running a busy ROM (corax+, a sprite-heavy XO-CHIP game) | No; tracked |
| `BenchmarkTrace` | Recording into the trace ring buffer while running | No; tracked |
| `BenchmarkFramebufferCopy` | Copying the display into the reused pixel buffer | No; tracked |
| `BenchmarkDXYN/lores,hires,xo-2planes` | Sprite drawing, the hottest instruction | No; tracked |
| `BenchmarkScroll/<mode>` | SCHIP/XO scrolling | No; tracked |
| `BenchmarkResolveProfile` | Hash lookup + profile resolution on ROM load | No; tracked |

**Tracking:**

- CI runs the benchmarks on PRs and compares them with `main` using
  `benchstat`. The result is posted as a job summary.
- A slowdown of more than 10% shows as a warning, not a failure, because CI
  machines are noisy.

**Budgets:**

- **CPU:** `RunFrame` at 1000 ticks well under 1 ms natively. Earlier timing
  in WASM: about 40 ns per instruction, so 1000 ticks ≈ 0.05 ms per frame.
- **GPU:** 60 FPS on a recent iPhone. The prototype reached 50+, so the CRT
  pass needs work:
  - render it at ≤2× device scale and upscale;
  - use a cheaper bloom (one separable blur pass at quarter resolution)
    instead of 9 samples per pixel;
  - skip the phosphor pass when the picture hasn't changed.
- **Download:** WASM ≤ 6 MB gzip, checked in CI. M3 is 4.6 MB. Art fetched
  separately (§12.2).

## 17. Build, CI and deployment

**Local:**

```sh
# Native window (fastest for UI and shader work)
go run ./cmd/chippie

# Tests and lint
go test ./...
golangci-lint run

# Browser build: wasm + gzip + loader page → web/dist/
./scripts/build.sh
cd web/dist && python3 -m http.server 8080
```

**`scripts/build.sh`** (from the prototype):

- `GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o chippie.wasm ./cmd/chippie`
- Compress with gzip only, as `chippie-gz.wasm`. Brotli is not shipped: the
  loader can't unpack it (`DecompressionStream` has no brotli), and GitHub
  Pages doesn't serve pre-compressed files with `Content-Encoding`.
- Inline `$(go env GOROOT)/lib/wasm/wasm_exec.js` into the loader page.
- Copy the Timendus test ROMs (GPLv3), with their licence, into
  `roms/test-suite/`, unmodified and separate from the WASM.
- Copy the hosted ROMs, and each skin's TV art (`assets/tv/<skin>/`: `@1x/@2x/@3x` and the
  backdrop) into `tv/`.

**GitHub Actions:**

| Workflow | Trigger | Does |
|---|---|---|
| `ci.yml` | push, PR | lint, unit + screenshot + perf tests, WASM build, native build, Playwright smoke test |
| `bench.yml` | PR | benchmarks vs `main` with benchstat → job summary |
| `pages.yml` | push to `main` | build and deploy to GitHub Pages |
| `refresh.yml` | weekly + manual | refresh database snapshot and `sources.json`, write coverage report, open PR if changed |

**Compression on GitHub Pages:** the build ships the pre-gzipped
`chippie-gz.wasm` and the loader unpacks it (proven in the prototype). The
loader checks the gzip magic bytes first, so it also works if a host decodes
the file itself.

