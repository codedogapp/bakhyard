import type { Page } from '@playwright/test';

import type { Plan } from '../src/planner/api';
import { home, plans } from './demo';
import { expect, test } from './fixtures';
import { group } from './groups';
import { conversation, serveConversation } from './plan';

const [newest, ...older] = plans as [Plan, ...Plan[]];

/** Serves the plans in listed, and answers each start with started, which lists the newest plan if it's a 201. */
async function servePlans(page: Page, started: { status: number; body?: string }) {
  const starts: unknown[] = [];
  let listed = older;
  await page.route('/api/planner/plans', async (route) => {
    if (route.request().method() !== 'POST') {
      await route.fulfill({ json: { plans: listed } });
      return;
    }
    starts.push(route.request().postDataJSON());
    if (started.status === 201) {
      listed = plans;
      await route.fulfill({ status: 201, json: { id: newest.id } });
      return;
    }
    await route.fulfill(started);
  });
  return starts;
}

/** The start's status line. */
function startStatus(page: Page) {
  return page.getByRole('main').locator('form').getByRole('status');
}

/** The URLs of the documents the page loads from now on, which a page the app shows in place doesn't add to. */
function documentLoads(page: Page): string[] {
  const loads: string[] = [];
  page.on('request', (request) => {
    if (request.resourceType() === 'document') {
      loads.push(request.url());
    }
  });
  return loads;
}

test.beforeEach(async ({ page }) => {
  await page.route('/api/home', (route) => route.fulfill({ json: home }));
});

test("a prompt starts a plan and opens the plan's page in place, and Back lists the plan first", async ({ page }) => {
  const starts = await servePlans(page, { status: 201 });
  await serveConversation(page, newest.id);
  await page.goto('/planner');
  await expect(group(page, 'Your plans').getByText('3 plans')).toBeVisible();
  const loads = documentLoads(page);

  await page
    .getByRole('textbox', { name: 'Prompt' })
    .fill('Add a dark theme to the docs site.\nKeep the light one as it is.');
  await page.getByRole('button', { name: 'Start' }).click();

  await expect(page).toHaveURL(`/planner/${newest.id}`);
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Plan');
  await expect(group(page, 'The conversation').getByRole('listitem')).toHaveCount(conversation.messages.length);
  expect(starts).toEqual([{ prompt: 'Add a dark theme to the docs site.\nKeep the light one as it is.' }]);

  await page.goBack();

  const yours = group(page, 'Your plans');
  await expect(yours.getByText('4 plans')).toBeVisible();
  await expect(yours.getByRole('listitem').first()).toHaveText('Today Add a dark theme to the docs site');
  await expect(page.getByRole('textbox', { name: 'Prompt' })).toHaveValue('');
  await expect(startStatus(page)).toHaveText('');
  expect(loads).toEqual([]);
});

test('a failed start keeps the prompt and says why', async ({ page }) => {
  const starts = await servePlans(page, { status: 502, body: 'Bad Gateway\n' });
  await page.goto('/planner');
  const prompt = page.getByRole('textbox', { name: 'Prompt' });

  await prompt.fill('Plan a trip');
  await page.getByRole('button', { name: 'Start' }).click();

  await expect(startStatus(page)).toHaveText("Copilot couldn't start the plan.");
  await expect(prompt).toHaveValue('Plan a trip');
  await expect(group(page, 'Your plans').getByText('3 plans')).toBeVisible();
  await expect(page).toHaveURL('/planner');
  expect(starts).toHaveLength(1);
});
