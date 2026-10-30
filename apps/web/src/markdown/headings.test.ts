import type { Heading } from 'mdast';
import remarkParse from 'remark-parse';
import { unified } from 'unified';
import { expect, test } from 'vitest';

import { headingsIn, shiftHeadings } from './headings';

const parser = unified().use(remarkParse);

// The text of each heading.
function texts(headings: Heading[]): string[] {
  return headings.map(({ children: [first] }) => (first?.type === 'text' ? first.value : ''));
}

test.each([
  {
    name: 'an H1 moves every heading one level down, an h6 staying an h6',
    markdown: '# 1\n## 2\n### 3\n#### 4\n##### 5\n###### 6',
    want: [2, 3, 4, 5, 6, 6],
  },
  { name: 'a quoted or listed H1 counts too', markdown: '## 2\n> # 1\n- ### 3', want: [3, 2, 4] },
  { name: 'with no H1 the levels stay', markdown: '## 2\n### 3\n###### 6', want: [2, 3, 6] },
  { name: 'with no heading nothing changes', markdown: 'Text', want: [] },
])('shiftHeadings: $name', ({ markdown, want }) => {
  const tree = parser.parse(markdown);

  shiftHeadings(tree);

  expect(headingsIn(tree).map((heading) => heading.depth)).toEqual(want);
});

test('headingsIn finds the headings in the order they come, quoted and listed ones too', () => {
  const tree = parser.parse('## A\n> ### B\n- #### C\n\nText\n\n## D');

  expect(texts(headingsIn(tree))).toEqual(['A', 'B', 'C', 'D']);
});

test('headingsIn gives a heading itself', () => {
  const [heading] = headingsIn(parser.parse('## A'));

  expect(heading && headingsIn(heading)).toEqual([heading]);
});
