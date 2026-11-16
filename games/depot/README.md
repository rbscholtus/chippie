# Depot

A lane-crossing game for XO-CHIP (128×64, two planes, 500 ticks per frame).
You're a courier in a parcel depot at night: cross the yard without being
hit by a forklift, ride the pallets over the sorting chute, and deliver the
parcel to the lit hatch at the top. Only one hatch is lit at a time, so plan
your route. Fill all four to clear the round; every round is faster.

**Status:** in development (step 6 of 9: the whole game with its title
and game-over screens, best score and sound; arrow keys, saving the best
score between sessions and tuning come next). Design and build order:
[docs/plans/M9-game.md](../../docs/plans/M9-game.md).

Press E (key 6) on the title screen to start. The best score is kept in the
flag registers, so Octo keeps it between sessions; Chippie will from step 8.

## Controls

| Action | Key | Chippie keyboard |
|---|---|---|
| Up | 5 | W |
| Left | 7 | A |
| Down | 8 | S |
| Right | 9 | D |
| Start | 6 | E |

## Licence

CC0 1.0, like everything in `games/`.
