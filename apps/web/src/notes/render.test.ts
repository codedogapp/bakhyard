import { expect, test } from 'vitest';

import { renderNote } from './render';

// The note's HTML, rendered under the title "To do".
function html(markdown: string): string {
  return renderNote(markdown, 'To do').html;
}

test('renders GFM: tables with their alignment, strikethrough, autolinks and tasks', () => {
  const markdown = [
    '| Left | Middle | Right |',
    '| :--- | :---: | ---: |',
    '| a | b | c |',
    '',
    '~~gone~~ and www.example.com',
    '',
    '- [ ] open',
    '- [x] done',
  ].join('\n');

  const out = html(markdown);

  expect(out).toContain('<th align="left">Left</th>');
  expect(out).toContain('<th align="center">Middle</th>');
  expect(out).toContain('<td align="right">c</td>');
  expect(out).toContain('<del>gone</del>');
  expect(out).toContain('<a href="http://www.example.com">www.example.com</a>');
  expect(out).toContain('<li class="task-list-item"><input type="checkbox" disabled> open</li>');
  expect(out).toContain('<li class="task-list-item"><input type="checkbox" checked disabled> done</li>');
});

test('breaks a line at each newline, as Obsidian does', () => {
  expect(html('one\ntwo')).toBe('<p>one<br>\ntwo</p>');
});

test('keeps a code block as written, in its case, with its language as a class', () => {
  expect(html('```Intro\nEm   C   G\n  Hello <there>\n```')).toBe(
    '<pre><code class="language-Intro">Em   C   G\n  Hello &#x3C;there>\n</code></pre>'
  );
});

test.each([
  { name: 'the title', markdown: '# To do\n\nText', want: '<p>Text</p>' },
  { name: 'the title after the frontmatter', markdown: '---\na: b\n---\n# To do\nText', want: '<p>Text</p>' },
  { name: 'the title with formatting', markdown: '# *To* `do`\n\nText', want: '<p>Text</p>' },
  { name: 'the title as setext', markdown: 'To do\n===\n\nText', want: '<p>Text</p>' },
  { name: 'another H1', markdown: '# Today\n\nText', want: '<h2>Today</h2>\n<p>Text</p>' },
  { name: 'the title in another case', markdown: '# to do', want: '<h2>to do</h2>' },
  { name: 'the title as an H2', markdown: '## To do', want: '<h2>To do</h2>' },
  { name: 'the title after a paragraph', markdown: 'Text\n\n# To do', want: '<p>Text</p>\n<h2>To do</h2>' },
  { name: 'the title twice', markdown: '# To do\n# To do', want: '<h2>To do</h2>' },
])('drops the first block only when it is an H1 with the title: $name', ({ markdown, want }) => {
  expect(html(markdown)).toBe(want);
});

test('moves every heading one level down when an H1 is left, in quotes and lists too, an h6 staying an h6', () => {
  const markdown = [
    '# One',
    '## Two',
    '### Three',
    '#### Four',
    '##### Five',
    '###### Six',
    '> # Quoted',
    '- # Listed',
  ];

  const out = html(markdown.join('\n'));

  expect(out).not.toMatch(/<h1/);
  for (const [tag, text] of [
    ['h2', 'One'],
    ['h3', 'Two'],
    ['h4', 'Three'],
    ['h5', 'Four'],
    ['h6', 'Five'],
    ['h6', 'Six'],
    ['h2', 'Quoted'],
    ['h2', 'Listed'],
  ]) {
    expect(out).toContain(`<${tag}>${text}</${tag}>`);
  }
});

test.each([
  { name: 'the title is dropped', markdown: '# To do\n## Section\n### Part\n###### Six' },
  { name: 'the note has no H1', markdown: '## Section\n### Part\n###### Six' },
])('keeps the levels when no H1 is left: $name', ({ markdown }) => {
  expect(html(markdown)).toBe('<h2>Section</h2>\n<h3>Part</h3>\n<h6>Six</h6>');
});

test('gives the properties from the frontmatter, which is not part of the HTML', () => {
  const note = renderNote('---\ntags: [a]\nalbum: B\n---\nText', 'To do');

  expect(note.properties).toEqual([
    { key: 'tags', value: ['a'] },
    { key: 'album', value: 'B' },
  ]);
  expect(note.html).toBe('<p>Text</p>');
});

test.each([
  { name: 'no frontmatter', markdown: 'Text' },
  { name: 'frontmatter that does not parse', markdown: '---\ntags: [a\n---\nText' },
  { name: 'a rule, not frontmatter, after the first line', markdown: '\n---\na: b\n---\nText' },
])('renders a note with $name, without properties', ({ markdown }) => {
  const note = renderNote(markdown, 'To do');

  expect(note.properties).toEqual([]);
  expect(note.html).toContain('<p>Text</p>');
});

test.each([
  { name: 'a script', markdown: '<script>alert(1)</script>' },
  { name: 'an inline script', markdown: 'a <script>alert(1)</script> b' },
  { name: 'an iframe', markdown: '<iframe src="https://example.com"></iframe>' },
  { name: 'a style', markdown: '<style>body { display: none }</style>' },
  { name: 'a form', markdown: '<form action="https://example.com"><button>Go</button></form>' },
  { name: 'an event handler', markdown: '<img src="x" onerror="alert(1)">' },
  { name: 'an inline event handler', markdown: 'a <b onclick="alert(1)">b</b>' },
  { name: 'an input', markdown: '<input type="text" value="x">' },
  { name: 'a comment', markdown: '<!-- hidden -->' },
])('drops raw HTML: $name', ({ markdown }) => {
  expect(html(markdown)).not.toMatch(/<(?!\/?p>)|on\w+=/);
});

test.each([
  { name: 'javascript:', markdown: '[x](javascript:alert(1))' },
  { name: 'JavaScript: in any case', markdown: '[x](JaVaScRiPt:alert(1))' },
  { name: 'data:', markdown: '[x](data:text/html,<script>alert(1)</script>)' },
  { name: 'vbscript:', markdown: '[x](vbscript:msgbox)' },
])('keeps a link to $name, without its href', ({ markdown }) => {
  expect(html(markdown)).toBe('<p><a>x</a></p>');
});

test('keeps links to the web and mail', () => {
  expect(html('[a](https://example.com) [b](mailto:a@example.com)')).toBe(
    '<p><a href="https://example.com">a</a> <a href="mailto:a@example.com">b</a></p>'
  );
});

test('prefixes the ids it keeps, so a note can not take the page’s own', () => {
  expect(html('Text[^1]\n\n[^1]: Note')).toMatch(/id="user-content-/);
  expect(html('Text[^1]\n\n[^1]: Note')).not.toMatch(/id="(?!user-content-)/);
});
