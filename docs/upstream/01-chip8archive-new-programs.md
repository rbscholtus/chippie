# Draft 01: add Dino Run, Mato8 and Tick to chip-8-database

- **Where:** PR to [chip-8/chip-8-database](https://github.com/chip-8/chip-8-database),
  from branch `add-chip8archive-2026` of the owner's fork.
- **Status:** submitted 2026-11-15: [chip-8-database#30](https://github.com/chip-8/chip-8-database/pull/30).
- **Overlay entries:** the three "add" entries in `catalogue/overlay.json`;
  once this is merged, the weekly refresh reports them done.

## Review checklist

- [x] Each ROM's SHA-1 matches chip8Archive's file at commit `761e3ff`.
- [x] Played in Chippie with these settings, 2026-11-15: Dino Run, Mato8 and
  Tick all work.
- [x] **Dino Run's platforms:** chip8Archive's settings for it (display
  wait and logic quirks on, the rest off) are the original COSMAC VIP's. It
  plays correctly as both `originalChip8` and `modernChip8` (2026-11-15), so
  both are listed, the author's first.
- [x] Compared with what the database's own script produces once its bug
  (draft 02) is fixed, tested 2026-11-15: every field matches, except that
  the script marks the platforms for manual checking, adds no `keys`, and
  gives Dino Run an empty gamejam `origin`. These entries set the
  play-tested platforms, add the keys and leave the empty origin out.
- [x] Text below approved.

## The PR

**Title:** Add Dino Run, Mato8 and Tick from the CHIP-8 Archive

**Body:**

> These three programs were added to the CHIP-8 Archive after the database
> was last updated from it: Dino Run (May 2026), Tick and Mato8 (August
> 2026). Mr Worm, also new in the archive, is already proposed in #28.
>
> The entries follow what `scripts/update-chip8-archive.js` produces
> (description, images, origin, tickrate, touchInputMode, fontStyle,
> colours), with the platforms set by playing each ROM, plus a `keys` hint
> where the archive's how-to names the keys. They're appended at the end of
> `programs.json`, and `sha1-hashes.json` is regenerated.
>
> Why not `npm run update`: it currently stops on Mato8, the only archive
> entry without `options` (see #ISSUE, which has a one-line fix). With that
> fix, its output matches these entries field by field, apart from the
> platforms (which it marks for manual checking), the `keys` hints, and an
> empty gamejam `origin` it gives Dino Run.
>
> Checked by playing each ROM in Chippie (https://rbscholtus.github.io/chippie/).

## The change

Appended at the end of `database/programs.json`:

```json
  {
    "title": "Dino Run",
    "description": "An endless runner game for Chip-8.",
    "release": "2026-05-25",
    "authors": ["Andrezinrc"],
    "images": ["screenshot.gif"],
    "roms": {
      "1a7a2138161b06e083722f978f8cbaaa1ad5344d": {
        "file": "dinorun.ch8",
        "platforms": ["originalChip8", "modernChip8"],
        "tickrate": 15,
        "touchInputMode": "none",
        "fontStyle": "octo",
        "colors": {
          "pixels": ["#c7f0d8", "#004020"],
          "buzzer": "#000000",
          "silence": "#000000"
        },
        "keys": {
          "a": 5
        }
      }
    }
  },
  {
    "title": "Mato8",
    "description": "Snake game (matopeli in Finnish)",
    "release": "2023-01-14",
    "authors": ["nortti"],
    "images": ["itch-image.png"],
    "origin": {
      "type": "gamejam",
      "reference": "GAMES MADE QVIICK???"
    },
    "roms": {
      "673f98c66168190aa5b1e5dda5d0d80da7826dd1": {
        "file": "mato8.ch8",
        "platforms": ["modernChip8"]
      }
    }
  },
  {
    "title": "Tick",
    "description": "Rhythm one button game for Chip-8",
    "release": "2026-08-11",
    "authors": ["StanislavDidus"],
    "images": ["recording.gif"],
    "roms": {
      "3d81c0462c722c2d8a79f6e1b571e9aa21d36b1f": {
        "file": "tick.ch8",
        "platforms": ["modernChip8"],
        "tickrate": 12,
        "touchInputMode": "none",
        "fontStyle": "octo",
        "colors": {
          "pixels": ["#1e1e1e", "#7bff00"],
          "buzzer": "#000000",
          "silence": "#000000"
        },
        "keys": {
          "a": 10
        }
      }
    }
  }
```

Appended to `database/sha1-hashes.json` (programs 484–486), as
`npm run generate-hashes-file` writes them:

```json
  "1a7a2138161b06e083722f978f8cbaaa1ad5344d": 484,
  "673f98c66168190aa5b1e5dda5d0d80da7826dd1": 485,
  "3d81c0462c722c2d8a79f6e1b571e9aa21d36b1f": 486
```

If #28 is merged first, its hash goes into program 364 and doesn't change
these numbers.
