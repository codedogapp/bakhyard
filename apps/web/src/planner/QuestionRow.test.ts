import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import type { Question } from './conversation';
import QuestionRow from './QuestionRow.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

const city: Question = { id: 'q1', text: 'Which city?', choices: ['Rome', 'Paris'], freeform: true };

function serve(response: Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(response);
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

const sent = () => Promise.resolve(new Response(null, { status: 204 }));
const buttons = () => screen.getAllByRole('button').map((button) => button.textContent?.trim());
const field = () => screen.getByRole('textbox', { name: 'Answer to: Which city?' });

test("shows the question as Copilot's, with its choices, and a field for another answer", () => {
  render(QuestionRow, { props: { id: 'p1', question: city } });

  expect(screen.getByText('Copilot')).toBeInTheDocument();
  expect(screen.getByText('Which city?')).toBeInTheDocument();
  expect(buttons()).toEqual(['Rome', 'Paris', 'Send']);
  expect(field()).toHaveAttribute('placeholder', 'Type an answer');
  expect(screen.getByRole('status')).toHaveTextContent('');
});

test.each([
  {
    about: 'its choices alone when it takes no other answer',
    question: { ...city, freeform: false },
    want: ['Rome', 'Paris'],
  },
  { about: 'the field alone without choices', question: { ...city, choices: [] }, want: ['Send'] },
  {
    about: 'a choice twice when the agent does',
    question: { ...city, choices: ['Rome', 'Rome'] },
    want: ['Rome', 'Rome', 'Send'],
  },
])('offers $about', ({ question, want }) => {
  render(QuestionRow, { props: { id: 'p1', question } });

  expect(buttons()).toEqual(want);
  expect(screen.queryAllByRole('textbox')).toHaveLength(question.freeform ? 1 : 0);
});

test('sends a choice as the answer, and says so', async () => {
  const fetchMock = serve(sent());
  render(QuestionRow, { props: { id: 'p1', question: city } });

  await fireEvent.click(screen.getByRole('button', { name: 'Paris' }));

  expect(await screen.findByText('Sent')).toHaveRole('status');
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans/p1/questions/q1/answer', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{"text":"Paris"}',
  });
});

test('sends the answer typed in the field with Send', async () => {
  const fetchMock = serve(sent());
  render(QuestionRow, { props: { id: 'p1', question: city } });

  await fireEvent.input(field(), { target: { value: 'Milan' } });
  await fireEvent.click(screen.getByRole('button', { name: 'Send' }));

  expect(await screen.findByText('Sent')).toHaveRole('status');
  expect(fetchMock.mock.calls[0]?.[1]?.body).toBe('{"text":"Milan"}');
});

// Enter in the field submits its form, as a browser does.
test('sends the answer typed in the field with Enter, and keeps the browser from leaving the page', async () => {
  const fetchMock = serve(sent());
  render(QuestionRow, { props: { id: 'p1', question: city } });
  await fireEvent.input(field(), { target: { value: 'Milan' } });
  const enter = new Event('submit', { bubbles: true, cancelable: true });

  field().closest('form')?.dispatchEvent(enter);

  expect(enter.defaultPrevented).toBe(true);
  expect(await screen.findByText('Sent')).toHaveRole('status');
  expect(fetchMock.mock.calls[0]?.[1]?.body).toBe('{"text":"Milan"}');
});

test('is read-only while it sends, and keeps the answer when the send fails', async () => {
  const response = Promise.withResolvers<Response>();
  serve(response.promise);
  render(QuestionRow, { props: { id: 'p1', question: city } });

  await fireEvent.input(field(), { target: { value: 'Milan' } });
  await fireEvent.click(screen.getByRole('button', { name: 'Send' }));

  expect(screen.getByRole('status')).toHaveTextContent('Sending…');
  expect(field()).toHaveAttribute('readonly');
  for (const button of screen.getAllByRole('button')) {
    expect(button).toHaveAttribute('aria-disabled', 'true');
  }

  response.resolve(new Response('no such question\n', { status: 404 }));

  await vi.waitFor(() =>
    expect(screen.getByRole('status')).toHaveTextContent('Copilot no longer waits for this question.')
  );
  expect(field()).not.toHaveAttribute('readonly');
  expect(field()).toHaveValue('Milan');
  for (const button of screen.getAllByRole('button')) {
    expect(button).toHaveAttribute('aria-disabled', 'false');
  }
});
