# Draft 02: `npm run update` stops on an archive entry without options

- **Where:** issue on [chip-8/chip-8-database](https://github.com/chip-8/chip-8-database).
- **Status:** draft, waiting for review.
- **Found by reading** `scripts/update-chip8-archive.js` at `4b2a2a3`; not
  run (Chippie's session doesn't install the project's npm packages).

## Review checklist

- [ ] Text below approved.

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
> Both throw a `TypeError` on `undefined`, so `npm run update` stops there
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
> and use `archiveProgram` instead of `object[fileName]` in that loop.
