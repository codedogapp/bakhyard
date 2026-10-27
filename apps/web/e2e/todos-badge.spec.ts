import type { Page } from '@playwright/test';

import type { Todo } from '../src/todos/api';
import { bounds } from './bounds';
import { home, tickets } from './demo';
import { expect, test } from './fixtures';
import { group } from './groups';
import { answerTickets, aTodo, chipLooks } from './jira';

/** Made-up todos: two name open tickets of the demo's snapshot, one a ticket that isn't open, one none. */
const linked: Todo[] = [
  { id: 4, title: 'Ship the date picker fix', done: false, priority: 'high', ticket: 'WEB-9' },
  { id: 3, title: 'Plan the certificate rotation', done: false, priority: 'normal', ticket: 'OPS-31' },
  { id: 2, title: 'Archive the old export', done: false, priority: 'normal', ticket: 'WEB-4' },
  { id: 1, title: 'Buy milk', done: false, priority: 'normal' },
];
const [picker] = linked;

/** The checkbox whose name is exactly the name. */
function box(page: Page, name: string) {
  return page.getByRole('checkbox', { name, exact: true });
}

/** The row of the todo with the title. */
function row(page: Page, title: string) {
  return page.getByRole('listitem').filter({ has: page.getByRole('checkbox', { name: title }) });
}

/** What makes text look like the list rows' mono label. */
function labelLook(node: Element) {
  const style = getComputedStyle(node);
  return [style.color, style.fontFamily, style.fontSize, style.fontWeight, style.letterSpacing];
}

/** The picker's todo alone, linked to my one open ticket, with the key and status. */
async function oneTicket(page: Page, key: string, status: string) {
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: [{ ...picker, ticket: key }] } }));
  await answerTickets(page, {
    tickets: [{ key, summary: 'Approve', status, category: 'to-do', url: `https://jira.example.com/browse/${key}` }],
    searched: '2026-10-21T00:38:00Z',
  });
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: linked } }));
});

test("a linked todo's row shows the key under its title, with the status of an open ticket", async ({ page }) => {
  await answerTickets(page, tickets);
  await page.goto('/todos');

  await expect(box(page, 'Ship the date picker fix WEB-9 In Progress')).toBeVisible();
  await expect(box(page, 'Plan the certificate rotation OPS-31 To Do')).toBeVisible();
  await expect(box(page, 'Archive the old export WEB-4')).toBeVisible();
  await expect(box(page, 'Buy milk')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Edit Ship the date picker fix', exact: true })).toBeVisible();
  const ship = row(page, picker.title);
  const [title, key, chip] = await Promise.all(
    [ship.getByText(picker.title), ship.getByText('WEB-9', { exact: true }), ship.locator('.chip')].map(bounds)
  );
  // The key 4 px under the title and at its x, the chip 10 px after the key and centred on its line.
  expect(key.y - (title.y + title.height)).toBeCloseTo(4, 0);
  expect(key.x).toBeCloseTo(title.x, 0);
  expect(chip.x - (key.x + key.width)).toBeCloseTo(10, 0);
  expect(chip.y + chip.height / 2).toBeCloseTo(key.y + key.height / 2, 0);
  // The key looks like the Jira section's: the list rows' mono label.
  const jiraKey = group(page, 'Jira tickets').getByText('WEB-9', { exact: true });
  expect(await ship.getByText('WEB-9', { exact: true }).evaluate(labelLook)).toEqual(await jiraKey.evaluate(labelLook));
  expect(await chipLooks(group(page, 'High priority').locator('.chip'))).toEqual(['filled']);
  expect(await chipLooks(group(page, 'Normal priority').locator('.chip'))).toEqual(['outlined']);
});

for (const [what, snapshot, sign] of [
  ['Jira is off', 404, undefined],
  ["the tickets don't load", 500, "Couldn't load the tickets."],
  ['no search has succeeded', { failure: 'search Jira: 401 Unauthorized' }, /^The last search failed/],
] as const) {
  test(`a linked todo's row shows the key alone when ${what}`, async ({ page }) => {
    await answerTickets(page, snapshot);
    const answered = page.waitForResponse('/api/todos/tickets');
    await page.goto('/todos');
    await answered;
    if (sign) {
      await expect(group(page, 'Jira tickets').getByText(sign)).toBeVisible();
    }

    await expect(box(page, 'Ship the date picker fix WEB-9')).toBeVisible();
    await expect(box(page, 'Plan the certificate rotation OPS-31')).toBeVisible();
    await expect(page.locator('main ul.card .chip')).toHaveCount(0);
  });
}

test('a click on the badge ticks the todo', async ({ page }) => {
  await answerTickets(page, tickets);
  const sent: string[] = [];
  await page.route(aTodo, async (route) => {
    sent.push(`${route.request().method()} ${new URL(route.request().url()).pathname} ${route.request().postData()}`);
    await route.fulfill({ json: { ...picker, done: true } });
  });
  await page.goto('/todos');

  await row(page, picker.title).getByText('In Progress').click();

  await expect(box(page, 'Ship the date picker fix WEB-9 In Progress')).toBeChecked();
  await expect.poll(() => sent).toEqual(['PATCH /api/todos/4 {"done":true}']);
});

test("a done linked todo's title is struck through, but not its ticket", async ({ page }) => {
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: [{ ...picker, done: true }] } }));
  await answerTickets(page, tickets);
  await page.goto('/todos');
  const done = row(page, picker.title);

  await expect(done.getByText(picker.title)).toHaveCSS('text-decoration-line', 'line-through');
  await expect(done.locator('.chip')).toHaveText('In Progress');
  // A box's decoration reaches the text inside it without showing in that text's computed style, so look at every box
  // from the key and the chip up to the row.
  const lines = await done.locator('.badge span').evaluateAll((spans) => {
    const boxes = new Set<Element>();
    for (const span of spans) {
      for (let node: Element | null = span; node && !boxes.has(node); node = node.parentElement) {
        boxes.add(node);
        if (node.tagName === 'LI') {
          break;
        }
      }
    }
    return [...boxes].map((node) => `${node.tagName} ${getComputedStyle(node).textDecorationLine}`);
  });
  expect(lines).toEqual(['SPAN none', 'SPAN none', 'SPAN none', 'LABEL none', 'LI none', 'SPAN none']);
});

test('a long key and status wrap on a narrow screen', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  const key = 'ABCDEFGHIJ-1234567';
  const status = 'Waiting for the customer to approve the change';
  await oneTicket(page, key, status);
  await page.goto('/todos');

  await expect(box(page, `${picker.title} ${key} ${status}`)).toBeVisible();
  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
  const ship = row(page, picker.title);
  const [label, keyBox, chip] = await Promise.all(
    [ship.locator('label'), ship.getByText(key, { exact: true }), ship.locator('.chip')].map(bounds)
  );
  expect(chip.x + chip.width).toBeLessThanOrEqual(label.x + label.width);
  // The chip takes the line under the key, 4 px down, rather than squeeze beside it.
  expect(chip.y - (keyBox.y + keyBox.height)).toBeCloseTo(4, 0);
  expect(chip.x).toBeCloseTo(keyBox.x, 0);
});

test('a long status breaks inside chips at most 160 px wide, with the key centred beside it', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  // One word with no hyphen, which a browser would break after.
  const status = 'Wartungsfreigabeausstehend';
  await oneTicket(page, 'OPS-77', status);
  await page.goto('/todos');

  await expect(box(page, `${picker.title} OPS-77 ${status}`)).toBeVisible();
  // The badge's chip and the Jira section's.
  const chips = page.locator('.chip');
  await expect(chips).toHaveCount(2);
  for (const each of await chips.all()) {
    const size = await bounds(each);
    expect(size.width).toBeLessThanOrEqual(160);
    expect(size.height).toBeGreaterThan(20);
    expect(await each.evaluate((node) => node.scrollWidth - node.clientWidth)).toBe(0);
  }
  const ship = row(page, picker.title);
  const [key, chip] = await Promise.all([ship.getByText('OPS-77', { exact: true }), ship.locator('.chip')].map(bounds));
  // The badge's line grows with its chip; the key keeps its height, centred on the line.
  expect(key.height).toBeCloseTo(20, 0);
  expect(key.y + key.height / 2).toBeCloseTo(chip.y + chip.height / 2, 0);
});
