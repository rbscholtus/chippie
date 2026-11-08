import { test, expect } from "@playwright/test";
import { golden, gridText, rowCentre, sampleScreen, touchDrag, touchTap, waitForApp } from "./screen";

// Phone only: touch input through the DevTools protocol (Chromium).
test.skip(({ hasTouch }) => !hasTouch, "touch tests run on the phone project");

// Rows of the M3 picker: GAMES heading, 6 games, TEST SUITE heading, 8 tests.
const ibmRow = 9;

test("tap to switch on, tap a test ROM, and it runs", async ({ page }) => {
  await page.goto("/?shader=clean");
  await waitForApp(page);

  await touchTap(page, 195, 400); // switch on: must not also pick a ROM
  await page.waitForTimeout(500);
  await touchTap(page, 100, rowCentre(ibmRow));

  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(golden("2-ibm-logo"));
});

test("drag scrolls the list without picking, then a tap picks", async ({ page }) => {
  // A short screen (like a phone in landscape), so the list must scroll.
  await page.setViewportSize({ width: 390, height: 500 });
  await page.goto("/?shader=clean");
  await waitForApp(page);

  await touchTap(page, 195, 400);
  await page.waitForTimeout(500);

  // Drag up by four rows, starting on Glitch Ghost: a drag must not pick.
  const scrolled = 4 * 44;
  await touchDrag(page, 100, rowCentre(1) + scrolled, rowCentre(1));
  await page.waitForTimeout(500);
  expect(gridText(await sampleScreen(page))).toContain("?"); // still the picker

  await touchTap(page, 100, rowCentre(ibmRow, scrolled));
  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(golden("2-ibm-logo"));
});

test("tapping under the picture goes back to the list", async ({ page }) => {
  await page.goto("/?rom=test/2-ibm-logo&shader=clean");
  await waitForApp(page);
  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(golden("2-ibm-logo"));

  const vp = page.viewportSize()!;
  await touchTap(page, vp.width / 2, vp.height - 40);
  await page.waitForTimeout(500);

  expect(gridText(await sampleScreen(page))).toContain("?"); // the picker covers it
});
