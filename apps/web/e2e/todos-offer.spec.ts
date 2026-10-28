import type { Page } from '@playwright/test';

import type { Todo } from '../src/todos/api';
import type { Snapshot } from '../src/todos/tickets';
import { bounds } from './bounds';
import { chipLooks } from './chips';
import { home, tickets } from './demo';
import { expect, test } from './fixtures';
import { answerDismissals, answerTickets, aTodo } from './jira';

/** Made-up todos: one names a ticket that's done in Jira, one an open ticket of the demo's snapshot. */
const notes: Todo = { id: 2, title: 'Send the release notes', done: false, priority: 'normal', ticket: 'WEB-3' };
const picker: Todo = { id: 1, title: 'Ship the date picker fix', done: false, priority: 'normal', ticket: 'WEB-9' };

/** The demo's open tickets, and the done ticket that the notes' todo names. */
const withDone: Snapshot = {
  ...tickets,
  done: [
    {
      key: 'WEB-3',
      summary: 'Write the release notes for 2.4',
      status: 'Closed',
      category: 'done',
      url: 'https://jira.example.com/browse/WEB-3',
    },
  ],
};

const box = (page: Page, name: string) => page.getByRole('checkbox', { name, exact: true });
const row = (page: Page, title: string) =>
  page.getByRole('listitem').filter({ has: page.getByRole('checkbox', { name: title }) });
const offers = (page: Page) => page.getByText('Done in Jira', { exact: true });

/** Answers GET /api/todos with the todos and each PATCH with its todo, or with the status; gives the PATCHes' bodies. */
async function serveTodos(page: Page, todos: Todo[], status?: number) {
  const sent: string[] = [];
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos } }));
  await page.route(aTodo, async (route) => {
    const request = route.request();
    sent.push(`${request.method()} ${new URL(request.url()).pathname} ${request.postData()}`);
    const todo = todos.find(({ id }) => request.url().endsWith(`/${id}`));
    await route.fulfill(status ? { status } : { json: { ...todo, ...request.postDataJSON() } });
  });
  return sent;
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await answerDismissals(page);
  await answerTickets(page, withDone);
});

test("a todo whose ticket is done shows the ticket's status and offers to tick the todo", async ({ page }) => {
  await serveTodos(page, [notes, picker]);
  await page.goto('/todos');

  await expect(box(page, 'Send the release notes WEB-3 Closed Done in Jira')).not.toBeChecked();
  await expect(box(page, 'Ship the date picker fix WEB-9 In Progress')).toBeVisible();
  await expect(offers(page)).toHaveCount(1);
  expect(await chipLooks(row(page, notes.title).locator('.chip'))).toEqual(['outlined']);
  // A done ticket shows only in its todo's badge: not in the Jira section, the inbox or the editor's choices.
  await expect(page.getByRole('main').getByText('WEB-3', { exact: true })).toHaveCount(1);
  await page.getByRole('button', { name: `Edit ${notes.title}`, exact: true }).click();
  await expect(page.getByRole('combobox', { name: 'Ticket' }).locator('option')).toHaveText([
    'No ticket',
    'WEB-3',
    'WEB-12 · Show the order history on the account page',
    'WEB-9 · Fix the date picker in Safari',
    'OPS-31 · Rotate the staging certificates',
  ]);
});

test('the offer looks like the key, but in the ink and in capitals, 10 px after the status', async ({ page }) => {
  await serveTodos(page, [notes, picker]);
  await page.goto('/todos');
  await page.mouse.move(0, 0);
  const done = row(page, notes.title);
  const [offer, key, chip] = [offers(page), done.getByText('WEB-3', { exact: true }), done.locator('.chip')];
  await expect(offer).toBeVisible();

  const look = (node: Element) => {
    const { color, fontFamily, fontSize, lineHeight, fontWeight, letterSpacing, textTransform } =
      getComputedStyle(node);
    return { color, font: [fontFamily, fontSize, lineHeight, fontWeight, letterSpacing], textTransform };
  };
  const [offerLook, keyLook, chipLook] = await Promise.all([offer, key, chip].map((each) => each.evaluate(look)));
  expect(offerLook.font).toEqual(keyLook.font);
  expect(offerLook.color).toBe(chipLook.color);
  expect(offerLook.color).not.toBe(keyLook.color);
  expect([offerLook.textTransform, keyLook.textTransform]).toEqual(['uppercase', 'none']);
  const [offerBox, chipBox] = await Promise.all([offer, chip].map(bounds));
  expect(offerBox.x - (chipBox.x + chipBox.width)).toBeCloseTo(10, 0);
  expect(offerBox.y + offerBox.height / 2).toBeCloseTo(chipBox.y + chipBox.height / 2, 0);
});

test('a tap on the offer ticks the todo, and the offer goes until the todo is unticked', async ({ page }) => {
  const sent = await serveTodos(page, [notes, picker]);
  await page.goto('/todos');

  await offers(page).click();

  await expect(box(page, 'Send the release notes WEB-3 Closed')).toBeChecked();
  await expect(offers(page)).toHaveCount(0);
  await expect.poll(() => sent).toEqual(['PATCH /api/todos/2 {"done":true}']);
  await box(page, 'Send the release notes WEB-3 Closed').click();
  await expect(box(page, 'Send the release notes WEB-3 Closed Done in Jira')).not.toBeChecked();
  await expect.poll(() => sent).toEqual(['PATCH /api/todos/2 {"done":true}', 'PATCH /api/todos/2 {"done":false}']);
});

test('a failed tick brings the offer back', async ({ page }) => {
  await serveTodos(page, [notes, picker], 500);
  await page.goto('/todos');

  await offers(page).click();

  await expect(row(page, notes.title).getByRole('status')).toHaveText('Tick failed');
  await expect(box(page, 'Send the release notes WEB-3 Closed Done in Jira')).not.toBeChecked();
});

test('a ticked todo whose ticket is done shows the status alone', async ({ page }) => {
  await serveTodos(page, [{ ...notes, done: true }, picker]);
  await page.goto('/todos');

  await expect(box(page, 'Send the release notes WEB-3 Closed')).toBeChecked();
  await expect(offers(page)).toHaveCount(0);
});

test('on a narrow screen, the offer takes the line under the key rather than squeeze beside the status', async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 640 });
  await serveTodos(page, [notes, picker]);
  await page.goto('/todos');
  const done = row(page, notes.title);
  await expect(offers(page)).toBeVisible();

  const scroll = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(scroll).toBe(0);
  const [label, key, chip, offer] = await Promise.all(
    [done.locator('label'), done.getByText('WEB-3', { exact: true }), done.locator('.chip'), offers(page)].map(bounds)
  );
  expect(offer.y - (chip.y + chip.height)).toBeCloseTo(4, 0);
  expect(offer.x).toBeCloseTo(key.x, 0);
  expect(offer.x + offer.width).toBeLessThanOrEqual(label.x + label.width);
});
