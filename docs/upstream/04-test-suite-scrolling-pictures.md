# Draft 04: two pixels in the scrolling test's reference pictures

- **Where:** issue on [Timendus/chip8-test-suite](https://github.com/Timendus/chip8-test-suite).
- **Status:** draft, waiting for review.
- **Background:** Chippie's golden frames come from the suite's pictures
  (`cmd/golden-import`), which corrects exactly these four pixels
  (`loresBorderFixes`, `hiresBorderFixes`); see the M5 commit `743e18d`.

## Review checklist

- [x] Checked against the test's source at `v4.2`
  (`src/tests/8-scrolling.8o`, lines 163–200 and 236–273): the four pixels
  are drawn exactly once, and nothing else is drawn there.
- [x] The pictures on `main` are the same files as in `v4.2` (same git
  hashes), so the gaps are still there.
- [x] No existing issue about it (searched "scrolling" and "pixel").
- [ ] Text below approved.

## The issue

**Title:** Scrolling test: two border pixels shown off in `lores-scrolling.png` and `hires-scrolling.png`

**Body:**

> Thanks for the test suite! While importing your reference pictures as
> pixel-exact golden frames for my emulator, I found four pixels that I
> think are drawn wrong in the pictures, not in the emulator.
>
> In `pictures/lores-scrolling.png`, the pixels at (26, 15) and (37, 15)
> are off: the two corners where the upper box's sides meet the line below
> it. But in
> [`scrolling-draw-borders-lores`](https://github.com/Timendus/chip8-test-suite/blob/v4.2/src/tests/8-scrolling.8o#L163-L200),
> the upper box's sides are drawn at x = 26 and x = 37 on rows 5–14 only,
> and row 15 is drawn by the two lines at x = 21–30 and x = 33–42. So each
> of those two pixels is XOR-drawn exactly once and should be on. The
> arrows end up inside the boxes and don't reach them.
>
> `pictures/hires-scrolling.png` has the same gaps at (58, 32) and (69, 32):
> [the hi-res borders](https://github.com/Timendus/chip8-test-suite/blob/v4.2/src/tests/8-scrolling.8o#L236-L273)
> draw the sides at x = 58 and x = 69 on rows 22–31, and row 32 with the
> lines at x = 53–62 and x = 65–74.
>
> It doesn't change the outcome of the test (the arrows are still in their
> boxes), but anyone comparing their output with the pictures pixel by
> pixel will see a difference there. Unless I'm misreading the code, the
> pictures could be regenerated, or these four pixels lit. I'm happy to
> send corrected versions of the two pictures if that helps.
