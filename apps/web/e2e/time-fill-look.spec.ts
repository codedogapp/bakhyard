import type { Locator, Page } from '@playwright/test';

import type { Snapshot } from '../src/jira/tickets';
import type { Report } from '../src/time/report';
import { bounds } from './bounds';
import { home, report, tickets } from './demo';
import { expect, test } from './fixtures';
import { answerTickets } from './jira';

/** The computed values of the properties, in their order, on what the locator finds. */
function styles(locator: Locator, properties: string[]) {
  return locator.evaluate(
    (node, names) => names.map((name) => getComputedStyle(node).getPropertyValue(name)),
    properties
  );
}

/** The colour that the custom property holds, as the browser computes a colour. */
function colour(page: Page, property: string) {
  return page.evaluate((name) => {
    const probe = document.createElement('span');
    probe.style.color = `var(${name})`;
    document.body.append(probe);
    const computed = getComputedStyle(probe).color;
    probe.remove();
    return computed;
  }, property);
}

// The demo report's hours, on a week whose first day has the longest name: "Wednesday 30 Sept".
const september: Report = {
  ...report,
  days: report.days?.map((day, index) => ({ ...day, date: `2026-${index ? `10-0${index}` : '09-30'}` })),
};

const names = ['Day', 'Ticket', 'Hours'];
const select = (page: Page, name: string) => page.getByRole('main').getByRole('combobox', { name });
const chevron = (page: Page, name: string) =>
  page
    .locator('main .form-select')
    .filter({ has: page.getByRole('combobox', { name }) })
    .locator('svg');
// The form's Log, not the dialog's, which is in the page's main too.
const logButton = (page: Page) => page.locator('main form').getByRole('button', { name: 'Log' });
const thisWeek = (page: Page) =>
  page.locator('main section').filter({ has: page.getByRole('heading', { name: 'This week' }) });
const bottom = ({ y, height }: { y: number; height: number }) => y + height;

/** Opens the Time page on the September week, with my open tickets as given, the pointer away from the form. */
async function open(page: Page, snapshot: Snapshot | number = tickets) {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/time/report', (route) => route.fulfill({ json: september }));
  await answerTickets(page, snapshot);
  await page.goto('/time');
  await expect(page.getByRole('main').getByRole('listitem')).toHaveCount(7);
  await page.mouse.move(0, 0);
}

/** How much wider the select's widest option is than the room for its text, in px. */
function overflow(locator: Locator) {
  return locator.evaluate((node: HTMLSelectElement) => {
    const style = getComputedStyle(node);
    const canvas = document.createElement('canvas').getContext('2d');
    if (!canvas) {
      throw new Error('no canvas');
    }
    canvas.font = `${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
    const room = node.clientWidth - Number.parseFloat(style.paddingLeft) - Number.parseFloat(style.paddingRight);
    return Math.max(...[...node.options].map((option) => canvas.measureText(option.text).width)) - room;
  });
}

test('the three selects look alike, as the form select does, with room on their right for the chevron', async ({
  page,
}) => {
  await open(page);
  const shared = [
    'height',
    'border-top-width',
    'border-top-style',
    'border-top-color',
    'background-color',
    'color',
    'font-family',
    'font-size',
    'line-height',
    'padding-left',
    'padding-right',
  ];

  const [day, ...others] = await Promise.all(names.map((name) => styles(select(page, name), shared)));
  expect(others).toEqual([day, day]);
  expect(day.slice(-2)).toEqual(['14px', '44px']);
  expect(await styles(select(page, 'Ticket'), ['text-overflow', 'cursor'])).toEqual(['ellipsis', 'pointer']);
});

test("each chevron points down 14 px inside its select's right edge, and a click on it reaches the select", async ({
  page,
}) => {
  await open(page);
  for (const name of names) {
    const [around, mark] = await Promise.all([select(page, name), chevron(page, name)].map(bounds));
    expect([mark.width, mark.height], name).toEqual([16, 16]);
    expect(around.x + around.width - (mark.x + mark.width), name).toBeCloseTo(14, 1);
    expect(mark.y + mark.height / 2, name).toBeCloseTo(around.y + around.height / 2, 1);
    const centre = { x: mark.x + mark.width / 2, y: mark.y + mark.height / 2 };
    const found = await page.evaluate(
      ({ x, y }) => document.elementFromPoint(x, y)?.getAttribute('aria-label'),
      centre
    );
    expect(found, name).toBe(name);
  }
});

test('the form sits 30 px over This week, 24 px on a phone, its lines 12 px apart', async ({ page }) => {
  await open(page);
  const form = page.locator('main form');

  await page.setViewportSize({ width: 1024, height: 800 });
  expect((await bounds(thisWeek(page))).y - bottom(await bounds(form))).toBeCloseTo(30, 0);

  await page.setViewportSize({ width: 390, height: 800 });
  expect((await bounds(thisWeek(page))).y - bottom(await bounds(form))).toBeCloseTo(24, 0);
  const [day, ticket, hours] = await Promise.all(names.map((name) => bounds(select(page, name))));
  expect([ticket.y - bottom(day), hours.y - bottom(ticket)]).toEqual([12, 12]);
});

test('the form takes one line on a wide screen, two on a narrow one and three on a phone', async ({ page }) => {
  await open(page);
  const controls = [...names.map((name) => select(page, name)), logButton(page)];

  for (const [width, lines] of [
    [1024, [0, 0, 0, 0]],
    [600, [0, 0, 1, 1]],
    [390, [0, 1, 2, 2]],
  ] as const) {
    await page.setViewportSize({ width, height: 800 });
    const tops = (await Promise.all(controls.map(bounds))).map(({ y }) => y);
    const order = [...new Set(tops)];
    expect(
      tops.map((top) => order.indexOf(top)),
      `at ${width} px`
    ).toEqual(lines);
  }
});

test('the ticket takes three parts in four of the room it shares with the day, the day the fourth', async ({
  page,
}) => {
  await open(page);
  const fields = ['Day', 'Ticket'].map((name) => select(page, name).locator('..'));
  const widths = () => Promise.all(fields.map(async (field) => (await bounds(field)).width));
  const grow = (value: string) =>
    Promise.all(
      fields.map((field) =>
        field.evaluate((node: HTMLElement, to) => {
          node.style.flexGrow = to;
        }, value)
      )
    );

  for (const width of [1024, 600]) {
    await page.setViewportSize({ width, height: 800 });
    const [day = 0, ticket = 0] = await widths();
    // Without growing, each is as wide as its basis; what it grew by is its part of the room.
    await grow('0');
    const [dayBasis = 0, ticketBasis = 0] = await widths();
    await grow('');
    expect(day - dayBasis, `at ${width} px`).toBeGreaterThan(0);
    expect(ticket - ticketBasis, `at ${width} px`).toBeCloseTo(3 * (day - dayBasis), 0);
  }
});

test('from 320 to 1440 px, days and hours show whole, Log stays by them, no sideways scroll', async ({ page }) => {
  // 113 widths take longer than the default limit while other sessions run their checks.
  test.slow();
  await open(page);
  for (let width = 320; width <= 1440; width += 10) {
    await page.setViewportSize({ width, height: 800 });
    const at = `at ${width} px`;
    expect(await overflow(select(page, 'Day')), `day ${at}`).toBeLessThanOrEqual(1);
    expect(await overflow(select(page, 'Hours')), `hours ${at}`).toBeLessThanOrEqual(1);
    const [hours, log] = await Promise.all([select(page, 'Hours'), logButton(page)].map(bounds));
    expect(log.y, `Log ${at}`).toBe(hours.y);
    expect(log.x - (hours.x + hours.width), `Log's gap ${at}`).toBeCloseTo(12, 1);
    const scroll = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth
    );
    expect(scroll, `sideways scroll ${at}`).toBe(0);
  }
});

test("the dialog's Log is paper on ink, its Cancel as the form's buttons are", async ({ page }) => {
  await open(page);
  await select(page, 'Ticket').selectOption('WEB-9');
  await logButton(page).click();
  const dialog = page.getByRole('alertdialog', { name: 'Log these hours?' });
  await expect(dialog).toBeVisible();
  await page.mouse.move(0, 0);
  const paint = ['background-color', 'color'];

  expect(await styles(dialog.getByRole('button', { name: 'Log' }), paint)).toEqual([
    await colour(page, '--ink'),
    await colour(page, '--paper'),
  ]);
  expect(await styles(dialog.getByRole('button', { name: 'Cancel' }), paint)).toEqual(
    await styles(logButton(page), paint)
  );
});

for (const [width, gap] of [
  [1024, 30],
  [390, 24],
] as const) {
  test(`at ${width} px, a line in the form's place is muted, ${gap} px over This week`, async ({ page }) => {
    await page.setViewportSize({ width, height: 800 });
    await open(page, { ...tickets, tickets: [] });
    const note = page.getByRole('main').getByText('No open Jira tickets to log hours on.');

    expect(await styles(note, ['color'])).toEqual([await colour(page, '--ink-muted')]);
    expect((await bounds(thisWeek(page))).y - bottom(await bounds(note))).toBeCloseTo(gap, 0);
  });
}
