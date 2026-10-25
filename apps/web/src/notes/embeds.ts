import { fileUrl } from './api';
import { type Finder, parseWikilink } from './links';

/** The image that an embed shows. */
export interface ImageEmbed {
  src: string;
  /** The path as the embed writes it, such as "x.png". */
  alt: string;
}

// The files that Obsidian shows as images, by their extensions, from its help's "Accepted file formats".
const image = /\.(?:avif|bmp|gif|jpe?g|png|svg|webp)$/i;

/**
 * The image that the embed whose inside is inside, such as "x.png" in "![[x.png]]", shows, from the note find looks
 * from; undefined when its path finds no image. What follows its "|" or "#" is left out, so the image shows at its own
 * size.
 */
export function imageOf(inside: string, find: Finder): ImageEmbed | undefined {
  const { path } = parseWikilink(inside);
  const file = find(path);
  return file !== undefined && image.test(file) ? { src: fileUrl(file), alt: path } : undefined;
}
