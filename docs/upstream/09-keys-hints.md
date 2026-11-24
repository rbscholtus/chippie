# Draft 09: keys hints for nine ROMs, notes for four

- **Where:** PR to [chip-8/chip-8-database](https://github.com/chip-8/chip-8-database),
  from branch `keys-hints` of the owner's fork, based on `master`. It only
  adds `keys` and `description` fields to ROMs the database already has.
- **Status:** submitted 2026-11-24: [chip-8-database#36](https://github.com/chip-8/chip-8-database/pull/36).
- **Overlay entries:** the ten "change" entries in
  `assets/catalogue/overlay.json`, linked to #36.

## What and why

The owner played the games Chippie gained from the Revival Studios pack
(Chippie PR #37) on 2026-11-24 and noted their controls. Where the authors'
notes exist, they agree. The database's `keys` field only has up, down,
left, right, a and b (and player 2's), so what fits goes there, and the
rest goes in the ROM's `description`, naming hex keys as the database's
other descriptions do ("Press 5 to begin").

**Keys hints** (hex keys):

| Program (database #) | ROM | SHA-1 | up | down | left | right | a | b | Source |
|---|---|---|---|---|---|---|---|---|---|
| Sokoban (#428) | `Sokoban (by hap)(2006).sc8` | `c7c59b38129fdcec5bb0775a9a141b6ba936e706` | 5 | 8 | 7 | 9 | A (select, undo) | 0 (restart) | hap's notes (the program's description); played |
| Loopz (#341) | `Loopz (by Andreas Daumann)(1993).sc8` | `3c444e43e5f02dac4324b7b24cd38ef4938a4b56` | C | D | 7 | 8 | 6 (turn) | 5 (drop, OK) | Daumann's `LOOPZ.DOC`; the program tests exactly these keys (`E0A1` with 7, 8, C, D, 6, 5) |
| Loopz (#341) | `Loopz (with difficulty select) (by hap)(2006).sc8` | `a2788177b820a28cd27e6d2d180340cb7f4948fb` | C | D | 7 | 8 | 6 (turn) | 5 (drop, OK) | as above (hap: "same keys as in loopz"); played |
| Astro Dodge (#210) | `Super Astro Dodge (by Revival Studios)(2008).sc8` | `c2a361700209116a300457eacbf33a8c40c01b83` | 2 | 8 | 4 | 6 | 5 (start) | | Revival's notes ("Button 2,4,6,8 will move your ship, button 5 will start the game", for both versions); played. The same as Astro Dodge Hires' hint |
| Astro Dodge (#210) | `Astro Dodge [Revival Studios, 2008].ch8` | `ac621d9fcada302ba6965768229ef130630bc525` | 2 | 8 | 4 | 6 | 5 (start) | | Revival's notes, as above |
| SuperWorm V4 (#72) | `SuperWorm V4 (by RB & Revival Studios)(2007).sc8` | `2d415bf1f31777b22ad73208c4d1ad27d5d4f367` | 2 | 8 | 4 | 6 | | | played |
| Climax Slideshow - Part 1 (#247) | `Climax Slideshow - Part 1 (by Revival Studios)(2008).sc8` | `9d9f88509b5033152b7b49d2c7ea3c3c5fce2bd6` | | | | | 5 (next picture) | | played; it reads key 5 only |
| Climax Slideshow - Part 2 (#248) | `Climax Slideshow - Part 2 (by Revival Studios)(2008).sc8` | `6bb78d8a0aba93ea18eabdd0134cbdccd1dc2d16` | | | | | 5 (next picture) | | as Part 1 |
| Alien-Inv8sion (#200) | `Alien-Inv8sion.xo8` | `24ef21009527ee674de44ccb37e37081654883f9` | 5 | 8 | 7 | 9 | 6 (fire) | | the program's description (Timendus' README); played |

**Descriptions** (the ROM's own; the programs' stay as they are):

- **Matches** (#352, `24fd50a95b84e3a42e336a06567a9752f17b9979`), whose
  keys don't fit the hint's names: "Keys 1 and 3 set fewer or more matches,
  and 2 starts the game. Player 1 takes 1, 2 or 3 matches with keys 4, 5
  and 6; player 2 with keys 7, 8 and 9." Played; the program reads 1, 2
  and 3 until a game starts.
- **Loopz (1993)**: "The HP48 original. On the calculator, its launcher
  chose the level and stored it in the RPL user flags with two check bytes
  ($AA), which the program reads with F285; without them, it exits at once
  (00FD). Hap's 2006 version adds a difficulty select and runs on any
  SUPER-CHIP interpreter. Keys: 7 and 8 move left and right, C and D up and
  down, 6 turns, 5 drops and confirms." Its code starts `00E0 00FF F285
  31AA 00FD 32AA 00FD`, and hap's notes on the 2006 version say the same
  ("v1 and v2 should be $AA, dunno why; v0 is the difficulty").
- **Climax Slideshow, Parts 1 and 2**: "Key 5 shows the next picture."

## Review checklist

- [x] SHA-1s are the files at Chippie's pinned commits (00SChip8
  `55cb8cd`, alien-inv8sion `aed7216`, kripod `c723a9e`).
- [x] Played in Chippie by the owner, 2026-11-24 (all but the CHIP-8 Astro
  Dodge, whose hint comes from Revival's notes for both versions).
- [x] The keys each program reads, checked headless (Climax: 5 only;
  Matches: 1, 2 and 3 at the start; Loopz: 7, 8, C, D, 6 and 5).
- [x] Text below approved (the owner, 2026-11-24: "chip-8-database PR is
  approved by me").

## The PR

**Title:** Keys hints for nine ROMs, and notes for four

**Body:**

> This adds `keys` hints to nine ROMs, and a `description` to four whose
> controls don't fit the hint's names or that need a note. All but the
> CHIP-8 Astro Dodge were played (its hint is from Revival's notes, which
> cover both versions), and they agree with the authors' notes where there
> are any:
>
> - **Sokoban** (hap): 5/8/7/9 move, A selects a floor and undoes a move,
>   0 restarts (hap's notes).
> - **Loopz**, both ROMs: 7/8 left and right, C/D up and down, 6 turns, 5
>   drops and confirms (Andreas Daumann's `LOOPZ.DOC`; hap's version uses
>   the same keys).
> - **Astro Dodge**, the SuperChip and CHIP-8 ROMs: 2/4/6/8 move, 5 starts
>   (Revival Studios' notes, for both), as in Astro Dodge Hires.
> - **SuperWorm V4**: 2/4/6/8.
> - **Climax Slideshow**, Parts 1 and 2: 5 shows the next picture.
> - **Alien-Inv8sion**: 5/8/7/9 move, 6 fires (Timendus' README).
>
> Descriptions:
>
> - **Matches**: its keys (1/3 set the number of matches, 2 starts, 4–6 and
>   7–9 for the two players) don't fit the hint.
> - **Loopz (1993)**: this is the HP48 original, which exits at once
>   (`00FD`) unless its calculator launcher has put the level and two
>   `$AA` check bytes in the RPL flags (it starts with `F285 31AA 00FD
>   32AA 00FD`). The note says so and points to hap's 2006 version, which
>   asks for the level itself.
> - **Climax Slideshow**, both parts: what key 5 does.
