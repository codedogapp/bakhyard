import { expect, test } from 'vitest';

import { imageOf } from './embeds';
import { finder } from './links';

const extensions = ['avif', 'bmp', 'gif', 'jpeg', 'jpg', 'png', 'svg', 'webp'];
const vault = [
  'Songs/Em.md',
  'Songs/covers/Em.png',
  'Songs/cover.png',
  'Art/cover.png',
  'Art/Photo.JPG',
  'My art/a b%?.png',
  'Docs/score.pdf',
  'Docs/png',
  'Songs/chart.png.md',
  'Docs/chart.png',
  ...extensions.map((extension) => `Art/x.${extension}`),
];
// The images that the embeds of the note "Songs/Em" show.
const find = finder(vault, 'Songs/Em.md');

test.each(extensions)('shows an image whose extension is %s', (extension) => {
  expect(imageOf(`x.${extension}`, find)).toEqual({
    src: `/api/notes/files/Art/x.${extension}`,
    alt: `x.${extension}`,
  });
});

test.each([
  { name: 'an extension in another case', inside: 'Photo.JPG', src: '/api/notes/files/Art/Photo.JPG' },
  { name: 'the note’s folder first', inside: 'cover.png', src: '/api/notes/files/Songs/cover.png' },
  { name: 'any folder', inside: 'Em.png', src: '/api/notes/files/Songs/covers/Em.png' },
  { name: 'a path', inside: 'Art/cover.png', src: '/api/notes/files/Art/cover.png' },
  { name: 'names escaped', inside: 'a b%?.png', src: '/api/notes/files/My%20art/a%20b%25%3F.png' },
])('finds the image as a link finds its file: $name', ({ inside, src }) => {
  expect(imageOf(inside, find)).toEqual({ src, alt: inside });
});

test.each([
  { name: 'a width', inside: 'cover.png|300' },
  { name: 'a width and a height', inside: 'cover.png|300x200' },
  { name: 'text', inside: 'cover.png|The cover' },
  { name: 'a subpath', inside: 'cover.png#outline' },
  { name: 'spaces', inside: ' cover.png | 300 ' },
])('shows the image at its own size, whatever follows its path: $name', ({ inside }) => {
  expect(imageOf(inside, find)).toEqual({ src: '/api/notes/files/Songs/cover.png', alt: 'cover.png' });
});

test.each([
  { name: 'a missing image', inside: 'nothing.png' },
  { name: 'a note', inside: 'Em' },
  { name: 'a note whose name ends like an image’s', inside: 'chart.png' },
  { name: 'a file of another kind', inside: 'score.pdf' },
  { name: 'a file named like an extension', inside: 'png' },
  { name: 'a heading', inside: '#Chords' },
])('shows no image for $name', ({ inside }) => {
  expect(imageOf(inside, find)).toBeUndefined();
});
