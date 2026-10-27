import type { Locator } from '@playwright/test';

import type { Todo } from '../src/todos/api';
import { bounds } from './bounds';
import { home, tickets } from './demo';
import { expect, test } from './fixtures';
import { answerDismissals, answerTickets } from './jira';

const milk: Todo = { id: 1, title: 'Buy milk', done: false, priority: 'normal' };

/** What the ticket select shares with the title field. */
const shared = [
  'appearance',
  'height',
  'border-top-width',
  'border-top-style',
  'border-top-color',
  'border-radius',
  'background-color',
  'color',
  'font-family',
  'font-size',
  'font-weight',
  'line-height',
  'padding-top',
  'padding-bottom',
  'padding-left',
];

/** The computed values of the properties, in their order, on what the locator finds. */
function styles(locator: Locator, properties: string[]) {
  return locator.evaluate(
    (node, names) => names.map((name) => getComputedStyle(node).getPropertyValue(name)),
    properties
  );
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
  await page.route('/api/todos', (route) => route.fulfill({ json: { todos: [milk] } }));
  await answerTickets(page, tickets);
  await answerDismissals(page);
  await page.goto('/todos');
  await page.getByRole('button', { name: `Edit ${milk.title}` }).click();
  // Away from the field the click opened under the pointer, which would show its hover border.
  await page.mouse.move(0, 0);
});

test('the ticket select looks as the title field does, with room on its right for the chevron', async ({ page }) => {
  const select = page.getByRole('combobox', { name: 'Ticket' });

  expect(await styles(select, shared)).toEqual(await styles(page.getByRole('textbox', { name: 'Title' }), shared));
  expect(await styles(select, ['padding-right', 'text-overflow', 'cursor'])).toEqual(['44px', 'ellipsis', 'pointer']);
});

test("the chevron points down 14 px inside the select's right edge, and a click on it reaches the select", async ({
  page,
}) => {
  const select = page.getByRole('combobox', { name: 'Ticket' });
  const chevron = page.locator('form .ticket svg');
  const [around, mark] = await Promise.all([select, chevron].map(bounds));

  expect([mark.width, mark.height]).toEqual([16, 16]);
  expect(around.x + around.width - (mark.x + mark.width)).toBeCloseTo(14, 1);
  expect(mark.y + mark.height / 2).toBeCloseTo(around.y + around.height / 2, 1);
  expect(await styles(chevron, ['fill', 'stroke-width', 'stroke-linecap'])).toEqual(['none', '1.5px', 'square']);
  // 8 units wide and 4 tall from (4, 6): a head pointing down.
  const shape = await chevron.locator('path').evaluate((path: SVGPathElement) => {
    const { x, y, width, height } = path.getBBox();
    return [x, y, width, height];
  });
  expect(shape).toEqual([4, 6, 8, 4]);
  const centre = { x: mark.x + mark.width / 2, y: mark.y + mark.height / 2 };
  expect(await page.evaluate(({ x, y }) => document.elementFromPoint(x, y)?.localName, centre)).toBe('select');
});

test("the ticket follows the priority on the field's line while it has room, and has a line of its own on a phone", async ({
  page,
}) => {
  const controls = [
    page.getByRole('textbox', { name: 'Title' }),
    page.getByRole('group', { name: 'Priority' }),
    page.getByRole('combobox', { name: 'Ticket' }),
    page.getByRole('button', { name: 'Save' }),
  ];
  const bottom = ({ y, height }: { y: number; height: number }) => y + height;

  for (const [width, place] of [
    [320, 'alone'],
    [390, 'alone'],
    [768, 'under the field'],
    [1024, 'beside the priority'],
    [1440, 'beside the priority'],
  ] as const) {
    await page.setViewportSize({ width, height: 800 });
    const [field, priority, select, save] = await Promise.all(controls.map(bounds));
    const at = `at ${width} px`;
    expect(select.height, `height ${at}`).toBe(field.height);
    if (place === 'beside the priority') {
      expect([select.y, select.x], `place ${at}`).toEqual([field.y, priority.x + priority.width + 12]);
    } else if (place === 'under the field') {
      expect([select.y, save.y], `place ${at}`).toEqual([bottom(field) + 12, bottom(field) + 12]);
    } else {
      expect([select.y, save.y], `place ${at}`).toEqual([bottom(priority) + 12, bottom(select) + 12]);
      expect(select.x + select.width, `width ${at}`).toBeCloseTo(field.x + field.width, 0);
    }
    const scroll = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth
    );
    expect(scroll, `sideways scroll ${at}`).toBe(0);
  }
});
