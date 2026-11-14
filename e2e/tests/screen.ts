import { readFileSync } from "node:fs";
import { Page, expect } from "@playwright/test";
import { PNG } from "pngjs";

// The app's layout and status, as it publishes them (app/game_flow.go
// publishLayout), in real (device) pixels: [x, y, w, h] and [x, y].
export type Box = [number, number, number, number];
export type AppLayout = {
  scale: number;
  art: boolean;
  cabinet: Box;
  tube: Box;
  picture: Box;
  library: Box;
  settings: Box;
  controls: Record<string, [number, number]>;
  remote?: Record<string, [number, number]>;
  status: { state: number; title: string; channel: number; volume: number; tint: number; speed: number };
};

// The app's states (app/game.go).
export const stateOff = 0, stateRunning = 4;

export async function appLayout(page: Page): Promise<AppLayout> {
  const json = await page.evaluate(() => (window as any).chippieLayout);
  return JSON.parse(json);
}

// The centre of a box, in CSS pixels (for clicks and taps).
export function centre(b: Box, scale: number): [number, number] {
  return [(b[0] + b[2] / 2) / scale, (b[1] + b[3] / 2) / scale];
}

// A sampled CHIP-8 frame: the colour at the centre of each emulated pixel.
export type Grid = { cols: number; rows: number; rgb: [number, number, number][] };

export async function sampleScreen(page: Page, cols = 64, rows = 32): Promise<Grid> {
  const png = PNG.sync.read(await page.screenshot());
  const [x0, y0, w, h] = (await appLayout(page)).picture;
  const r = { x: x0, y: y0, w, h };
  const rgb: [number, number, number][] = [];
  for (let y = 0; y < rows; y++) {
    for (let x = 0; x < cols; x++) {
      const px = Math.floor(r.x + ((x + 0.5) * r.w) / cols);
      const py = Math.floor(r.y + ((y + 0.5) * r.h) / rows);
      const i = (py * png.width + px) * 4;
      rgb.push([png.data[i], png.data[i + 1], png.data[i + 2]]);
    }
  }
  return { cols, rows, rgb };
}

// Renders a grid as golden-style text: '#' lit, '.' dark, '?' anything else.
export function gridText(g: Grid): string {
  let out = "";
  for (let y = 0; y < g.rows; y++) {
    for (let x = 0; x < g.cols; x++) {
      const [r, gr, b] = g.rgb[y * g.cols + x];
      out += r === 255 && gr === 255 && b === 255 ? "#" : r === 0 && gr === 0 && b === 0 ? "." : "?";
    }
    out += "\n";
  }
  return out;
}

export function golden(name: string): string {
  return readFileSync(new URL(`../../testdata/golden/${name}.txt`, import.meta.url), "utf8");
}

// A game is running when the screen shows only the emulated picture (at most
// four colours; an overlay's text has many more) and it has drawn something.
export async function running(page: Page) {
  const { rgb } = await sampleScreen(page);
  const counts = new Map<string, number>();
  for (const c of rgb) counts.set(c.join(), (counts.get(c.join()) ?? 0) + 1);
  const background = Math.max(...counts.values());
  return counts.size <= 4 && rgb.length - background > 20;
}

// Waits until the loader has started Go and removed itself.
export async function waitForApp(page: Page) {
  await page.waitForFunction(() => (window as any).chippieTimings !== undefined, null, { timeout: 60_000 });
  await expect(page.locator("#loader")).toBeHidden();
}

// Touch helpers (Chromium only): Playwright's tap() presses and releases
// within one frame, which the app can miss; real fingers are slower.
export async function touchTap(page: Page, x: number, y: number) {
  const cdp = await page.context().newCDPSession(page);
  await cdp.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: [{ x, y }] });
  await page.waitForTimeout(150);
  await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
  await page.waitForTimeout(150);
}

export async function touchDrag(page: Page, x: number, fromY: number, toY: number) {
  const cdp = await page.context().newCDPSession(page);
  await cdp.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: [{ x, y: fromY }] });
  const steps = 10;
  for (let i = 1; i <= steps; i++) {
    await page.waitForTimeout(50);
    await cdp.send("Input.dispatchTouchEvent", { type: "touchMove", touchPoints: [{ x, y: fromY + ((toY - fromY) * i) / steps }] });
  }
  await page.waitForTimeout(100);
  await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
  await page.waitForTimeout(150);
}

// A mouse click that lasts long enough for the app to see it.
export async function click(page: Page, x: number, y: number) {
  await page.mouse.move(x, y);
  await page.mouse.down();
  await page.waitForTimeout(150);
  await page.mouse.up();
  await page.waitForTimeout(150);
}

// Touch layout of the drawer in CSS pixels (app/ui.go computeLayout): the
// search field (one 44 px row) sits below a debug line and two paddings,
// and the list starts half a padding below it. Row 0 is "Load a ROM from
// this device", row 1 the PLAY heading, row 2 the first ROM.
export const pickerSearchY = 24 + 16 + 16 + 22;
export const pickerListTop = 24 + 16 + 16 + 44 + 8;
export const touchRowHeight = 44;
export function rowCentre(row: number, scrolled = 0) {
  return pickerListTop + row * touchRowHeight + touchRowHeight / 2 - scrolled;
}

// The tests never use the internet: GitHub (ROM downloads and the
// background database refresh) is blocked, except for files a test serves
// itself from fixtures/.
export async function offline(page: Page, serve: Record<string, string> = {}) {
  await page.route("https://raw.githubusercontent.com/**", (route) => {
    for (const [suffix, fixture] of Object.entries(serve)) {
      if (route.request().url().endsWith(suffix)) {
        return route.fulfill({
          body: readFileSync(new URL(`../fixtures/${fixture}`, import.meta.url)),
          headers: { "Access-Control-Allow-Origin": "*", "Content-Type": "application/octet-stream" },
        });
      }
    }
    return route.abort();
  });
}

// Desktop (mouse) layout of the drawer in CSS pixels at 1×: a debug-line
// gap and padding above the search field (24 px), then the list.
export const desktopListTop = 24 + 16 + 16 + 24 + 8;
export const desktopRowHeight = 24;
export function desktopRow(row: number) {
  return desktopListTop + row * desktopRowHeight + desktopRowHeight / 2;
}

export const br8kout = "31fc1c53cc610a9f4b9c5705c5a0f33fc028d123";
