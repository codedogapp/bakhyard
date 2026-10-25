import { expect, test } from 'vitest';

import { renderNote } from './render';

const vault = [
  'Daily/To do.md',
  'Daily/Plan.md',
  'Plan.md',
  'Songs/Em.md',
  'Attachments/x.png',
  'Attachments/score.pdf',
];
const toDo = { path: 'Daily/To do.md', title: 'To do', folder: 'Daily' };

// The HTML of the note "Daily/To do", whose file holds markdown, in the vault above.
function html(markdown: string): string {
  return renderNote(markdown, toDo, vault).html;
}

const em = '<a href="/notes/Songs/Em">Em</a>';

test.each([
  { name: 'a note', markdown: 'See [[Em]] now.', want: `<p>See ${em} now.</p>` },
  { name: 'the note in its own folder', markdown: '[[Plan]]', want: '<p><a href="/notes/Daily/Plan">Plan</a></p>' },
  { name: 'a name with a space', markdown: '[[To do]]', want: '<p><a href="/notes/Daily/To%20do">To do</a></p>' },
  { name: 'display text', markdown: '[[Em|E minor]]', want: '<p><a href="/notes/Songs/Em">E minor</a></p>' },
  { name: 'a heading of a note', markdown: '[[Em#Chords]]', want: '<p><a href="/notes/Songs/Em">Em > Chords</a></p>' },
  { name: 'a note embed', markdown: '![[Em]]', want: `<p>${em}</p>` },
  { name: 'two links', markdown: '[[Em]] and [[Em]]', want: `<p>${em} and ${em}</p>` },
  { name: 'emphasis', markdown: '*[[Em]]* **[[Em]]**', want: `<p><em>${em}</em> <strong>${em}</strong></p>` },
  { name: 'a list', markdown: '- [[Em]]', want: `<ul>\n<li>${em}</li>\n</ul>` },
  { name: 'a quote', markdown: '> [[Em]]', want: `<blockquote>\n<p>${em}</p>\n</blockquote>` },
  { name: 'a heading', markdown: '## [[Em]]', want: `<h2 id="user-content-em">${em}</h2>` },
])('links a wikilink: $name', ({ markdown, want }) => {
  expect(html(markdown)).toBe(want);
});

test('links a wikilink in a table, its "|" escaped', () => {
  expect(html('| A |\n| - |\n| [[Em\\|E minor]] |')).toContain('<td><a href="/notes/Songs/Em">E minor</a></td>');
});

test.each([
  { name: 'a missing note', markdown: '[[Nothing|Gone]]', want: 'Gone' },
  { name: 'a file that is not a note', markdown: '[[x.png]]', want: 'x.png' },
  { name: 'an embed of a missing image', markdown: '![[y.png]]', want: 'y.png' },
  { name: 'an embed of a file that is neither a note nor an image', markdown: '![[score.pdf]]', want: 'score.pdf' },
])('dims the text of a wikilink that leads nowhere: $name', ({ markdown, want }) => {
  expect(html(markdown)).toBe(`<p><span class="unresolved">${want}</span></p>`);
});

const image = '<img src="/api/notes/files/Attachments/x.png" alt="x.png">';

test.each([
  { name: 'alone', markdown: '![[x.png]]', want: `<p>${image}</p>` },
  { name: 'in a sentence', markdown: 'See ![[x.png]] now.', want: `<p>See ${image} now.</p>` },
  {
    name: 'on a list item’s next line',
    markdown: '- Text\n  ![[x.png]]',
    want: `<ul>\n<li>Text<br>\n${image}</li>\n</ul>`,
  },
  { name: 'with a size', markdown: '![[x.png|300]]', want: `<p>${image}</p>` },
])('shows the image of an image embed: $name', ({ markdown, want }) => {
  expect(html(markdown)).toBe(want);
});

test('shows two image embeds side by side in a table', () => {
  expect(html('| A | B |\n| - | - |\n| ![[x.png]] | ![[x.png]] |')).toContain(
    `<tr>\n<td>${image}</td>\n<td>${image}</td>\n</tr>`
  );
});

test.each([
  { name: 'inline code', markdown: '`[[Em]]`', want: '<p><code>[[Em]]</code></p>' },
  { name: 'an image embed in inline code', markdown: '`![[x.png]]`', want: '<p><code>![[x.png]]</code></p>' },
  { name: 'a code block', markdown: '```\n[[Em]]\n```', want: '<pre><code>[[Em]]\n</code></pre>' },
  {
    name: 'a link’s text',
    markdown: '[[[Em]]](https://example.com)',
    want: '<p><a href="https://example.com">[[Em]]</a></p>',
  },
  {
    name: 'a reference link’s text',
    markdown: '[[[Em]]][r]\n\n[r]: https://example.com',
    want: '<p><a href="https://example.com">[[Em]]</a></p>',
  },
  { name: 'no path and no heading', markdown: '[[#]] [[ | Text]] [[]]', want: '<p>[[#]] [[ | Text]] [[]]</p>' },
  { name: 'a bracket inside', markdown: '[[a[b]] [[a]b]]', want: '<p>[[a[b]] [[a]b]]</p>' },
  { name: 'a newline inside', markdown: '[[Em\nE minor]]', want: '<p>[[Em<br>\nE minor]]</p>' },
  { name: 'emphasis inside', markdown: '[[_Em_]]', want: '<p>[[<em>Em</em>]]</p>' },
  {
    name: 'an autolink inside',
    markdown: '[[www.example.com]]',
    want: '<p>[[<a href="http://www.example.com">www.example.com</a>]]</p>',
  },
  {
    name: 'a link definition of its name',
    markdown: '[[Em]]\n\n[Em]: https://example.com',
    want: '<p>[<a href="https://example.com">Em</a>]</p>',
  },
])('keeps a wikilink as written: $name', ({ markdown, want }) => {
  expect(html(markdown)).toBe(want);
});

// The heading that the link's href leads to, as a browser finds it.
function target(markdown: string): Element | null {
  const page = new DOMParser().parseFromString(html(markdown), 'text/html');
  const href = page.querySelector('a')?.getAttribute('href') ?? '';
  expect(href).toMatch(/^#user-content-[^\s#]+$/);
  return page.getElementById(decodeURIComponent(href.slice(1)));
}

test.each([
  { name: 'in its case', markdown: '## Chords\n\n[[#Chords]]', want: 'Chords' },
  { name: 'in any case', markdown: '## Chords\n\n[[#cHORDS]]', want: 'Chords' },
  {
    name: 'with non-ASCII letters and spaces',
    markdown: '## Café  au lait\n\n[[#café au lait]]',
    want: 'Café  au lait',
  },
  { name: 'with a "#"', markdown: '## C# major\n\n[[#C# major|C#]]', want: 'C# major' },
  { name: 'shifted down a level', markdown: '# Intro\n\n[[#Intro]]', want: 'Intro' },
  { name: 'with formatting', markdown: '## *Big* `idea`\n\n[[#Big idea]]', want: 'Big idea' },
  { name: 'the first of two', markdown: '## A\n\nOne\n\n## A\n\nTwo\n\n[[#A]]', want: 'A' },
])('links a heading of the note itself: $name', ({ markdown, want }) => {
  const heading = target(markdown);

  expect(heading?.tagName).toBe('H2');
  expect(heading?.textContent).toBe(want);
});

test('gives an ID to the first heading of a kind only, and none to an empty one', () => {
  expect(html('## A\n## a\n##')).toBe('<h2 id="user-content-a">A</h2>\n<h2>a</h2>\n<h2></h2>');
});

test('does not let raw HTML add the class of a wikilink that leads nowhere', () => {
  expect(html('<span class="unresolved">x</span>')).toBe('<p>x</p>');
});
