import type { Locator, Page } from '@playwright/test';

import { bounds } from './bounds';
import { home } from './demo';
import { expect, test } from './fixtures';

/**
 * Gives the page what a browser that can take pushes, and was never asked, shows it: a push manager that subscribes at
 * once, the "default" permission, and a service worker that's ready. Neither headless Chromium, whose permission says
 * "denied", nor Playwright's WebKit, which has no push manager, can subscribe otherwise.
 */
async function fakePush(page: Page) {
  await page.addInitScript(() => {
    let subscription: object | null = null;
    const pushManager = {
      getSubscription: async () => subscription,
      subscribe: async () => {
        subscription = {
          options: { applicationServerKey: new Uint8Array([1, 2, 3]).buffer },
          toJSON: () => ({ endpoint: 'https://push.example/1' }),
          unsubscribe: async () => true,
        };
        return subscription;
      },
    };
    Object.defineProperty(window, 'PushManager', { value: class {} });
    if ('Notification' in window) {
      Object.defineProperty(Notification, 'permission', { get: () => 'default' });
    } else {
      Object.defineProperty(window, 'Notification', { value: { permission: 'default' } });
    }
    Object.defineProperty(navigator.serviceWorker, 'register', { value: async () => ({}) });
    Object.defineProperty(navigator.serviceWorker, 'ready', { value: Promise.resolve({ pushManager }) });
  });
}

/** Answers the service's key, which fakePush's subscription was made with, and keeps what the page subscribes. */
async function serve(page: Page) {
  const subscriptions: unknown[] = [];
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/notify/key', (route) => route.fulfill({ json: { key: 'AQID' } }));
  await page.route('/api/notify/subscriptions', (route) => {
    subscriptions.push(route.request().postDataJSON());
    return route.fulfill({ status: 204 });
  });
  return subscriptions;
}

function bellOf(page: Page): Locator {
  return page.getByRole('banner').getByRole('button', { name: /^Notifications/ });
}

async function panelOf(bell: Locator): Promise<Locator> {
  const id = await bell.getAttribute('popovertarget');
  return bell.page().locator(`[id="${id}"]`);
}

test('the bell turns notifications on from its panel', async ({ page }) => {
  await fakePush(page);
  const subscriptions = await serve(page);
  await page.goto('/');
  const bell = bellOf(page);
  await expect(bell).toHaveAccessibleName('Notifications: Off');

  await bell.click();
  const panel = await panelOf(bell);
  await expect(panel).toBeVisible();
  await expect(panel.getByRole('status')).toHaveText('');
  await panel.getByRole('button', { name: 'Turn on notifications' }).click();

  await expect(bell).toHaveAccessibleName('Notifications: On');
  await expect(panel.getByRole('status')).toHaveText('Notifications on');
  await expect(panel.getByRole('button')).toHaveCount(0);
  await expect(bell).toBeFocused();
  expect(new Set(subscriptions.map((body) => JSON.stringify(body)))).toEqual(
    new Set([JSON.stringify({ endpoint: 'https://push.example/1' })])
  );
});

test('the bell says why a browser gets no notifications', async ({ page }, testInfo) => {
  await serve(page);
  await page.goto('/');
  const bell = bellOf(page);
  // Headless Chromium says the permission is denied, and Playwright's WebKit is a Safari tab with no push manager.
  const [name, status] =
    testInfo.project.name === 'phone'
      ? ['Notifications: Unavailable', 'Notifications need the Home Screen app']
      : ['Notifications: Blocked', 'Notifications are blocked in the browser settings'];
  await expect(bell).toHaveAccessibleName(name);

  await bell.click();

  await expect((await panelOf(bell)).getByRole('status')).toHaveText(status);
});

test('Escape and a tap outside close the panel, and the bell opens it again', async ({ page }) => {
  await serve(page);
  await page.goto('/');
  const bell = bellOf(page);
  const panel = await panelOf(bell);

  await bell.click();
  await expect(panel).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(panel).toBeHidden();

  await bell.click();
  await expect(panel).toBeVisible();
  await page.locator('main').click({ position: { x: 5, y: 5 } });
  await expect(panel).toBeHidden();

  await bell.click();
  await expect(panel).toBeVisible();
});

for (const width of [320, 1280]) {
  test(`the panel hangs under the header, in line with its right end, ${width} px wide`, async ({ page }) => {
    await page.setViewportSize({ width, height: 800 });
    await serve(page);
    await page.goto('/');
    const bell = bellOf(page);
    await bell.click();
    const panel = await panelOf(bell);
    await expect(panel).toBeVisible();

    const banner = page.getByRole('banner');
    const header = await bounds(banner);
    const padding = Number.parseFloat(await banner.evaluate((node) => getComputedStyle(node).paddingRight));
    const box = await bounds(panel);
    expect(box.y).toBeCloseTo(header.y + header.height + 12, 0);
    expect(box.x + box.width).toBeCloseTo(width - padding, 0);
    expect(box.x).toBeGreaterThanOrEqual(padding - 0.5);
    expect(box.width).toBeLessThanOrEqual(360);
    // The status line sits inside the card's 1 px border and 20 px of padding, with no margin of its own.
    const status = await bounds(panel.getByRole('status'));
    expect(status.y - box.y).toBeCloseTo(21, 0);
    expect(status.x - box.x).toBeCloseTo(21, 0);
  });
}

test('the bell, the toggle and Sign out keep to one row at 320 px', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 800 });
  await serve(page);
  await page.goto('/');
  const banner = page.getByRole('banner');
  const signOut = banner.getByRole('button', { name: 'Sign out' });
  const middles: number[] = [];
  for (const button of [bellOf(page), banner.getByRole('button', { name: /^Theme/ }), signOut]) {
    const box = await bounds(button);
    middles.push(Math.round(box.y + box.height / 2));
  }

  expect(new Set(middles).size).toBe(1);
  const lines = await signOut.evaluate((button) => {
    const range = document.createRange();
    range.selectNodeContents(button);
    return new Set([...range.getClientRects()].map((rect) => Math.round(rect.top))).size;
  });
  expect(lines).toBe(1);
  const padding = Number.parseFloat(await banner.evaluate((node) => getComputedStyle(node).paddingRight));
  const end = await bounds(signOut);
  expect(end.x + end.width).toBeCloseTo(320 - padding, 0);
});

test('the panel stays under the header as the page scrolls', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 300 });
  await serve(page);
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 2, name: 'This week' })).toBeVisible();
  const bell = bellOf(page);
  await bell.click();
  const panel = await panelOf(bell);
  await expect(panel).toBeVisible();
  const banner = page.getByRole('banner');
  const top = (await bounds(banner)).y;

  await page.evaluate(() => window.scrollTo(0, 40));

  const header = await bounds(banner);
  expect(header.y).toBeCloseTo(top - 40, 0);
  expect((await bounds(panel)).y).toBeCloseTo(header.y + header.height + 12, 0);
});
