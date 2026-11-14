import { test, expect } from "@playwright/test";
import {
  appLayout, centre, golden, gridText, offline, pickerSearchY, rowCentre, running, sampleScreen, touchDrag, touchTap,
  waitForApp,
} from "./screen";

// Phone only: touch input through the DevTools protocol (Chromium).
test.skip(({ hasTouch }) => !hasTouch, "touch tests run on the phone project");

test.beforeEach(async ({ page }) => offline(page));

test("tap the search field, type, tap a result, and it runs", async ({ page }) => {
  await page.goto("/?shader=clean");
  await waitForApp(page);

  await touchTap(page, 195, 400); // switch on: must not also pick a ROM
  await page.waitForTimeout(500);
  await touchTap(page, 195, pickerSearchY); // focus: the phone keyboard opens
  // Headless Chromium drops the first keys typed into the input method's
  // field (README §15.3; real phones don't): start with spaces, which the
  // search ignores.
  await page.keyboard.type("  ibm logo", { delay: 150 });
  await page.waitForTimeout(500);
  await touchTap(page, 100, rowCentre(2)); // the first result

  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(golden("2-ibm-logo"));
});

test("a drag scrolls the list and doesn't pick", async ({ page }) => {
  await page.goto("/?shader=clean");
  await waitForApp(page);

  await touchTap(page, 195, 400);
  await page.waitForTimeout(500);
  await touchDrag(page, 100, rowCentre(8), rowCentre(2)); // starts on a ROM
  await page.waitForTimeout(1000);

  expect(gridText(await sampleScreen(page))).toContain("?"); // still the drawer
});

test("tapping under the picture opens the list or the settings", async ({ page }) => {
  await page.goto("/?rom=game/glitchGhost&shader=clean");
  await waitForApp(page);
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);

  let l = await appLayout(page);
  await touchTap(page, ...centre(l.library, l.scale)); // left: ROM list
  await page.waitForTimeout(500);
  expect(gridText(await sampleScreen(page))).toContain("?");

  await page.goto("/?rom=game/glitchGhost&shader=clean");
  await waitForApp(page);
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
  l = await appLayout(page);
  await touchTap(page, ...centre(l.settings, l.scale)); // right: settings
  await page.waitForTimeout(500);
  expect(gridText(await sampleScreen(page))).toContain("?");
});

test("the remote shows on a touch screen and presses keypad keys", async ({ page }) => {
  await page.goto("/?rom=test/6-keypad&shader=clean");
  await waitForApp(page);
  await expect.poll(async () => (await appLayout(page)).remote !== undefined, { intervals: [500] }).toBe(true);
  await page.waitForTimeout(1000); // the test's menu is drawn

  const before = gridText(await sampleScreen(page));
  const l = await appLayout(page);
  const [x, y] = l.remote!["1"]; // "1": the FX0A test
  const cdp = await page.context().newCDPSession(page);
  await cdp.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: [{ x: x / l.scale, y: y / l.scale }] });
  await page.waitForTimeout(300);
  await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });

  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).not.toBe(before);
});
