import { alerts } from './alerts';
import { bounds } from './bounds';
import { home } from './demo';
import { expect, test } from './fixtures';
import { group } from './groups';

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test('the Alerts page lists the alerts that fire, then those of the last 30 days', async ({ page }, testInfo) => {
  await page.route('/api/notify/alerts', (route) => route.fulfill({ json: { alerts } }));
  await page.goto('/alerts');

  await expect(page).toHaveTitle('Alerts · Bakhyard');
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Alerts');
  // The tab bar, the phone layout, has no tab for the Alerts.
  const [name, current] = testInfo.project.name === 'phone' ? ['More', 'true'] : ['Alerts', 'page'];
  await expect(page.getByRole('link', { name, exact: true })).toHaveAttribute('aria-current', current);
  const firing = group(page, 'Firing now');
  await expect(firing.getByText('2 alerts')).toBeVisible();
  await expect(firing.getByRole('listitem')).toHaveText([
    'Today The laptop runs on battery No charger has powered the laptop for 2 minutes. Since 09:12',
    'Yesterday A filesystem is nearly full /dev/disk3s1 is 91.2% full. Since 22:30',
  ]);
  await expect(firing.getByText('Today')).toHaveAttribute('datetime', '2026-10-21T00:12:00.512304Z');
  const past = group(page, 'The last 30 days');
  await expect(past.getByText('3 alerts')).toBeVisible();
  await expect(past.getByRole('listitem')).toHaveText([
    "Yesterday The laptop's RAM is nearly full 93.4% of the RAM is in use. 14:05–14:20",
    '17 Oct Flux fails to reconcile Kustomization flux-system/apps has not been ready for 15 minutes. 23:45 – 18 Oct, 00:20',
    '2 Oct The CPU is too hot The CPU is at 92 °C. 15:00–15:15',
  ]);
});

// 981 px is the narrowest width with the header's nav, and 1101 px the narrowest with its links 32 px apart.
for (const width of [981, 1101]) {
  test(`the header's nav fits its ten links within the page's padding, ${width} px wide`, async ({ page }) => {
    await page.setViewportSize({ width, height: 800 });
    await page.route('/api/notify/alerts', (route) => route.fulfill({ json: { alerts } }));
    await page.goto('/alerts');
    const banner = page.getByRole('banner');
    await expect(banner.getByRole('navigation', { name: 'Apps' }).getByRole('link')).toHaveCount(10);

    const scroll = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth
    );
    expect(scroll).toBe(0);
    const padding = Number.parseFloat(await banner.evaluate((node) => getComputedStyle(node).paddingRight));
    const end = await bounds(banner.getByRole('button', { name: 'Sign out' }));
    expect(end.x + end.width).toBeCloseTo(width - padding, 0);
  });
}

test('the Alerts page says when no alert fires, nor fired in the last 30 days', async ({ page }) => {
  await page.route('/api/notify/alerts', (route) => route.fulfill({ json: { alerts: [] } }));
  await page.goto('/alerts');

  await expect(group(page, 'Firing now').getByText('No alerts are firing.')).toBeVisible();
  await expect(group(page, 'The last 30 days').getByText('No alerts in the last 30 days.')).toBeVisible();
});
