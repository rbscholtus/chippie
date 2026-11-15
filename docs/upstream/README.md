# Upstream drafts (M8)

Everything Chippie proposes to other projects, drafted here first
([docs/plans/M6-M8.md](../plans/M6-M8.md), M8). Nothing is posted until the
owner approves the draft; then it's submitted with `gh` under the owner's
account and its link recorded in the draft, in `catalogue/overlay.json`
(for database changes) and in the table below.

| # | Draft | Where | Kind | Status |
|---|---|---|---|---|
| 01 | [Add Dino Run, Mato8 and Tick](01-chip8archive-new-programs.md) | chip-8-database | PR | [#30](https://github.com/chip-8/chip-8-database/pull/30), open |
| 02 | [Update script fails without `options`](02-update-script-missing-options.md) | chip-8-database | issue | [#29](https://github.com/chip-8/chip-8-database/issues/29), open |
| 03 | [Support Mr Worm (#28)](03-comment-pr28-mr-worm.md) | chip-8-database#28 | comment | [posted](https://github.com/chip-8/chip-8-database/pull/28#issuecomment-6881844988) |
| 04 | [Two pixels in the scrolling pictures](04-test-suite-scrolling-pictures.md) | Timendus/chip8-test-suite | issue | [#38](https://github.com/Timendus/chip8-test-suite/issues/38), open |
| 05 | [Add 26 Octojam games](05-octojam-itch-games.md) | chip-8-database | PR (on #30) | [#31](https://github.com/chip-8/chip-8-database/pull/31), open |
| 06 | [Four alternate ROMs](06-alternate-roms.md) | chip-8-database | PR | [#32](https://github.com/chip-8/chip-8-database/pull/32), open |
| 07 | [A Hires CHIP-8 (64×64) platform](07-hires-platform.md) | chip-8-database | issue | [#33](https://github.com/chip-8/chip-8-database/issues/33), open |
| 08 | [Add the 8 Hires CHIP-8 programs](08-hires-roms.md) | chip-8-database | PR (with the platform) | [#34](https://github.com/chip-8/chip-8-database/pull/34), open |

**Deferred until the maintainers react to #29–#31 (owner's decision,
2026-11-15):** issues proposing a "where to download" field, asking where
the unsourced hashes came from. (The 7 metadata conflicts were dropped: see
the plan.) The 64×64 platform (drafts 07 and 08, from M10) was posted
anyway, on 2026-11-16, at the owner's request.

Submitted from the owner's fork, [rbscholtus/chip-8-database](https://github.com/rbscholtus/chip-8-database), one branch per PR.

Chippie doesn't wait for upstream: pending database changes live in
`catalogue/overlay.json` until they're merged ([DESIGN §10](../DESIGN.md#10-metadata)).

## Longer term

1. **Missing ROMs:** PRs to chip-8-database adding metadata for the ROMs it
   lacks, after confirming their platforms by running them in Chippie.
2. **Download sources:** an issue proposing a standard "where to download"
   field in the database. Today only 12 programs have a `urls` field. If it is
   accepted, Chippie's `sources.json` becomes a thin layer over the shared
   database and other emulators benefit too.
3. **Hires CHIP-8 platform:** propose the 64×64 platform and its 8 ROMs
   (drafts 07 and 08), which Chippie plays since M10.
4. **Conflicts:** report metadata disagreements between chip8Archive and
   chip-8-database found by `catalogue-gen`.
