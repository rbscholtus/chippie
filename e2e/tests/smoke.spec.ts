import { test, expect } from "@playwright/test";
import { appLayout, click, golden, gridText, offline, sampleScreen, waitForApp } from "./screen";

// Software rendering in headless Chromium is slow, so the tests poll for
// the expected picture instead of waiting a fixed time (README §15.3).

test.beforeEach(async ({ page }) => offline(page));

test("the IBM logo test ROM renders exactly", async ({ page }) => {
  await page.goto("/?rom=test/2-ibm-logo&shader=clean");
  await waitForApp(page);

  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(golden("2-ibm-logo"));
});

test("the CHIP-8 logo test ROM renders exactly, by SHA-1", async ({ page }) => {
  await page.goto("/?rom=30f27e5cee5b325fd1681ee98a14de60bfbe951f&shader=clean");
  await waitForApp(page);

  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(golden("1-chip8-logo"));
});

test("a SUPER-CHIP game runs in hi-res", async ({ page }) => {
  await page.goto("/?rom=game/turnover77&shader=clean");
  await waitForApp(page);

  // Sampled at 128x64: the picture is hi-res only if it isn't made of whole
  // 2x2 blocks.
  const fineDetail = async () => {
    const g = await sampleScreen(page, 128, 64);
    const lit = (x: number, y: number) => g.rgb[y * g.cols + x].join();
    for (let y = 0; y < g.rows; y += 2) {
      for (let x = 0; x < g.cols; x += 2) {
        if (lit(x, y) !== lit(x + 1, y) || lit(x, y) !== lit(x, y + 1)) return true;
      }
    }
    return false;
  };

  await expect.poll(fineDetail, { intervals: [500], timeout: 30_000 }).toBe(true);
});

test("a ROM Chippie hosts itself downloads and runs", async ({ page }) => {
  const hosted = page.waitForResponse((r) => r.url().includes("/roms/hosted/") && r.ok());
  await page.goto("/?rom=d6fa9dc9005dc0496f39ba52fef56f9fd0a5a158&shader=clean"); // KALEID
  await waitForApp(page);
  await hosted;

  await expect.poll(async () => (gridText(await sampleScreen(page)).match(/#/g) ?? []).length, { intervals: [500] })
    .toBeGreaterThan(0);
});

test("an XO-CHIP game shows its own colours on two planes", async ({ page }) => {
  await page.goto("/?rom=game/redOctober&shader=clean");
  await waitForApp(page);

  const colours = async () => new Set((await sampleScreen(page)).rgb.map((c) => c.join())).size;

  await expect.poll(colours, { intervals: [500], timeout: 30_000 }).toBeGreaterThan(2);
});

test("switching on opens the library", async ({ page }) => {
  await page.goto("/?shader=clean");
  await waitForApp(page);

  // The loader is gone, so the click reaches the canvas and switches on.
  await click(page, 10, 10);

  // The drawer covers the picture, so it's no longer plain black.
  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toContain("?");
});

test("a game made for a sideways screen gets a tall picture", async ({ page }) => {
  // Sub-Terr8nia: screenRotation 270 in chip-8-database.
  await offline(page, { "/roms/sub8.ch8": "sub8.ch8" });
  await page.goto("/?rom=64536d549c986e9edf25de9fa89db60d2ade85c0&shader=clean");
  await waitForApp(page);

  await expect
    .poll(async () => {
      const l = await appLayout(page);
      return l.status.title === "Sub-Terr8nia" && l.picture[3] > l.picture[2];
    }, { intervals: [500] })
    .toBe(true);
});
