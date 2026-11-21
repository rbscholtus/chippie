import { test, expect } from "@playwright/test";
import { PNG } from "pngjs";
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
  // field (DESIGN §15.3; real phones don't): start with spaces, which the
  // search ignores.
  await page.keyboard.type("  ibm logo", { delay: 150 });
  await page.waitForTimeout(500);
  await touchTap(page, 100, rowCentre(1)); // the first result

  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(golden("2-ibm-logo"));
});

test("a drag scrolls the list and doesn't pick", async ({ page }) => {
  await page.goto("/?shader=clean");
  await waitForApp(page);

  await touchTap(page, 195, 400);
  await page.waitForTimeout(500);
  await touchDrag(page, 100, rowCentre(7), rowCentre(1)); // starts on a ROM
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

// keyColour samples a remote key's colour above its label: a third of the
// way from its centre towards the key above it.
async function keyColour(page, key: string) {
  const l = await appLayout(page);
  const [x, y] = l.remote![key];
  const rowGap = l.remote!["4"][1] - l.remote!["1"][1];
  const png = PNG.sync.read(await page.screenshot());
  const i = (Math.round(y - rowGap / 3) * png.width + Math.round(x)) * 4;
  return [png.data[i], png.data[i + 1], png.data[i + 2]].join();
}

async function openOnRemote(page, rom: string) {
  await page.goto(`/?rom=${rom}&shader=clean`);
  await waitForApp(page);
  await expect.poll(async () => (await appLayout(page)).remote !== undefined, { intervals: [500] }).toBe(true);
}

test("without a keys hint, the remote lights the keys the game reads", async ({ page }) => {
  await openOnRemote(page, "game/turnover77"); // no hint; reads 5 7 8 9 at once
  const plain = await keyColour(page, "1");
  for (const key of ["5", "7", "8", "9"]) {
    await expect.poll(() => keyColour(page, key), { intervals: [300] }).not.toBe(plain);
  }
  expect(await keyColour(page, "6")).toBe(plain);
});

test("a game that waits for any key lights no key on the remote", async ({ page }) => {
  await openOnRemote(page, "game/redOctober"); // no hint; waits for any key
  await page.waitForTimeout(1000);
  const plain = await keyColour(page, "1");
  for (const key of ["2", "4", "5", "6", "8", "A", "F"]) {
    expect(await keyColour(page, key)).toBe(plain);
  }
});

// The phone's keyboard is open while Ebitengine's hidden textarea has the
// DOM focus (headless Chromium shows no keyboard, so the tests check that).
const textInputFocused = (page) => page.evaluate(() => document.activeElement?.id === "ebitengine-textinput");

async function searchFor(page, query: string) {
  await page.goto("/?shader=clean");
  await waitForApp(page);
  await touchTap(page, 195, 400); // switch on: the library opens
  await page.waitForTimeout(500);
  await touchTap(page, 195, pickerSearchY);
  await page.keyboard.type("  " + query, { delay: 150 }); // see the first test about the spaces
  await page.waitForTimeout(500);
  expect(await textInputFocused(page)).toBe(true);
}

test("the keyboard closes when a game starts", async ({ page }) => {
  await searchFor(page, "glitch");
  await touchTap(page, 100, rowCentre(1)); // Glitch Ghost

  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
  await expect.poll(() => textInputFocused(page), { intervals: [300] }).toBe(false);
});

test("a closed keyboard doesn't come back on the next tap", async ({ page }) => {
  await searchFor(page, "glitch");

  // The user closes the keyboard (Safari's done button blurs the textarea).
  await page.evaluate(() => {
    const w = window as any;
    w.textInputFocusins = 0;
    document.addEventListener("focusin", (e) => {
      if ((e.target as HTMLElement).id === "ebitengine-textinput") w.textInputFocusins++;
    });
    (document.activeElement as HTMLElement).blur();
  });
  await page.waitForTimeout(500);
  await touchTap(page, 100, rowCentre(1)); // Glitch Ghost

  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
  expect(await page.evaluate(() => (window as any).textInputFocusins)).toBe(0);
});

// Chrome on iOS hides the keyboard with its keyboard-down key while the
// textarea keeps the focus, then ignores focus() from code. Headless
// Chromium never shows a keyboard, so it looks the same: the field puts the
// keyboard box, a real but invisible text box, over itself, and a tap on
// the field focuses the box, which hands the focus to the textarea
// (DESIGN §13).
test("a hidden keyboard puts a real text box over the search field", async ({ page }) => {
  await searchFor(page, "glitch");

  const box = page.locator("#chippie-keyboard-box");
  await expect(box).toBeVisible();
  const b = (await box.boundingBox())!;
  expect(b.y).toBeLessThan(pickerSearchY);
  expect(b.y + b.height).toBeGreaterThan(pickerSearchY);

  await page.evaluate(() => {
    const w = window as any;
    w.boxFocused = false;
    document.getElementById("chippie-keyboard-box")!.addEventListener("focus", () => (w.boxFocused = true));
  });
  await touchTap(page, 195, pickerSearchY);
  expect(await page.evaluate(() => (window as any).boxFocused)).toBe(true);
  expect(await textInputFocused(page)).toBe(true);

  await touchTap(page, 100, rowCentre(1)); // Glitch Ghost
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
  await expect(box).toBeHidden();
});

test("when the TV is off, Settings under it switches on and opens them", async ({ page }) => {
  await page.goto("/?rom=game/glitchGhost&shader=clean");
  await waitForApp(page);
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
  await page.keyboard.press("p"); // switch off
  await expect.poll(async () => (await appLayout(page)).status.state, { intervals: [300] }).toBe(0);

  const l = await appLayout(page);
  await touchTap(page, ...centre(l.settings, l.scale));

  await expect.poll(async () => (await appLayout(page)).status.state, { intervals: [300] }).toBe(2); // stateSettings
});
