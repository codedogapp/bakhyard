import { describe, expect, it } from 'vitest';

import page from '../index.html?raw';
import manifestText from '../public/manifest.json?raw';

interface Icon {
  href: string;
  sizes: string | null;
}

interface Manifest {
  icons: { src: string; sizes: string }[];
}

// public/ is the root of the built app, so "../public/icon.svg" is served as "/icon.svg".
const served = (file: string) => file.replace('../public', '');
const files = Object.keys(import.meta.glob('../public/*')).map(served);
const pngs = new Map(
  Object.entries(import.meta.glob<string>('../public/*.png', { query: '?inline', import: 'default', eager: true })).map(
    ([file, dataUrl]) => [served(file), dataUrl]
  )
);

const links = [
  ...new DOMParser()
    .parseFromString(page, 'text/html')
    .querySelectorAll('link[rel="icon"], link[rel="apple-touch-icon"], link[rel="manifest"]'),
];
const pageIcons: Icon[] = links.map((link) => ({
  href: link.getAttribute('href') ?? '',
  sizes: link.getAttribute('sizes'),
}));
const manifestIcons: Icon[] = (JSON.parse(manifestText) as Manifest).icons.map(({ src, sizes }) => ({
  href: src,
  sizes,
}));
const icons = [...pageIcons, ...manifestIcons];

// A PNG's width and height are the first fields of its IHDR chunk, after the 8-byte signature and the chunk's
// length and type.
function pngSize(dataUrl: string): string {
  const bytes = Uint8Array.from(atob(dataUrl.split(',')[1]), (char) => char.charCodeAt(0));
  const view = new DataView(bytes.buffer);

  return `${view.getUint32(16)}x${view.getUint32(20)}`;
}

describe('the app icons', () => {
  it('names a favicon, a Home Screen icon and a manifest in the page', () => {
    expect(new Set(links.map((link) => link.getAttribute('rel')))).toEqual(
      new Set(['icon', 'apple-touch-icon', 'manifest'])
    );
  });

  it.each(icons)('serves $href from public/', ({ href }) => {
    expect(files).toContain(href);
  });

  it.each(icons.filter((icon) => icon.sizes !== null))('gives $href the size $sizes', ({ href, sizes }) => {
    expect(pngSize(pngs.get(href) ?? '')).toBe(sizes);
  });
});
