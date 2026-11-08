# Asset origins and licences

Every file in `assets/`, and every file shipped next to the app, is listed
here with where it came from and its licence (CONTRIBUTING.md, "Assets").

## Bundled ROMs (`assets/roms/`, compiled into the app)

From [John Earnest's chip8Archive](https://github.com/JohnEarnest/chip8Archive),
commit `761e3ffc63f43e6a849e80714122feff2afca208`. The archive places
everything under **CC0 1.0** ("No Rights Reserved"). Platform and tick rate
come from [chip-8-database](https://github.com/chip-8/chip-8-database).

| File | Title | Author | SHA-1 |
|---|---|---|---|
| `glitchGhost.ch8` | Glitch Ghost | Jackie Kircher | `2cdcb3c29a5f013a991db5909ca8e18e27b3c42b` |
| `danm8ku.ch8` | Danm8ku | buffi | `ff6b8ac59bf281cd4b5ab6e161600b00f85a0265` |
| `turnover77.ch8` | Turnover '77 | "your name here" | `d03f27f85a1cf68465e0853cc0c4abee4a94a4e5` |
| `octopeg.ch8` | Octopeg | Chromatophore | `7a4a89870f2ab23c28024dd1c3dd52cf1af1ad00` |
| `redOctober.ch8` | Red October V | Kesha | `e14350d3b19443e5ad2848172bef9719a8680b01` |
| `anEveningToDieFor.ch8` | An Evening to Die For | John Earnest | `afd9fee7565c54970b6bd7758aa8aa7843dd2e86` |

## Timendus CHIP-8 test suite (shipped separately, not compiled in)

The test ROMs from [Timendus/chip8-test-suite](https://github.com/Timendus/chip8-test-suite)
(tag `v4.2`) are licensed under **GPLv3**. They are not compiled into
`chippie.wasm`. The build copies them, unmodified, with the suite's `LICENSE`
and a link to its source, into `roms/test-suite/` next to the app, and the app
fetches them at runtime. Native builds read them from
`third_party/chip8-test-suite/bin/`.

## Hosted ROMs (`web/hosted-roms/`, shipped separately, not compiled in)

Twelve ROMs that no browser-reachable archive carries, unmodified, from
[Zophar's Domain CHIP-8 public-domain packs](https://www.zophar.net/pdroms/chip8.html)
(ALIEN, ANT, BLINKY, CAR, FIELD, JOUST, PIPER, RACE, SPACEFIG, UBOAT, WORM3 from
the Super Chip Games Pack; KALEID from the Chip-8 Games Pack). Credit belongs to
their authors. The build copies them, with `web/hosted-roms/README.md` and the
authors' `.DOC` notes, into `roms/hosted/` next to the app.

## Fonts

- **Go fonts** (`golang.org/x/image/font/gofont`), BSD 3-Clause, by the Go
  Authors and Bigelow & Holmes. Used for the M3 picker and messages; a
  period font replaces them in M6.

## Game metadata (`catalogue/snapshot/`, compiled in)

- **chip-8-database** JSON files: MIT, © 2023 The CHIP-8 database authors.
  Program descriptions are the original authors' text, republished in good
  faith with credit (see `catalogue/snapshot/SNAPSHOT.md` for the full licence
  and the takedown note).
- **chip8Archive** `programs.json`: CC0 1.0.

ROMs in the library are not shipped: they are downloaded from their original
archives (chip8Archive, kripod/chip8-roms, the Timendus test suite) at pinned
commits and cached in the browser, apart from the hosted ROMs above.
