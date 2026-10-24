import { afterEach, expect, test, vi } from 'vitest';

import { noteProperties } from './properties';

// The tests run yaml's Node build, which warns through process.emitWarning; the browser's warns through console.warn.
declare const process: { emitWarning(warning: unknown): void };

afterEach(() => {
  vi.restoreAllMocks();
});

test("gives the frontmatter's keys in their order, without the title, and a list's values as chips", () => {
  const source = [
    'title: Ignored',
    'tags:',
    '  - guitar',
    '  - songs/acoustic',
    'author:',
    '  - "[[Someone]]"',
    'album: First Light',
    'created: 2024-03-01',
    'at: 12:30',
  ].join('\n');

  expect(noteProperties(source)).toEqual([
    { key: 'tags', value: ['guitar', 'songs/acoustic'] },
    { key: 'author', value: ['[[Someone]]'] },
    { key: 'album', value: 'First Light' },
    { key: 'created', value: '2024-03-01' },
    { key: 'at', value: '12:30' },
  ]);
});

test('keeps keys that look like numbers in the order the file gives them', () => {
  expect(noteProperties('b: x\n2: y\n1: z').map((property) => property.key)).toEqual(['b', '2', '1']);
});

test('gives numbers and booleans as text, in a list too', () => {
  expect(noteProperties('rating: 4.5\ndone: true\ncount: 0\nmixed: [1, false, a]')).toEqual([
    { key: 'rating', value: '4.5' },
    { key: 'done', value: 'true' },
    { key: 'count', value: '0' },
    { key: 'mixed', value: ['1', 'false', 'a'] },
  ]);
});

test('leaves out the keys whose value is null, empty or a map, and what a list holds that is none of those', () => {
  const source = [
    'nothing:',
    'tilde: ~',
    'empty: ""',
    'none: []',
    'blanks: ["", null]',
    'map: { a: 1 }',
    'nested: [[a], { b: 2 }, c]',
    '"": no key',
    '? [a, b]',
    ': complex key',
    'kept: yes',
  ].join('\n');

  expect(noteProperties(source)).toEqual([
    { key: 'nested', value: ['c'] },
    { key: 'kept', value: 'yes' },
  ]);
});

test.each([
  { name: 'YAML that does not parse', source: 'tags: [a, b' },
  { name: 'a key twice', source: 'a: 1\na: 2' },
  { name: 'an alias to no anchor', source: 'a: *missing' },
  { name: 'a list', source: '- a\n- b' },
  { name: 'a string', source: 'just text' },
  { name: 'nothing', source: '' },
])('gives no properties for $name', ({ source }) => {
  expect(noteProperties(source)).toEqual([]);
});

test('gives no properties for an alias bomb, rather than expand it', () => {
  const lines = ['a: &a [x, x, x, x, x, x, x, x, x, x]'];
  for (let level = 1; level < 8; level++) {
    const before = String.fromCharCode(96 + level);
    const name = String.fromCharCode(97 + level);
    lines.push(`${name}: &${name} [${Array(10).fill(`*${before}`).join(', ')}]`);
  }

  expect(noteProperties(lines.join('\n'))).toEqual([]);
});

test('logs nothing for a tag it does not know or YAML that does not parse', () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
  const log = vi.spyOn(console, 'log').mockImplementation(() => undefined);
  const emit = vi.spyOn(process, 'emitWarning').mockImplementation(() => undefined);

  expect(noteProperties('when: !date 2024-03-01')).toEqual([{ key: 'when', value: '2024-03-01' }]);
  expect(noteProperties('tags: [a')).toEqual([]);
  expect(warn).not.toHaveBeenCalled();
  expect(log).not.toHaveBeenCalled();
  expect(emit).not.toHaveBeenCalled();
});
