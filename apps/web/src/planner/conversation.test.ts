import { expect, test } from 'vitest';

import { type Change, type Conversation, changed } from './conversation';

const prompt = { id: 'u1', from: 'user', text: 'Plan a trip' } as const;
const talk: Conversation = {
  messages: [prompt],
  writing: [{ id: 'a1', from: 'assistant', text: 'Where' }],
  working: true,
  failure: '',
};
const other: Conversation = { messages: [], writing: [], working: false, failure: '' };

test.each<{ about: string; change: Change; want: Conversation }>([
  { about: 'a new conversation replaces it', change: { name: 'conversation', data: other }, want: other },
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
    about: 'a status of an agent at work keeps the messages being written',
    change: { name: 'status', data: { working: true, failure: '' } },
    want: talk,
  },
  {
    about: 'a status of an agent that stopped drops them, and tells why',
    change: { name: 'status', data: { working: false, failure: 'Rate limited' } },
    want: { ...talk, writing: [], working: false, failure: 'Rate limited' },
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

test('leaves the conversation it changes as it was', () => {
  const before = structuredClone(talk);

  changed(talk, { name: 'delta', data: { id: 'a1', from: 'assistant', text: ' to?' } });
  changed(talk, { name: 'message', data: { id: 'a1', from: 'assistant', text: 'Where to?' } });

  expect(talk).toEqual(before);
});
