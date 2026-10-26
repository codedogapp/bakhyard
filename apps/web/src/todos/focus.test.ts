import { afterEach, expect, test, vi } from 'vitest';

import { Focus, focusField } from './focus';

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

test('gives the lost focus to the button', () => {
  const focusAdd = vi.fn<() => void>();
  const target = button();

  new Focus(focusAdd).recover(target);

  expect(document.activeElement).toBe(target);
  expect(focusAdd).not.toHaveBeenCalled();
});

test('gives the focus left in a closed dialog to the button', () => {
  const target = button();
  const closed = dialog();
  button(closed).focus();
  closed.open = false;

  new Focus(() => undefined).recover(target);

  expect(document.activeElement).toBe(target);
});

test('leaves the focus in an open dialog', () => {
  const inside = button(dialog());
  inside.focus();

  new Focus(() => undefined).recover(button());

  expect(document.activeElement).toBe(inside);
});

test('leaves the focus where the user moved it', () => {
  const focusAdd = vi.fn<() => void>();
  const elsewhere = button();
  elsewhere.focus();

  new Focus(focusAdd).recover(button());

  expect(document.activeElement).toBe(elsewhere);
  expect(focusAdd).not.toHaveBeenCalled();
});

test('gives the lost focus to the Add field without a button', () => {
  const focusAdd = vi.fn<() => void>();

  new Focus(focusAdd).recover(undefined);

  expect(focusAdd).toHaveBeenCalledOnce();
});

test("gives the lost focus to the todo's Edit button", () => {
  const focus = new Focus(() => undefined);
  const [bankEdit, milkEdit] = [button(), button()];
  focus.editButton(2)(bankEdit);
  focus.editButton(1)(milkEdit);

  focus.recoverEdit(1);

  expect(document.activeElement).toBe(milkEdit);
});

test('gives the lost focus to the Add field once the Edit button is gone', () => {
  const focusAdd = vi.fn<() => void>();
  const focus = new Focus(focusAdd);
  const forget = focus.editButton(2)(button());

  forget?.();
  focus.recoverEdit(2);

  expect(focusAdd).toHaveBeenCalledOnce();
});

test('gives the lost focus to the Add field without a todo', () => {
  const focusAdd = vi.fn<() => void>();
  const focus = new Focus(focusAdd);
  focus.editButton(2)(button());

  focus.recoverEdit(undefined);

  expect(focusAdd).toHaveBeenCalledOnce();
});

test('focuses the field with the caret after the title', () => {
  const input = document.body.appendChild(document.createElement('input'));
  input.value = 'Call the bank';
  input.setSelectionRange(0, 4);

  focusField(input);

  expect(document.activeElement).toBe(input);
  expect([input.selectionStart, input.selectionEnd]).toEqual([13, 13]);
});
