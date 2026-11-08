import { readFileSync } from "node:fs";
import { Page, expect } from "@playwright/test";
import { PNG } from "pngjs";

// Same as screenRect in app/game.go: the largest 2:1 rectangle centred in
// the canvas, with a margin and room for the status line.
export function screenRect(w: number, h: number) {
  const margin = 0.06, statusGap = 0.08, aspect = 2;
  const availW = w * (1 - 2 * margin);
  const availH = h * (1 - 2 * margin - statusGap);
  const sw = Math.min(availW, availH * aspect);
  const sh = sw / aspect;
  const x = (w - sw) / 2;
  const y = ((h - sh) / 2) * (1 - statusGap);
  return { x: Math.trunc(x), y: Math.trunc(y), w: Math.trunc(x + sw) - Math.trunc(x), h: Math.trunc(y + sh) - Math.trunc(y) };
}

// A sampled CHIP-8 frame: the colour at the centre of each emulated pixel.
export type Grid = { cols: number; rows: number; rgb: [number, number, number][] };

export async function sampleScreen(page: Page, cols = 64, rows = 32): Promise<Grid> {
  const png = PNG.sync.read(await page.screenshot());
  const r = screenRect(png.width, png.height);
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

// Waits until the loader has started Go and removed itself.
export async function waitForApp(page: Page) {
  await page.waitForFunction(() => (window as any).chippieTimings !== undefined, null, { timeout: 60_000 });
  await expect(page.locator("#loader")).toBeHidden();
}
