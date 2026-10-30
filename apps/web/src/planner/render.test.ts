import { expect, test } from 'vitest';

import { renderPlan } from './render';

test('renders GFM: tables with their alignment, strikethrough, autolinks and tasks', () => {
  const markdown = [
    '| Left | Right |',
    '| :--- | ---: |',
    '| a | b |',
    '',
    '~~gone~~ and www.example.com',
    '',
    '- [ ] open',
    '- [x] done',
  ].join('\n');

  const out = renderPlan(markdown);

  expect(out).toContain('<th align="left">Left</th>\n<th align="right">Right</th>');
  expect(out).toContain('<p><del>gone</del> and <a href="http://www.example.com">www.example.com</a></p>');
  expect(out).toContain('<li class="task-list-item"><input type="checkbox" disabled> open</li>');
  expect(out).toContain('<li class="task-list-item"><input type="checkbox" checked disabled> done</li>');
});

test('breaks the line at each newline, as a note does', () => {
  expect(renderPlan('One\nTwo')).toBe('<p>One<br>\nTwo</p>');
});

test.each([
  { name: 'raw HTML', markdown: '<div onclick="alert(1)">Hi</div>\n\nText', want: '<p>Text</p>' },
  { name: 'a script', markdown: '<script>alert(1)</script>\n\nText', want: '<p>Text</p>' },
  { name: 'inline HTML', markdown: 'Text <img src=x onerror=alert(1)>', want: '<p>Text </p>' },
  { name: "a script's link", markdown: '[Go](javascript:alert(1))', want: '<p><a>Go</a></p>' },
])('drops $name', ({ markdown, want }) => {
  expect(renderPlan(markdown)).toBe(want);
});

test.each([
  {
    name: 'an H1 moves every heading one level down',
    markdown: '# Trip\n## Days\n###### Six',
    want: '<h2>Trip</h2>\n<h3>Days</h3>\n<h6>Six</h6>',
  },
  { name: 'with no H1 the levels stay', markdown: '## Days\n### Rome', want: '<h2>Days</h2>\n<h3>Rome</h3>' },
])('$name', ({ markdown, want }) => {
  expect(renderPlan(markdown)).toBe(want);
});
