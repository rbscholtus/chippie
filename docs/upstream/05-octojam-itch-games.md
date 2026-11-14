# Draft 05: add 26 Octojam games to chip-8-database

- **Where:** PR to [chip-8/chip-8-database](https://github.com/chip-8/chip-8-database),
  from branch `add-octojam-itch` of the owner's fork, **built on #30's
  branch** (`add-chip8archive-2026`): both append to the end of
  `programs.json`, so this one numbers its programs 487–512 and asks for #30
  to be merged first.
- **Status:** submitted 2026-11-15: [chip-8-database#31](https://github.com/chip-8/chip-8-database/pull/31).
- **Overlay entries:** the 26 Octojam "add" entries in `catalogue/overlay.json`;
  [`05-octojam/programs-append.json`](05-octojam/programs-append.json) and
  [`05-octojam/sha1-hashes-append.json`](05-octojam/sha1-hashes-append.json)
  are generated from them and are exactly what gets appended.

## Where the data comes from

- **The ROMs:** each game's playable itch.io page is an Octo HTML export
  with the ROM embedded; [maxisimonazzi/chip8-emulator](https://github.com/maxisimonazzi/chip8-emulator)
  extracted them (`roms/octojam/`) along with each export's Octo settings
  (`data/octojam-itch.json`). The SHA-1s are of those files.
- **The entries:** converted the way the database's own
  `scripts/update-chip8-archive.js` converts Octo settings: `origin`
  (gamejam), `release`, the itch.io page as `urls`, a platform where the
  quirks match one exactly, otherwise `quirkyPlatforms` (as for Skyward),
  `tickrate`, `colors`, `touchInputMode`, `fontStyle`, `screenRotation`.
- **Title:** "Octojam 10 Entry - Fencing" shortened to "Fencing" (the
  owner's choice, 2026-11-15).

## Review checklist

- [x] All 26 run headless in Chippie for 20 seconds with these settings,
  without errors (2026-11-15).
- [x] Played in Chippie by the owner, 2026-11-15: all work (speed seemed
  fine).
- [x] **Rotation:** H8 2: Carry Flag (270°), Destroy The Line (270°) and
  Octris (90°) re-checked the right way up by the owner, 2026-11-15, after
  Chippie learned to apply `screenRotation`: all play well.
- [x] **The three rotation corrections** (below) played turned, 2026-11-15:
  all play well. (Computer Simulator shows no text, which isn't a rotation
  problem.)
- [x] **Octojam 6:** nothing to add. Of its 11 entries, 10 are already in
  the database (6 from itch.io, 4 via the CHIP-8 Archive), and Dump Trump's
  page has no playable version.
- [x] Text below approved.

## The programs

| # | Title (itch.io page) | Authors | Event | Platform | Tickrate | Rotation |
|---|---|---|---|---|---|---|
| 487 | [Bad Apple!!](https://koorogi.itch.io/bad-apple) | koorogi | Octojam 9 | xochip | 20000 | |
| 488 | [Bad Apple](https://timendus.itch.io/bad-apple) | Timendus | Octojam 9 | xochip | 40000 | |
| 489 | [Beware of Sn8k!](https://cratmang.itch.io/beware-of-sn8k) | Cratmang | Octojam 9 | xochip (quirky) | 20000 | |
| 490 | [BIM Logo Animation](https://sysl.itch.io/bim-logo-animation) | Sysl | Octojam 9 | xochip | 500 | |
| 491 | [Blobo](https://voror.itch.io/blobo) | Voror | Octojam 9 | superchip (quirky) | 100 | |
| 492 | [C8SALT logo](https://ninjaweedle.itch.io/c8salt-logo) | NinjaWeedle | Octojam 9 | superchip (quirky) | 7 | |
| 493 | [Chipception OS](https://timendus.itch.io/chipception-os) | Timendus | Octojam 10 | xochip | 200000 | |
| 494 | [Collision Course](https://ninjaweedle.itch.io/collision-course) | NinjaWeedle | Octojam 9 | modernChip8 (quirky) | 200 | |
| 495 | [Destroy The Line](https://atomr.itch.io/destroy-the-line) | TomR | Octojam 9 | superchip (quirky) | 7 | 270° |
| 496 | [Dodge Catch Shoot](https://miratha.itch.io/dodge-catch-shoot) | miratha | Octojam 10 | modernChip8 (quirky) | 500 | |
| 497 | [Duel Of The F8s: Special Edition](https://chromatophore.itch.io/duel-of-the-f8s-special-edition) | Chromatophore | Octojam 10 | xochip | 20000 | |
| 498 | [Enter the mine](https://tquester.itch.io/enter-the-mine) | tquester | Octojam 10 | superchip (quirky) | 100 | |
| 499 | [F-Chip8 Racing](https://darklynx.itch.io/f-chip8-racing) | darklynx | Octojam 10 | modernChip8 (quirky) | 20 | |
| 500 | [Falling Ghosts](https://veganjay.itch.io/falling-ghosts) | veganjay | Octojam 9 | modernChip8 (quirky) | 100 | |
| 501 | [Fencing](https://estus-dev.itch.io/fencing) | Estus Dev | Octojam 10 | modernChip8 (quirky) | 20 | |
| 502 | [H8 2: Carry Flag](https://chromatophore.itch.io/h8-2-carry-flag) | Chromatophore | Octojam 9 | xochip (quirky) | 20000 | 270° |
| 503 | [Halloween-Blaster](https://tquester.itch.io/halloween-blaster) | tquester | Octojam 10 | superchip (quirky) | 100 | |
| 504 | [Invisible Man](https://mremerson.itch.io/invisible-man) | MrEmerson | Octojam 7 | originalChip8 | 7 | |
| 505 | [Kanji Calculator (WIP)](https://cratmang.itch.io/kanji-calculator) | Cratmang | Octojam 10 | xochip | 200 | |
| 506 | [Man Hang](https://atomr.itch.io/man-hang) | TomR | Octojam 10 | originalChip8 | 15 | |
| 507 | [Octogon Trail](https://tarsi.itch.io/octogon-trail) | tarsi | Octojam 9 | xochip | 500 | |
| 508 | [Octris](https://zunil.itch.io/octris) | sunil. | Octojam 9 | modernChip8 (quirky) | 100 | 90° |
| 509 | [Termlib demo](https://pushfoo.itch.io/termlib-demo) | pushfoo | Octojam 10 | xochip | 100 | |
| 510 | [RPN Calculator](https://tquester.itch.io/upn-calculator) | tquester | Octojam 10 | superchip (quirky) | 200 | |
| 511 | [XO-NYANCAT!!](https://kouzeru.itch.io/xo-nyancat) | Kouzerumatsukite | Octojam 9 | xochip | 20000 | |
| 512 | [XO-TRACKER DEMO](https://kouzeru.itch.io/xo-tracker-demo) | Kouzerumatsukite | Octojam 8 | xochip | 6400 | |

## Also: rotation for three games already in the database

These Octojam games are in `programs.json`, but without the
`screenRotation` their Octo exports set, so they show sideways. In the same
PR, one line added to each ROM entry (from the overlay's three "change"
entries):

| Program | SHA-1 | `screenRotation` | Octo export |
|---|---|---|---|
| Computer Simulator | `46b281516a3e9d1526bea224b79cc18ddd71833d` | 90 | [atomr.itch.io/computer-simulator](https://atomr.itch.io/computer-simulator) |
| Infini Br8kr | `2cda3b309234e693e5ab6179767a8f019dfd5c6e` | 90 | [hailthefish.itch.io/infinibr8kr](https://hailthefish.itch.io/infinibr8kr) |
| Octo Space Program | `838706ee2d7001e6e909360f9a02f48d81d3a0d6` | 270 | [badcloudgames.itch.io/octo-space-program](https://badcloudgames.itch.io/octo-space-program) |

One entry in full (Octris):

```json
{
  "title": "Octris",
  "authors": [
    "sunil."
  ],
  "release": "2022",
  "origin": {
    "type": "gamejam",
    "reference": "Octojam 9"
  },
  "urls": [
    "https://zunil.itch.io/octris"
  ],
  "roms": {
    "6d073d659c5bfe625498f1cb45b9a3684463d721": {
      "file": "octris.ch8",
      "platforms": [],
      "quirkyPlatforms": {
        "modernChip8": {
          "wrap": true
        }
      },
      "tickrate": 100,
      "touchInputMode": "swipe",
      "fontStyle": "octo",
      "screenRotation": 90,
      "colors": {
        "pixels": [
          "#996600",
          "#ffcc00"
        ],
        "buzzer": "#ffaa00",
        "silence": "#000000"
      }
    }
  }
}
```

## The PR

**Title:** Add 26 Octojam games from itch.io

**Body:**

> Octojams 6–10 were hosted on itch.io, and many of their entries never
> made it into the CHIP-8 Archive. The database already has nearly all of
> Octojam 6–8, but none of Octojam 9 and 10. This adds the 26 that are
> missing (13 from Octojam 9, 11 from Octojam 10, and one each from 7 and
> 8), appended after #30's programs: please merge #30 first.
>
> The ROMs are the ones embedded in each game's playable page on itch.io
> (an Octo HTML export), as extracted by
> [maxisimonazzi/chip8-emulator](https://github.com/maxisimonazzi/chip8-emulator).
> The entries convert each export's Octo settings the way
> `scripts/update-chip8-archive.js` does: origin, release year, the itch.io
> page in `urls`, the platform (or `quirkyPlatforms` where the quirks match
> none exactly), tickrate, colours, touchInputMode, fontStyle and
> screenRotation. "Octojam 10 Entry - Fencing" is entered as "Fencing".
>
> It also adds the `screenRotation` that the Octo exports of three games
> already in the database set, but their entries lack, so they show
> sideways: Computer Simulator (90), Infini Br8kr (90) and Octo Space
> Program (270).
>
> Each was played in Chippie (https://rbscholtus.github.io/chippie/) with
> exactly these settings, the right way up. `sha1-hashes.json` is extended
> to match.
