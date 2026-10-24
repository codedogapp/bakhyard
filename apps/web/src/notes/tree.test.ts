import { expect, test } from 'vitest';

import { type Folder, matches, noteAt, noteHref, noteTree, withoutExtension } from './tree';

const files = [
  'Work/Plans/Q3.md',
  'Work/Plans/Day 10.md',
  'Work/Plans/Day 2.md',
  'Work/standup.md',
  'Work/diagram.png',
  'Inbox.md',
  'archive/Old.md',
  'Attachments/photo.png',
  'about.md',
];

// The tree as an outline: each folder with its count, over its folders and notes, which are indented.
function outline(folder: Folder, depth = 0): string[] {
  const indent = '  '.repeat(depth);
  return [
    ...folder.folders.flatMap((child) => [`${indent}${child.name}/ (${child.count})`, ...outline(child, depth + 1)]),
    ...folder.notes.map((note) => `${indent}${note.title}`),
  ];
}

test('lists the notes in their folders: folders first, then notes, each sorted by name, numbers in numeric order', () => {
  const tree = noteTree(files);

  expect(outline(tree)).toEqual([
    'archive/ (1)',
    '  Old',
    'Work/ (4)',
    '  Plans/ (3)',
    '    Day 2',
    '    Day 10',
    '    Q3',
    '  standup',
    'about',
    'Inbox',
  ]);
  expect(tree.count).toBe(7);
});

test('gives each folder and note its path in the vault', () => {
  const tree = noteTree(files);
  const work = tree.folders[1];
  const plans = work.folders[0];

  expect(tree).toMatchObject({ name: '', path: '' });
  expect(work).toMatchObject({ name: 'Work', path: 'Work' });
  expect(plans).toMatchObject({ name: 'Plans', path: 'Work/Plans' });
  expect(plans.notes[0]).toEqual({ path: 'Work/Plans/Day 2.md', title: 'Day 2', folder: 'Work/Plans' });
  expect(tree.notes[0]).toEqual({ path: 'about.md', title: 'about', folder: '' });
});

test('keeps a note and a folder of the same name apart, and a dot in a title', () => {
  const tree = noteTree(['Work.md', 'Work/v1.2 notes.md']);

  expect(outline(tree)).toEqual(['Work/ (1)', '  v1.2 notes', 'Work']);
});

test.each([
  { name: 'an empty vault', files: [] },
  { name: 'a vault without notes', files: ['photo.png', 'Attachments/scan.pdf'] },
])('gives an empty tree for $name', ({ files }) => {
  expect(noteTree(files)).toEqual({ name: '', path: '', folders: [], notes: [], count: 0 });
});

test.each([
  { path: 'Inbox.md', want: '/notes/Inbox' },
  { path: 'Work/Plans/Day 2.md', want: '/notes/Work/Plans/Day%202' },
  { path: 'Q&A?/50% #1.md', want: '/notes/Q%26A%3F/50%25%20%231' },
  { path: 'Café/Ünï.md', want: '/notes/Caf%C3%A9/%C3%9Cn%C3%AF' },
])('links $path to $want', ({ path, want }) => {
  expect(noteHref(path)).toBe(want);
});

test.each([
  'Inbox.md',
  'Work/Plans/Day 2.md',
  'Q&A?/50% #1.md',
  'Café/Ünï.md',
  'a+b/c=d;e,f@g$h!.md',
  "it's (ok) [1] {2} ~`^|\\.md",
  '日本語/メモ 😀.md',
])('finds %s at its own link', (path) => {
  expect(noteAt(noteHref(path))?.path).toBe(path);
});

test.each([
  { href: '/notes/Inbox', want: { path: 'Inbox.md', title: 'Inbox', folder: '' } },
  { href: '/notes/Work/Plans/Day%202', want: { path: 'Work/Plans/Day 2.md', title: 'Day 2', folder: 'Work/Plans' } },
  { href: '/notes/Work/Plans/Q3.md', want: { path: 'Work/Plans/Q3.md.md', title: 'Q3.md', folder: 'Work/Plans' } },
  // Only noteHref's links are expected, but an unescaped name works too.
  { href: '/notes/Work/To do', want: { path: 'Work/To do.md', title: 'To do', folder: 'Work' } },
])('gives the note at $href', ({ href, want }) => {
  expect(noteAt(href)).toEqual(want);
});

test.each([
  { name: 'the Notes page', href: '/notes' },
  { name: 'the Notes page with a slash', href: '/notes/' },
  { name: 'another page', href: '/notesbook/Inbox' },
  { name: 'an empty name', href: '/notes//Inbox' },
  { name: 'a trailing slash', href: '/notes/Work/' },
  { name: 'a cut escape', href: '/notes/%E0%A4%A' },
  { name: 'an escape of no UTF-8', href: '/notes/%FF' },
  { name: 'a lone %', href: '/notes/50%' },
])('gives no note for $name', ({ href }) => {
  expect(noteAt(href)).toBeUndefined();
});

test('gives the path without ".md"', () => {
  expect(withoutExtension('Work/Plans/Q3.md')).toBe('Work/Plans/Q3');
});

// The titles of the notes that match the filter.
function matching(filter: string): string[] {
  return matches(noteTree(files), filter).map((note) => note.title);
}

test.each([
  { filter: 'q3', want: ['Q3'] },
  { filter: 'DAY', want: ['Day 2', 'Day 10'] },
  { filter: 'plans', want: ['Day 2', 'Day 10', 'Q3'] },
  { filter: 'work q3', want: ['Q3'] },
  { filter: '  day   plans ', want: ['Day 2', 'Day 10'] },
  { filter: 'plans/d', want: ['Day 2', 'Day 10'] },
  { filter: 'o', want: ['Old', 'Day 2', 'Day 10', 'Q3', 'standup', 'about', 'Inbox'] },
  { filter: 'md', want: [] },
  { filter: 'diagram', want: [] },
  { filter: 'q3 inbox', want: [] },
  { filter: '.*', want: [] },
  { filter: 'day (', want: [] },
])('matches "$filter" with $want', ({ filter, want }) => {
  expect(matching(filter)).toEqual(want);
});

test('gives each match its folder', () => {
  expect(matches(noteTree(files), 'standup')).toEqual([{ path: 'Work/standup.md', title: 'standup', folder: 'Work' }]);
});
