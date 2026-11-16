# Depot

A lane-crossing game for XO-CHIP (128×64, two planes, 500 ticks per frame).
You're a courier in a parcel depot at night: cross the yard without being
hit by a forklift, ride the pallets over the sorting chute, and deliver the
parcel to the lit hatch at the top. Only one hatch is lit at a time, so plan
your route. Fill all four to clear the round; every round is faster.

**Status:** done (M9, 2026-11-17). Design:
[docs/plans/M9-game.md](../../docs/plans/M9-game.md).

Press E (key 6) on the title screen to start. The best score is kept in the
flag registers, which Octo and Chippie keep between sessions.

## Controls

| Action | Key | Chippie keyboard |
|---|---|---|
| Up | 5 | W or ↑ |
| Left | 7 | A or ← |
| Down | 8 | S or ↓ |
| Right | 9 | D or → |
| Start | 6 | E or Space |

## Rules and scoring

- Three lives. A forklift or trolley costs one; so does standing in the
  chute with no pallet under you, being carried off the edge, or running
  out of time.
- The time bar holds 16 s per delivery. From round 3 the lit hatch moves to
  another empty one when it's half gone.
- 10 points for each row reached for the first time on a crossing, 50 for a
  delivery plus 1 per time-bar pixel left, 200 for clearing a round. The
  best score is saved in the flag registers.

## Design notes

- **Two planes:** the lanes and the HUD are on plane 1, the courier and the
  lit hatch on plane 2. Standing on a pallet shows the overlap colour.
- **Locked to the timer:** one game frame per 60 Hz frame (the delay timer),
  so the game plays the same at any tick rate that keeps up (the tests check
  500 and 1000).
- **Frame budget:** 500 instructions a frame. Profiled over 10 minutes of
  random play (2026-11-17), a play frame averages ~270 (99% under ~435);
  dying, game over and building the board stay under ~380, the title under
  ~245. Moving the lanes is ~70% of all the work. Big jobs (the title, a
  new board, a delivery, game over) are spread over several frames, and
  half the lanes move each frame.
- **Known limit:** the rare worst case, a hop to a new row while riding a
  pallet (score redraw, hop sound, carry and three busy lanes in one
  frame), needs a few more than 500 instructions, so that frame starts a
  frame late (2 in the 10 minutes). It isn't noticeable in play, so it's
  left as is; deferring the score redraw a frame would fix it.
- **Collisions are worked out**, not read from VF: the courier and the lanes
  are on different planes. Each lane object is checked against the courier's
  x range.
- **State block** at 0x202 and the lane table at 0x218: the tests read and
  poke them (see the top of `depot.8o`).

## Tuning

Lane speeds are in 1/16 px per update; a lane updates every other frame,
so 16 is 30 px a second. Each cleared round adds 2 to every lane, up to 40
(75 px a second). Play-tested and kept as they were on 2026-11-17.

| Lane | Objects | Speed (round 1) | px/s | Cap reached |
|---|---|---|---|---|
| Chute 1 | 3 pallets, left | 10 | 19 | round 16 |
| Chute 2 | 4 pallets, right | 16 | 30 | round 13 |
| Chute 3 | 3 pallets, left | 12 | 22.5 | round 15 |
| Yard 1 | 3 forklifts, right | 12 | 22.5 | round 15 |
| Yard 2 | 3 trolleys, left | 18 | 34 | round 12 |
| Yard 3 | 2 forklifts, right | 24 | 45 | round 9 |

The numbers are in the lane table and `lane-speeds` in `depot.8o`, and the
constants `SPEED-STEP`, `SPEED-CAP`, `TIME-FULL` and `TICK-FRAMES`.

## Licence

CC0 1.0, like everything in `games/`.
