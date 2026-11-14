# Draft 02: `npm run update` stops on an archive entry without options

- **Where:** issue on [chip-8/chip-8-database](https://github.com/chip-8/chip-8-database).
- **Status:** draft, waiting for review.
- **Tested 2026-11-15** on `4b2a2a3` with the CHIP-8 Archive at `761e3ff`,
  offline: the real script, with stand-ins for its two npm dependencies
  (Prettier and css-color-names, which play no part in the bug) and
  `fetch` serving a local copy of the archive. Unchanged, it fails with
  `TypeError: Cannot use 'in' operator to search for 'clipQuirks' in
  undefined`. With the fix, it runs through and adds exactly the four new
  programs (484 → 488), changing none of the existing ones.

## Review checklist

- [x] Bug reproduced and fix tested (above).
- [x] Text below approved.

## The issue

**Title:** update-chip8-archive.js fails on CHIP-8 Archive entries without `options`

**Body:**

> The CHIP-8 Archive's `programs.json` now has an entry without an `options`
> object: `mato8` (added 2023-01-14 as a program, to the archive in August
> 2026). `scripts/update-chip8-archive.js` assumes every entry has one:
>
> - `cleanedROM` destructures it:
>   `let { backgroundColor, fillColor, fillColor2, blendColor } = archiveProgram.options;`
> - `getQuirks` uses `"clipQuirks" in options`.
>
> Both throw a `TypeError` on `undefined` (`Cannot use 'in' operator to
> search for 'clipQuirks' in undefined`), so `npm run update` stops there
> and adds nothing, including the other programs added since the last
> update (Dino Run, Tick, Mr Worm).
>
> A one-line fix: treat a missing `options` as empty, e.g. at the top of
> `convert`'s loop:
>
> ```js
> const archiveProgram = { options: {}, ...object[fileName] };
> ```
>
> and use `archiveProgram` instead of `object[fileName]` in that loop (two
> places). With that change the update runs through, adds the four new
> programs and changes none of the existing ones. Its output then marks all
> four platforms for manual checking, as expected, and gives Dino Run an
> `origin` with an empty gamejam `reference`, because its archive entry has
> `"event": ""`.
