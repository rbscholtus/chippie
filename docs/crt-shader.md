# The CRT effect

How Chippie draws the emulated picture as a 1970s picture tube, as of M6. The
code is in `app/crt/`: `renderer.go` and two Kage shaders. The `clean` mode
(`?shader=clean`, `-shader clean`) skips all of this and draws the plain,
pixel-exact picture, which the tests use.

## Passes

1. **Upscale:** the framebuffer (64×32, 128×64 or XO-CHIP's two planes, in
   the game's palette) is scaled up with nearest-neighbour filtering into the
   picture rectangle, a 2:1 area across the middle of the tube. The rest of
   the tube is filled with the game's background colour. The on-screen
   display (channel, volume, tint, speed) is drawn on top here, so it goes
   through the same effects.
2. **Phosphor** (`phosphor.kage`): new glow = the larger of the current
   picture and the previous glow × 0.82, once per emulated frame.
3. **CRT** (`crt.kage`): everything else, every displayed frame.

## Effects

| Effect | How it's done | Where |
|---|---|---|
| Phosphor afterglow | Keeps 82% of the previous glow per emulated frame. Also hides CHIP-8's sprite flicker, so it's functional, not only decoration. | `phosphor.kage`, `PhosphorDecay` |
| Curvature | Barrel warp: coordinates scale by `1 + 0.06·r²` from the centre; outside is black | `warp` |
| Power on/off | The picture squashes vertically, then horizontally, into a dot, with a white flash; driven by `Power` (0–1, 40 updates each way) | `Power` |
| Bloom | The centre pixel plus 8 neighbours 3 px away, added at 60% | "Cheap bloom" |
| Tint | Multiplied by a warm tube colour (0.95, 0.93, 0.85) | `Tint` |
| Scanlines | Brightness 75–100% along a sine with a period of about 3 screen pixels | `scan` |
| Shadow mask | Aperture-grille stripes, one screen pixel each: red, green, blue boosted to 1.1, the others 0.8 | `mask` |
| Vignette | Darkens towards the edges: `1 - 1.6·d²` | |
| Flicker | A 2% brightness ripple over time | `Time` |
| Tube glow | A faint grey (0.03) across the whole tube while it's on | |
| Static | Grey noise in the framebuffer while a game downloads ("tuning in") | `FillStatic` |

The cabinet's mask hides everything outside the tube's real (rounded,
bulging) shape, so the shader can draw a plain rectangle.

## Known limits

Recorded on 2026-11-14 while reviewing the effect. Nothing is planned; these
are notes in case the look is revisited.

- **Scanlines and the shadow mask are sized in screen pixels**, not relative
  to the picture. On high-density screens (3 px ≈ 1 pt on a Retina display)
  they blend into an even texture and barely show, and their size changes
  with the screen.
- **Interlacing would be inaccurate.** Broadcast TV interlaced (odd lines one
  field, even lines the next). The COSMAC VIP's video chip drew the same
  lines every frame, each CHIP-8 row 4 lines tall, which gives visible dark
  gaps between lines rather than interlace twitter.
- **Possible improvements:** line gaps tied to the picture (e.g. 4 lines per
  CHIP-8 row, or about 240 over the picture), lit lines that widen when
  bright; a mask sized to the tube; a soft reflection on the glass; slight
  horizontal beam softness. Any of these should be checked against the
  iPhone frame rate.
