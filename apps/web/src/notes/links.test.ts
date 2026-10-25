import { expect, test } from 'vitest';

import { finder, headingId, linkTo, parseWikilink } from './links';

test.each([
  { inside: 'Note', want: { path: 'Note', subpath: '', text: 'Note' } },
  { inside: 'Note|Text', want: { path: 'Note', subpath: '', text: 'Text' } },
  { inside: 'Note#Heading', want: { path: 'Note', subpath: 'Heading', text: 'Note > Heading' } },
  { inside: '#Heading', want: { path: '', subpath: 'Heading', text: 'Heading' } },
  { inside: 'A#B#C', want: { path: 'A', subpath: 'B#C', text: 'A > B > C' } },
  { inside: 'Note#^block-1', want: { path: 'Note', subpath: '^block-1', text: 'Note > ^block-1' } },
  { inside: 'Folder/Note', want: { path: 'Folder/Note', subpath: '', text: 'Folder/Note' } },
  { inside: 'Note.md', want: { path: 'Note.md', subpath: '', text: 'Note.md' } },
  { inside: ' Note # Heading | Text ', want: { path: 'Note', subpath: 'Heading', text: 'Text' } },
  { inside: 'A # B # C', want: { path: 'A', subpath: 'B # C', text: 'A > B > C' } },
  { inside: 'Note|', want: { path: 'Note', subpath: '', text: 'Note' } },
  { inside: 'Note|a|b', want: { path: 'Note', subpath: '', text: 'a|b' } },
  { inside: 'Note#H|a#b', want: { path: 'Note', subpath: 'H', text: 'a#b' } },
  { inside: 'Note#', want: { path: 'Note', subpath: '', text: 'Note' } },
])('reads [[$inside]]', ({ inside, want }) => {
  expect(parseWikilink(inside)).toEqual(want);
});

const vault = [
  'Plan.md',
  'Daily/Plan.md',
  'Work/Plan.md',
  'Work/Old/Plan.md',
  'Songs/Em.md',
  'Deep/Down/Here/Far.md',
  'Near/Far.md',
  'B/Same.md',
  'A/Same.md',
  'Twin.png.md',
  'Twin.png',
  'Attachments/x.png',
];

test.each([
  { name: 'the linking note’s folder first', from: 'Daily/To do.md', path: 'Plan', want: 'Daily/Plan.md' },
  { name: 'then the vault’s top', from: 'Songs/Em.md', path: 'Plan', want: 'Plan.md' },
  { name: 'a path from the vault’s top', from: 'Daily/To do.md', path: 'Work/Plan', want: 'Work/Plan.md' },
  { name: 'then any folder', from: 'Daily/To do.md', path: 'Em', want: 'Songs/Em.md' },
  { name: 'a path in any folder', from: 'Daily/To do.md', path: 'Old/Plan', want: 'Work/Old/Plan.md' },
  { name: 'the shortest path of several', from: 'Daily/To do.md', path: 'Far', want: 'Near/Far.md' },
  { name: 'the first by name of the shortest', from: 'Daily/To do.md', path: 'Same', want: 'A/Same.md' },
  { name: 'a note with its extension', from: 'Daily/To do.md', path: 'Songs/Em.md', want: 'Songs/Em.md' },
  { name: 'a note before a file as written', from: 'Daily/To do.md', path: 'Twin.png', want: 'Twin.png.md' },
  { name: 'another file by its extension', from: 'Daily/To do.md', path: 'x.png', want: 'Attachments/x.png' },
  { name: 'nothing in another case', from: 'Daily/To do.md', path: 'plan', want: undefined },
  { name: 'nothing without its extension', from: 'Daily/To do.md', path: 'x', want: undefined },
  { name: 'nothing whose name only ends with it', from: 'Daily/To do.md', path: 'lan', want: undefined },
  { name: 'nothing that is not there', from: 'Daily/To do.md', path: 'Nothing', want: undefined },
])('finds $name', ({ from, path, want }) => {
  expect(finder(vault, from)(path)).toBe(want);
});

test.each([
  { text: 'Chords', want: 'chords' },
  { text: ' Café  au\tlait ', want: 'café-au-lait' },
  { text: 'C# major', want: 'c#-major' },
  { text: ' ', want: '' },
])('gives the heading "$text" the ID "$want"', ({ text, want }) => {
  expect(headingId(text)).toBe(want);
});

test.each([
  { inside: 'Em', want: { text: 'Em', href: '/notes/Songs/Em' } },
  { inside: 'Em#Chords|E minor', want: { text: 'E minor', href: '/notes/Songs/Em' } },
  { inside: 'Work/Plan', want: { text: 'Work/Plan', href: '/notes/Work/Plan' } },
  { inside: '#Café au lait', want: { text: 'Café au lait', href: '#user-content-caf%C3%A9-au-lait' } },
  { inside: '#C# major', want: { text: 'C > major', href: '#user-content-c%23-major' } },
  { inside: 'Nothing|Shown', want: { text: 'Shown', href: undefined } },
  { inside: 'x.png', want: { text: 'x.png', href: undefined } },
  { inside: '#', want: undefined },
  { inside: ' | Text', want: undefined },
])('links [[$inside]]', ({ inside, want }) => {
  expect(linkTo(inside, finder(vault, 'Daily/To do.md'))).toEqual(want);
});
