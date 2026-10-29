import { afterEach, expect, test } from 'vitest';

import { focusLost } from './focus';

afterEach(() => {
  document.body.replaceChildren();
});

function button(parent: HTMLElement = document.body): HTMLButtonElement {
  return parent.appendChild(document.createElement('button'));
}

function dialog(): HTMLDialogElement {
  const element = document.body.appendChild(document.createElement('dialog'));
  element.open = true;
  return element;
}

test('counts the focus on the body as lost', () => {
  expect(document.activeElement).toBe(document.body);
  expect(focusLost()).toBe(true);
});

// jsdom focuses nothing in a closed dialog: the focus goes in while it's open, and stays as it closes.
test('counts the focus left in a closed dialog as lost', () => {
  const closed = dialog();
  button(closed).focus();
  closed.open = false;

  expect(focusLost()).toBe(true);
});

test('keeps the focus in an open dialog', () => {
  button(dialog()).focus();

  expect(focusLost()).toBe(false);
});

test('keeps the focus where the user moved it', () => {
  button().focus();

  expect(focusLost()).toBe(false);
});
