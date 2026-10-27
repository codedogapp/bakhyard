import type { Locator, Page } from '@playwright/test';

import { bounds } from './bounds';
import { home, tickets, todos } from './demo';
import { expect, test } from './fixtures';
import { group } from './groups';
import { answerDismissals } from './jira';

/** The computed values of the properties, in their order, on what the locator finds. */
function styles(locator: Locator, properties: string[]) {
  return locator.evaluate(
    (node, names) => names.map((name) => getComputedStyle(node).getPropertyValue(name)),
    properties
  );
}

/** The inbox row of the ticket. */
function row(page: Page, key: string) {
  return group(page, 'Jira inbox').getByRole('listitem').filter({ hasText: key });
}

/** What an Add button shares with the form's. */
const button = [
  'min-height',
  'padding-top',
  'padding-left',
  'border-top-width',
  'border-top-style',
  'border-top-color',
  'background-color',
  'color',
  'font-family',
  'font-size',
  'letter-spacing',
  'text-transform',
];

/** What a row's status line shares with the form's. */
const status = ['color', 'font-family', 'font-size', 'font-weight', 'line-height', 'letter-spacing', 'text-transform'];

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/todos/tickets', (route) => route.fulfill({ json: tickets }));
  await answerDismissals(page);
  await page.route('/api/todos', (route) =>
    route.request().method() === 'GET'
      ? route.fulfill({ json: { todos } })
      : route.fulfill({ status: 400, body: 'invalid title: over 500 characters\n' })
  );
  await page.goto('/todos');
});

test("a row's Add, Dismiss and status line look as the form's do, the line 4 px under the summary", async ({
  page,
}) => {
  const web12 = row(page, 'WEB-12');
  const form = page.locator('main form').first();
  const theirs = await styles(form.getByRole('button', { name: 'Add' }), button);
  for (const name of ['Add WEB-12', 'Dismiss WEB-12']) {
    expect(await styles(web12.getByRole('button', { name }), button)).toEqual(theirs);
  }
  // An empty line takes no room: the summary alone sets the text's height.
  const [text, title] = await Promise.all([web12.locator('.list-text'), web12.locator('.list-title')].map(bounds));
  expect(text.height).toBe(title.height);

  await web12.getByRole('button', { name: 'Add WEB-12' }).click();
  await page.mouse.move(0, 0);
  const line = web12.getByRole('status');
  await expect(line).toHaveText('invalid title: over 500 characters');
  expect(await styles(line, status)).toEqual(await styles(form.getByRole('status'), status));
  const [summary, below] = await Promise.all([web12.getByText('Show the order history'), line].map(bounds));
  expect(below.y - (summary.y + summary.height)).toBeCloseTo(4, 0);
});

test("the inbox's card and rows look as the Jira section's do, and line up with them", async ({ page }) => {
  const card = ['border-top-width', 'border-top-style', 'border-top-color', 'background-color', 'box-shadow'];
  const jira = group(page, 'Jira tickets');
  expect(await styles(group(page, 'Jira inbox').getByRole('list'), card)).toEqual(
    await styles(jira.locator('.card'), card)
  );
  const theirRow = jira.getByRole('listitem').filter({ hasText: 'WEB-12' });
  const [ours, theirs] = await Promise.all([row(page, 'WEB-12'), theirRow].map(bounds));
  expect([ours.x, ours.width]).toEqual([theirs.x, theirs.width]);

  const text = ['color', 'font-family', 'font-size', 'font-weight', 'letter-spacing', 'text-transform'];
  for (const part of ['WEB-12', 'Show the order history']) {
    expect(await styles(row(page, 'WEB-12').getByText(part), text)).toEqual(
      await styles(theirRow.getByText(part), text)
    );
  }
});

test('with no todos, the inbox sits under the form as a group would', async ({ page }) => {
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: [] } }));
  await page.goto('/todos');

  const [form, inbox] = await Promise.all([page.locator('main form').first(), group(page, 'Jira inbox')].map(bounds));
  const phone = (page.viewportSize()?.width ?? 0) <= 700;
  expect(inbox.y - (form.y + form.height)).toBeCloseTo(phone ? 24 : 30, 0);
});

test('an inbox row keeps its background under the pointer, as only a row that is a link fills', async ({
  page,
}, testInfo) => {
  // A phone has no pointer to hover with.
  test.skip(testInfo.project.name === 'phone');
  const web12 = row(page, 'WEB-12');
  const before = await styles(web12, ['background-color']);

  await web12.getByText('WEB-12').hover();
  expect(await styles(web12, ['background-color'])).toEqual(before);

  const link = group(page, 'Jira tickets').getByRole('link', { name: /^WEB-12 / });
  await link.hover();
  await expect(link).not.toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
});

test("on a wide screen, a row's Add and Dismiss follow its status, 20 px after it and 10 px apart", async ({
  page,
}, testInfo) => {
  // A phone puts the buttons on a line of their own, as the tests below check.
  test.skip(testInfo.project.name === 'phone');
  const web12 = row(page, 'WEB-12');
  const [chip, add, dismiss, card] = await Promise.all(
    [
      web12.locator('.chip'),
      web12.getByRole('button', { name: 'Add WEB-12' }),
      web12.getByRole('button', { name: 'Dismiss WEB-12' }),
      group(page, 'Jira inbox').getByRole('list'),
    ].map(bounds)
  );
  for (const each of [add, dismiss]) {
    expect(each.y + each.height / 2).toBeCloseTo(chip.y + chip.height / 2, 0);
  }
  expect(add.x - (chip.x + chip.width)).toBeCloseTo(20, 0);
  expect(dismiss.x - (add.x + add.width)).toBeCloseTo(10, 0);
  expect(card.x + card.width - (dismiss.x + dismiss.width)).toBeCloseTo(21, 0);
});

for (const width of [390, 320]) {
  test(`at ${width} px, a row's key and status share its first line, then its summary, then its buttons`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 800 });

    const web12 = row(page, 'WEB-12');
    const [key, chip, summary, add, dismiss, card, listed] = await Promise.all(
      [
        web12.getByText('WEB-12'),
        web12.locator('.chip'),
        web12.getByText('Show the order history'),
        web12.getByRole('button', { name: 'Add WEB-12' }),
        web12.getByRole('button', { name: 'Dismiss WEB-12' }),
        group(page, 'Jira inbox').getByRole('list'),
        group(page, 'Jira tickets')
          .getByRole('link', { name: /^WEB-12 / })
          .locator('.chip'),
      ].map(bounds)
    );
    expect(chip.y + chip.height / 2).toBeCloseTo(key.y + key.height / 2, 0);
    // The buttons, on a line of their own, leave the chip as wide as the Jira section's, on one line.
    expect([chip.width, chip.height]).toEqual([listed.width, listed.height]);
    expect(summary.y).toBeGreaterThanOrEqual(Math.max(key.y + key.height, chip.y + chip.height));
    expect(add.y).toBeGreaterThanOrEqual(summary.y + summary.height);
    expect(add.x).toBeCloseTo(key.x, 0);
    expect(dismiss.y).toBeCloseTo(add.y, 0);
    expect(dismiss.x - (add.x + add.width)).toBeCloseTo(10, 0);
    expect(chip.x + chip.width).toBeLessThanOrEqual(card.x + card.width);
    const scroll = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth
    );
    expect(scroll).toBe(0);
  });
}
