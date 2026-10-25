import { expect, onTestFinished, test, vi } from 'vitest';

import { renderNote } from './render';

// The renderer's math (render.ts), which KaTeX shows.

const maths = { path: 'Maths.md', title: 'Maths', folder: '' };

function html(markdown: string): string {
  return renderNote(markdown, maths, ['Maths.md']).html;
}

function page(markdown: string): Document {
  return new DOMParser().parseFromString(html(markdown), 'text/html');
}

// The TeX of each formula on the page, from the MathML that KaTeX writes beside it, with "block: " before a block's.
function formulas(markdown: string): string[] {
  return [...page(markdown).querySelectorAll('.katex')].map((formula) => {
    const tex = formula.querySelector('annotation[encoding="application/x-tex"]')?.textContent ?? '';
    return formula.parentElement?.classList.contains('katex-display') ? `block: ${tex}` : tex;
  });
}

test.each([
  { name: 'a paragraph', markdown: 'Let $x$ be the count' },
  { name: 'a list item', markdown: '- where $x$ is the count' },
  { name: 'a table cell', markdown: '| Value |\n| - |\n| $x$ |' },
  { name: 'a heading', markdown: '## Solve for $x$' },
  { name: 'a quote', markdown: '> so $x$ grows' },
])('shows math between single dollars in $name, in line with the text', ({ markdown }) => {
  expect(formulas(markdown)).toEqual(['x']);
  expect(html(markdown)).not.toContain('$');
});

test('shows the symbols a formula names', () => {
  expect(page('- when $a \\neq b$').querySelector('math')?.textContent).toContain('≠');
});

test('shows math between lines of double dollars as a block', () => {
  expect(formulas('Before\n\n$$\n\\sum_{i=1}^{n} i\n$$\n\nAfter')).toEqual(['block: \\sum_{i=1}^{n} i']);
});

test.each([
  { name: 'in a code span', markdown: 'Type `$x$`', want: '<p>Type <code>$x$</code></p>' },
  { name: 'in a code block', markdown: '```\n$x$\n```', want: '<pre><code>$x$\n</code></pre>' },
  { name: 'escaped', markdown: 'From \\$5 to \\$10', want: '<p>From $5 to $10</p>' },
])('keeps dollars $name as written', ({ markdown, want }) => {
  expect(html(markdown)).toBe(want);
});

test('shows a formula that KaTeX can not parse as its source, with the error as its title, and the rest of the note', () => {
  const shown = page('Before $\\frac{1$ after\n\n$y$');
  const error = shown.querySelector('.katex-error');

  expect(error?.textContent).toBe('\\frac{1');
  expect(error?.getAttribute('title')).toMatch(/^ParseError: KaTeX parse error: /);
  expect(shown.body.textContent).toContain(' after');
  expect(shown.querySelector('annotation')?.textContent).toBe('y');
});

test.each([
  { name: 'a link', markdown: '$\\href{https://example.com}{x}$' },
  { name: 'a javascript: link', markdown: '$\\href{javascript:alert(1)}{x}$' },
  { name: 'a URL', markdown: '$\\url{https://example.com}$' },
  { name: 'an image', markdown: '$\\includegraphics{https://example.com/a.png}$' },
  { name: 'a class', markdown: '$\\htmlClass{evil}{x}$' },
  { name: 'an ID', markdown: '$\\htmlId{evil}{x}$' },
  { name: 'a style', markdown: '$\\htmlStyle{color: red}{x}$' },
  { name: 'a data attribute', markdown: '$\\htmlData{evil=1}{x}$' },
  { name: 'HTML', markdown: '$<img src=x onerror=alert(1)>$' },
])('keeps $name in math inert', ({ markdown }) => {
  // KaTeX warns about the HTML commands, which LaTeX lacks.
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
  onTestFinished(() => warn.mockRestore());

  const shown = page(markdown);

  expect(shown.querySelector('.katex')).not.toBeNull();
  expect(
    shown.querySelector('a, img, script, .evil, #evil, [href], [src], [onerror], [data-evil], [style*="red"]')
  ).toBeNull();
});
