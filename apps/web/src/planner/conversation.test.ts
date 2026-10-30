import { expect, test } from 'vitest';

import { type Change, type Conversation, changed } from './conversation';

const prompt = { id: 'u1', from: 'user', text: 'Plan a trip' } as const;
const city = { id: 'q1', text: 'Which city?', choices: ['Rome', 'Paris'], freeform: false };
const days = { id: 'q2', text: 'How many days?', choices: [], freeform: true };
const talk: Conversation = {
  messages: [prompt],
  writing: [{ id: 'a1', from: 'assistant', text: 'Where' }],
  questions: [city],
  working: true,
  failure: '',
};
const other: Conversation = { messages: [], writing: [], questions: [days], working: true, failure: '' };

test.each<{ about: string; change: Change; want: Conversation }>([
  { about: 'a new conversation replaces it', change: { name: 'conversation', data: other }, want: other },
  {
    about: 'a new conversation of an agent that stopped leaves out the questions that it took back',
    change: { name: 'conversation', data: { ...other, working: false } },
    want: { ...other, questions: [], working: false },
  },
  {
    about: "a delta adds to the message with its ID that's being written",
    change: { name: 'delta', data: { id: 'a1', from: 'assistant', text: ' to?' } },
    want: { ...talk, writing: [{ id: 'a1', from: 'assistant', text: 'Where to?' }] },
  },
  {
    about: 'a delta of another message starts one',
    change: { name: 'delta', data: { id: 'a2', from: 'assistant', text: 'Hi' } },
    want: { ...talk, writing: [...talk.writing, { id: 'a2', from: 'assistant', text: 'Hi' }] },
  },
  {
    about: 'a whole message joins the messages and leaves those being written',
    change: { name: 'message', data: { id: 'a1', from: 'assistant', text: 'Where to?' } },
    want: { ...talk, messages: [prompt, { id: 'a1', from: 'assistant', text: 'Where to?' }], writing: [] },
  },
  {
    about: "another's whole message leaves the one being written",
    change: { name: 'message', data: { id: 'u2', from: 'user', text: 'Rome' } },
    want: { ...talk, messages: [prompt, { id: 'u2', from: 'user', text: 'Rome' }] },
  },
  {
    about: 'a question joins those that wait, after them',
    change: { name: 'question', data: days },
    want: { ...talk, questions: [city, days] },
  },
  {
    about: 'a question answered leaves them',
    change: { name: 'answered', data: city },
    want: { ...talk, questions: [] },
  },
  { about: "another's answer leaves them as they are", change: { name: 'answered', data: days }, want: talk },
  {
    about: 'a status of an agent at work keeps the messages being written and the questions',
    change: { name: 'status', data: { working: true, failure: '' } },
    want: talk,
  },
  {
    about: 'a status of an agent that stopped drops them, and tells why',
    change: { name: 'status', data: { working: false, failure: 'Rate limited' } },
    want: { ...talk, writing: [], questions: [], working: false, failure: 'Rate limited' },
  },
])('$about', ({ change, want }) => {
  expect(changed(talk, change)).toEqual(want);
});

test('adds a delta to its own message among several being written', () => {
  const writing = [
    { id: 'a1', from: 'assistant', text: 'One' },
    { id: 'a2', from: 'assistant', text: 'Two' },
  ] as const;

  const next = changed({ ...talk, writing: [...writing] }, { name: 'delta', data: { ...writing[1], text: '!' } });

  expect(next.writing).toEqual([writing[0], { ...writing[1], text: 'Two!' }]);
});

test("forgets a question that the agent took back, so it doesn't come back as the agent works again", () => {
  const stopped = changed(talk, { name: 'status', data: { working: false, failure: '' } });

  const again = changed(stopped, { name: 'status', data: { working: true, failure: '' } });

  expect(again.questions).toEqual([]);
});

test('leaves the conversation it changes as it was', () => {
  const before = structuredClone(talk);

  changed(talk, { name: 'delta', data: { id: 'a1', from: 'assistant', text: ' to?' } });
  changed(talk, { name: 'message', data: { id: 'a1', from: 'assistant', text: 'Where to?' } });
  changed(talk, { name: 'question', data: days });
  changed(talk, { name: 'answered', data: city });
  changed(talk, { name: 'status', data: { working: false, failure: '' } });

  expect(talk).toEqual(before);
});
