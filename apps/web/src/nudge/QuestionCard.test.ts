import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Question } from './nudge';
import QuestionCard from './QuestionCard.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

const question: Question = {
  id: 'q1',
  project: 'atlas (work-laptop)',
  text: 'Merge the pull request?',
  details: 'CI is green.',
  options: ['Yes', 'No'],
  asked_at: '2026-10-21T00:41:00Z',
};

function reply(status: number, body = '') {
  const fetchMock = vi.fn<typeof fetch>(async () => new Response(body || null, { status }));
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

function sent(fetchMock: ReturnType<typeof reply>): unknown {
  return JSON.parse(String(fetchMock.mock.calls[0][1]?.body));
}

test('shows the question, its project, when it was asked, and its details', () => {
  render(QuestionCard, { question, answered: vi.fn() });

  expect(screen.getByRole('heading', { level: 3, name: 'Merge the pull request?' })).toBeInTheDocument();
  expect(screen.getByText('atlas (work-laptop)')).toBeInTheDocument();
  expect(screen.getByText('21 October at 09:41')).toBeInTheDocument();
  expect(screen.getByText('CI is green.')).toBeInTheDocument();
});

test('sends an option at once, and says that it is answered', async () => {
  const fetchMock = reply(204);
  const answered = vi.fn();
  render(QuestionCard, { question, answered });

  await fireEvent.click(screen.getByRole('button', { name: 'Yes' }));

  expect(sent(fetchMock)).toEqual({ option: 'Yes' });
  expect(answered).toHaveBeenCalledOnce();
});

test('sends an option with the reply that is typed', async () => {
  const fetchMock = reply(204);
  render(QuestionCard, { question, answered: vi.fn() });

  await fireEvent.input(screen.getByRole('textbox'), { target: { value: '  and add a test ' } });
  await fireEvent.click(screen.getByRole('button', { name: 'No' }));

  expect(sent(fetchMock)).toEqual({ option: 'No', text: 'and add a test' });
});

test('sends a reply alone, and not an empty one', async () => {
  const fetchMock = reply(204);
  render(QuestionCard, { question: { ...question, options: [], details: '' }, answered: vi.fn() });

  await fireEvent.click(screen.getByRole('button', { name: 'Send' }));
  expect(fetchMock).not.toHaveBeenCalled();

  const field = screen.getByRole('textbox', { name: /Reply to/ });
  await fireEvent.input(field, { target: { value: 'Not yet' } });
  await fireEvent.submit(field);
  expect(sent(fetchMock)).toEqual({ text: 'Not yet' });
});

test('says why an answer failed, and stays', async () => {
  reply(404);
  const answered = vi.fn();
  render(QuestionCard, { question, answered });

  await fireEvent.click(screen.getByRole('button', { name: 'Yes' }));

  expect(await screen.findByText('The agent no longer waits for this question.')).toBeInTheDocument();
  expect(answered).not.toHaveBeenCalled();
});
