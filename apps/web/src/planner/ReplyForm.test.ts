import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import ReplyForm from './ReplyForm.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

function renderForm() {
  render(ReplyForm, { props: { id: 'p1' } });
  const reply = screen.getByRole('textbox', { name: 'Reply' });
  const button = screen.getByRole('button', { name: 'Send' });
  return { reply, button };
}

test('sends the reply to the plan, empties it, and says so', async () => {
  const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 }));
  vi.stubGlobal('fetch', fetchMock);
  const form = renderForm();
  expect(form.reply).toHaveAttribute('placeholder', 'Reply to Copilot');
  expect(screen.getByRole('status')).toHaveTextContent('');

  await fireEvent.input(form.reply, { target: { value: 'Make it a week\nand cheap' } });
  await fireEvent.click(form.button);

  expect(await screen.findByText('Sent')).toHaveRole('status');
  expect(form.reply).toHaveValue('');
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/planner/plans/p1/messages', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{"text":"Make it a week\\nand cheap"}',
  });
});

test('is read-only while it sends, and keeps the reply when the send fails', async () => {
  const answer = Promise.withResolvers<Response>();
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockReturnValue(answer.promise));
  const form = renderForm();

  await fireEvent.input(form.reply, { target: { value: 'Make it a week' } });
  await fireEvent.click(form.button);

  expect(screen.getByRole('status')).toHaveTextContent('Sending…');
  expect(form.reply).toHaveAttribute('readonly');
  expect(form.button).toHaveAttribute('aria-disabled', 'true');

  answer.resolve(new Response('no such plan\n', { status: 404 }));

  await vi.waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Copilot no longer has this plan.'));
  expect(form.reply).not.toHaveAttribute('readonly');
  expect(form.button).toHaveAttribute('aria-disabled', 'false');
  expect(form.reply).toHaveValue('Make it a week');
});
