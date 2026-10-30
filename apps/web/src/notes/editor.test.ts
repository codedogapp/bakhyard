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

function press(view: EditorView, key: string): void {
  view.contentDOM.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }));
}

test.each([
  { name: 'a list item starts the next item', text: '- Tune', after: '- Tune\n- ' },
  { name: 'a task starts the next task', text: '- [x] Tune', after: '- [x] Tune\n- [ ] ' },
  { name: 'a quote goes on quoting', text: '> Slow', after: '> Slow\n> ' },
  { name: 'plain text starts a plain line', text: 'Tune', after: 'Tune\n' },
  {
    name: 'a list in the frontmatter, which is YAML, starts a plain line',
    text: '---\ntags:\n- guitar\n---\n',
    at: 18,
    after: '---\ntags:\n- guitar\n\n---\n',
  },
])('Enter after $name', ({ text, at = text.length, after }) => {
  const { host } = attached(text);
  const view = EditorView.findFromDOM(host)!;
  view.dispatch({ selection: { anchor: at } });

  press(view, 'Enter');

  expect(view.state.doc.toString()).toBe(after);
});

// A paste with text/plain on the clipboard, which jsdom's events can't carry.
function paste(view: EditorView, text: string): void {
  const event = new Event('paste', { bubbles: true, cancelable: true });
  Object.defineProperty(event, 'clipboardData', { value: { getData: () => text } });
  view.contentDOM.dispatchEvent(event);
}

test('a URL pasted over text links the text', () => {
  const { host } = attached('See the lesson');
  const view = EditorView.findFromDOM(host)!;
  view.dispatch({ selection: { anchor: 4, head: 14 } });

  paste(view, 'https://example.com/lesson');

  expect(view.state.doc.toString()).toBe('See [the lesson](https://example.com/lesson)');
});

test('goes with the element', () => {
  const { host, cleanup } = attached(markdown);

  (cleanup as () => void)();

  expect(host).toBeEmptyDOMElement();
});
