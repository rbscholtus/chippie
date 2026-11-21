import { test, expect, Page } from "@playwright/test";
import { appLayout, click, offline, press, running, stateOff, stateRunning, waitForApp, waitForArt } from "./screen";

// Desktop: the TV's controls by mouse, wheel and keyboard. The app
// publishes what they did (window.chippieLayout.status).
test.skip(({ hasTouch }) => hasTouch, "desktop only");

test.beforeEach(async ({ page }) => offline(page));

async function startGame(page: Page) {
  await page.goto("/?rom=game/glitchGhost&shader=clean");
  await waitForApp(page);
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
  await waitForArt(page);
}

async function control(page: Page, id: string, dx = 0): Promise<[number, number]> {
  const l = await appLayout(page);
  const [x, y] = l.controls[id];
  return [x / l.scale + dx, y / l.scale];
}

test("a click on the channel dial's right half tunes to the next game", async ({ page }) => {
  // Downloads stay blocked, but slowly, so the TV is seen tuning in.
  await page.route("https://raw.githubusercontent.com/**", async (route) => {
    await new Promise((r) => setTimeout(r, 5000));
    await route.abort();
  });
  await startGame(page);

  await click(page, ...(await control(page, "channel", 15)));

  await expect.poll(async () => (await appLayout(page)).status.channel, { intervals: [200] }).toBe(1);
  expect((await appLayout(page)).status.title).not.toBe("Glitch Ghost");
});

test("the wheel over the volume knob changes the volume", async ({ page }) => {
  await startGame(page);
  const before = (await appLayout(page)).status.volume;

  await page.mouse.move(...(await control(page, "volume")));
  await page.mouse.wheel(0, 100);

  await expect.poll(async () => (await appLayout(page)).status.volume, { intervals: [300] }).not.toBe(before);
});

test("the power switch and P switch off; a key switches back on", async ({ page }) => {
  await startGame(page);

  await click(page, ...(await control(page, "power")));
  await expect.poll(async () => (await appLayout(page)).status.state, { intervals: [300] }).toBe(stateOff);

  await press(page, "x");
  await expect.poll(async () => (await appLayout(page)).status.state, { intervals: [300] }).toBe(stateRunning);

  await press(page, "p");
  await expect.poll(async () => (await appLayout(page)).status.state, { intervals: [300] }).toBe(stateOff);
});

test("T turns the tint knob", async ({ page }) => {
  await startGame(page);

  await press(page, "t");

  await expect.poll(async () => (await appLayout(page)).status.tint, { intervals: [300] }).toBe(1);
});

test("Settings switches the TV skin, and it stays after a reload", async ({ page }) => {
  await startGame(page);
  expect((await appLayout(page)).status.skin).toBe("classic");

  await press(page, "F2"); // Settings: the TV row is first
  await press(page, "ArrowRight");

  await expect.poll(async () => (await appLayout(page)).status.skin, { intervals: [300] }).toBe("portable");
  await expect.poll(async () => (await appLayout(page)).art, { intervals: [500] }).toBe(true);

  await page.reload();
  await waitForApp(page);
  await expect.poll(async () => (await appLayout(page)).status.skin, { intervals: [300] }).toBe("portable");
});
