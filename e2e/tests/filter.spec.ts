import { test, expect, Page } from "@playwright/test";
import { appLayout, click, offline, touchTap, waitForApp } from "./screen";

// The library's platform filter chips, by mouse (desktop) and by touch
// (phone). The app publishes the filter and how many games are listed.
test.beforeEach(async ({ page }) => offline(page));

async function openLibrary(page: Page, hasTouch: boolean) {
  await page.goto("/?shader=clean");
  await waitForApp(page);
  if (hasTouch) await touchTap(page, 195, 400);
  else await click(page, 10, 10);
  await expect.poll(async () => (await appLayout(page)).status.state, { intervals: [300] }).toBe(1); // the drawer
}

test("the XO-CHIP filter lists only XO-CHIP games, and stays after a reload", async ({ page }, info) => {
  const hasTouch = !!info.project.use.hasTouch;
  await openLibrary(page, hasTouch);

  const all = (await appLayout(page)).status.listed;
  const l = await appLayout(page);
  const [x, y] = l.filters["xochip"];
  if (hasTouch) await touchTap(page, x / l.scale, y / l.scale);
  else await click(page, x / l.scale, y / l.scale);

  await expect.poll(async () => (await appLayout(page)).status.filter, { intervals: [300] }).toBe("xochip");
  const xo = (await appLayout(page)).status.listed;
  expect(xo).toBeGreaterThan(10);
  expect(xo).toBeLessThan(all);

  await openLibrary(page, hasTouch);
  await expect.poll(async () => (await appLayout(page)).status.filter, { intervals: [300] }).toBe("xochip");
  expect((await appLayout(page)).status.listed).toBe(xo);
});
