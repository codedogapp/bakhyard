import { expect, test } from 'vitest';

import type { Priority, Todo } from './api';
import { Draft } from './draft.svelte';

const bank: Todo = { id: 2, title: 'Call the bank', done: false, priority: 'normal' };

/** A draft of the bank's todo, then typed and picked as given. */
function drafted(title: string, priority: Priority): Draft {
  const draft = new Draft();
  draft.reset(bank);
  draft.title = title;
  draft.priority = priority;
  return draft;
}

test("starts over from the todo's title and priority", () => {
  const draft = drafted('Call the bank today', 'high');

  draft.reset({ ...bank, title: 'Buy milk', priority: 'low' });

  expect([draft.title, draft.priority]).toEqual(['Buy milk', 'low']);
});

test.each([
  ['nothing', 'Call the bank', 'normal', undefined],
  ['the title', 'Call the bank today', 'normal', { title: 'Call the bank today' }],
  ['the title by white space alone, which the service trims', ' Call the bank', 'normal', { title: ' Call the bank' }],
  ['the priority', 'Call the bank', 'high', { priority: 'high' }],
  ['both', 'Call the bank today', 'low', { title: 'Call the bank today', priority: 'low' }],
] as const)('changes %s', (_, title, priority, edit) => {
  expect(drafted(title, priority).changes(bank)).toStrictEqual(edit);
});
