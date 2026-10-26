import { expect, test } from 'vitest';

import type { Todo } from './api';
import { Draft } from './draft.svelte';

const bank: Todo = { id: 2, title: 'Call the bank', done: false, priority: 'normal' };
const linked: Todo = { ...bank, ticket: 'WEB-4' };

/** A draft of the todo, then typed and picked as given. */
function drafted(todo: Todo, picks: Partial<Pick<Draft, 'title' | 'priority' | 'ticket'>>): Draft {
  const draft = new Draft();
  draft.reset(todo);
  Object.assign(draft, picks);
  return draft;
}

test("starts over from the todo's title, priority and ticket, none as empty", () => {
  const draft = drafted(bank, { title: 'Call the bank today', priority: 'high', ticket: 'WEB-9' });

  draft.reset({ ...linked, title: 'Buy milk', priority: 'low' });
  expect([draft.title, draft.priority, draft.ticket]).toEqual(['Buy milk', 'low', 'WEB-4']);
  draft.reset(bank);

  expect([draft.title, draft.priority, draft.ticket]).toEqual(['Call the bank', 'normal', '']);
});

test.each([
  ['nothing', bank, {}, undefined],
  ['the title', bank, { title: 'Call the bank today' }, { title: 'Call the bank today' }],
  [
    'the title by white space alone, which the service trims',
    bank,
    { title: ' Call the bank' },
    { title: ' Call the bank' },
  ],
  ['the priority', bank, { priority: 'high' }, { priority: 'high' }],
  ['the ticket, linking one', bank, { ticket: 'WEB-9' }, { ticket: 'WEB-9' }],
  ['the ticket, linking another', linked, { ticket: 'WEB-9' }, { ticket: 'WEB-9' }],
  ['the ticket, unlinking it with an empty key', linked, { ticket: '' }, { ticket: '' }],
  ['nothing, keeping the ticket', linked, {}, undefined],
  [
    'all three',
    linked,
    { title: 'Call the bank today', priority: 'low', ticket: '' },
    { title: 'Call the bank today', priority: 'low', ticket: '' },
  ],
] as const)('changes %s', (_, todo, picks, edit) => {
  expect(drafted(todo, picks).changes(todo)).toStrictEqual(edit);
});
