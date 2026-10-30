import type { Page } from '@playwright/test';

import type { Conversation, Question } from '../src/planner/conversation';

/** A made-up conversation of the demo's newest plan, as its stream first sends it while the agent works. */
export const conversation: Conversation = {
  messages: [
    { id: 'u1', from: 'user', text: 'Add a dark theme to the docs site.\nKeep the light one as it is.' },
    { id: 'a1', from: 'assistant', text: 'Should the code blocks and the diagrams get dark colours too?' },
    { id: 'u2', from: 'user', text: 'Yes, all of it.' },
    {
      id: 'a2',
      from: 'assistant',
      text: "Here's what I found:\n\n1. The colours live in one file, styles/colours.css.\n2. The code blocks have a theme of their own.\n3. The diagrams are SVG files with fixed colours.",
    },
  ],
  writing: [],
  questions: [],
  plan: '',
  working: true,
  failure: '',
};

/** A made-up plan for that conversation, as the agent writes one: a title, then its sections. */
export const themePlan = [
  '# Dark theme for the docs site',
  '',
  '## Approach',
  'Add a dark palette beside the light one, and let `prefers-color-scheme` choose between them.',
  '',
  '## Steps',
  '- [ ] Move the colours into custom properties in `styles/colours.css`',
  '- [ ] Add the dark palette',
  '- [ ] Give the code blocks a dark theme',
  '- [ ] Draw the diagrams with `currentColor`',
  '',
  '## Files',
  '| File | Change |',
  '| :--- | :--- |',
  '| `styles/colours.css` | Both palettes |',
  '| `styles/code.css` | The dark code theme |',
].join('\n');

/** A made-up question of the agent's in that conversation, which waits for the user's answer. */
export const question: Question = {
  id: '6f1c2b9e-3d4a-4e5f-8a7b-9c0d1e2f3a4b',
  text: "Should the theme follow the system's setting, or have a toggle of its own?",
  choices: ['Follow the system', 'Add a toggle'],
  freeform: true,
};

/** A stream's body: each event with its data, after the time the browser should wait before it opens it again. */
export function stream(retryMs: number, ...events: [name: string, data: unknown][]): string {
  const sent = events.map(([name, data]) => `event: ${name}\ndata: ${JSON.stringify(data)}\n\n`);
  return [`retry: ${retryMs}\n\n`, ...sent].join('');
}

/**
 * Answers the stream of the plan with the ID with the conversation, and a wait long enough that the browser doesn't open
 * it again while a test runs.
 */
export async function serveConversation(page: Page, id: string): Promise<void> {
  await page.route(`/api/planner/plans/${id}/events`, (route) =>
    route.fulfill({ contentType: 'text/event-stream', body: stream(600_000, ['conversation', conversation]) })
  );
}
