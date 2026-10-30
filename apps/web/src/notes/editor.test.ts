import { EditorView } from '@codemirror/view';
import { afterEach, expect, test } from 'vitest';

import { noteEditor } from './editor';

const markdown = ['---', 'tags: [guitar]', '---', '# To do', '', '- [ ] Tune'].join('\n');

afterEach(() => {
  document.body.replaceChildren();
});

// An element on the page with the editor in it, and the attachment's cleanup.
function attached(text: string) {
  const host = document.createElement('div');
  document.body.append(host);
  const cleanup = noteEditor(text)(host);
  return { host, cleanup };
}

test('shows the note’s Markdown in a field named Markdown, with the focus in it', () => {
  const { host } = attached(markdown);

  const field = host.querySelector('.cm-content');
  expect(field).toHaveAttribute('role', 'textbox');
  expect(field).toHaveAccessibleName('Markdown');
  expect(field).toHaveFocus();
  expect([...host.querySelectorAll('.cm-line')].map((line) => line.textContent)).toEqual(markdown.split('\n'));
  expect(EditorView.findFromDOM(host)?.state.doc.toString()).toBe(markdown);
});

test('undoes an edit', () => {
  const { host } = attached('Tune');
  const view = EditorView.findFromDOM(host)!;

  view.dispatch({ changes: { from: 4, insert: ' the guitar' }, userEvent: 'input.type' });
  expect(view.state.doc.toString()).toBe('Tune the guitar');

  view.contentDOM.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', ctrlKey: true, bubbles: true }));
  expect(view.state.doc.toString()).toBe('Tune');
});

test('goes with the element', () => {
  const { host, cleanup } = attached(markdown);

  (cleanup as () => void)();

  expect(host).toBeEmptyDOMElement();
});
