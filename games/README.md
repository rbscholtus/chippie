# Chippie's own games

CHIP-8 games written for Chippie in [Octo](https://github.com/JohnEarnest/Octo),
bundled with the app. Everything in this folder is **CC0 1.0** ([LICENSE](LICENSE)):
use it however you like.

| Game | Platform | What it is |
|---|---|---|
| [Depot](depot/) | XO-CHIP, 500 ticks per frame | Lane crossing: cross the yard, ride the pallets, deliver to the lit hatch |

## Building

```sh
./scripts/build-games.sh     # from the repository root; needs Node
```

This assembles `games/<name>/<name>.8o` with
[`@chipcode/octo-assembler`](https://www.npmjs.com/package/@chipcode/octo-assembler)
(pinned in `package.json`) into `assets/roms/<name>.ch8`, and updates the
ROM's SHA-1 in `assets/roms/bundled.json` and `catalogue/overlay.json`.
Commit all three. The built ROM is committed so Go builds and tests never
need Node; CI rebuilds it and fails if it's stale.

## Testing

```sh
go test ./games/...          # headless: golden frames, state, frame budget
go test ./games/... -update  # rewrite the golden frames, then check them by eye
```

The tests read the game's state at fixed addresses (the state block at the
top of each `.8o`). To debug a game step by step, paste its source into the
[Octo web IDE](https://johnearnest.github.io/Octo/) and set the options to
XO-CHIP with the game's tick rate.

## Adding a game

1. Create `games/<name>/<name>.8o` and run the build script.
2. Add an entry to `assets/roms/bundled.json` and an `add` entry with
   `"upstream": "pending"` to `catalogue/overlay.json` (title, authors,
   platform, tick rate, colours, keys).
3. Add it to the table above and to `assets/LICENSES.md`, and add tests in
   `games/<name>/`.
