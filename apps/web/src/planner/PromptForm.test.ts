import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import PromptForm from './PromptForm.svelte';

afterEach(() => {
  vi.unstubAllGlobals();
});

function renderForm() {
  const started = vi.fn();
  render(PromptForm, { props: { started } });
  const prompt = screen.getByRole('textbox', { name: 'Prompt' });
  const button = screen.getByRole('button', { name: 'Start' });
  return { started, prompt, button };
}

test('starts a plan from the prompt, empties it, and tells the page', async () => {
  const started = Response.json({ id: '0b6c1f9e-2d4a-4c3b-9a8e-5f7d6c4b3a21' }, { status: 201 });
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockResolvedValue(started));
  const form = renderForm();
  expect(form.prompt).toHaveAttribute('placeholder', 'What should Copilot plan?');
  expect(screen.getByRole('status')).toHaveTextContent('');

  await fireEvent.input(form.prompt, { target: { value: 'Plan a trip\nto Rome' } });
  await fireEvent.click(form.button);

  expect(await screen.findByText('Started')).toHaveRole('status');
  expect(form.prompt).toHaveValue('');
  expect(form.started).toHaveBeenCalledOnce();
});

test('is read-only while it starts, and keeps the prompt when the start fails', async () => {
  const answer = Promise.withResolvers<Response>();
  vi.stubGlobal('fetch', vi.fn<typeof fetch>().mockReturnValue(answer.promise));
  const form = renderForm();

  await fireEvent.input(form.prompt, { target: { value: 'Plan a trip' } });
  await fireEvent.click(form.button);

  expect(screen.getByRole('status')).toHaveTextContent('Starting…');
  expect(form.prompt).toHaveAttribute('readonly');
  expect(form.button).toHaveAttribute('aria-disabled', 'true');

  answer.resolve(new Response('Bad Gateway\n', { status: 502 }));

  await vi.waitFor(() => expect(screen.getByRole('status')).toHaveTextContent("Copilot couldn't start the plan."));
  expect(form.prompt).not.toHaveAttribute('readonly');
  expect(form.button).toHaveAttribute('aria-disabled', 'false');
  expect(form.prompt).toHaveValue('Plan a trip');
  expect(form.started).not.toHaveBeenCalled();
});
