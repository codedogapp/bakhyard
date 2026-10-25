import type { Parent, PhrasingContent, Root } from 'mdast';
import { findAndReplace } from 'mdast-util-find-and-replace';

import { imageOf } from './embeds';
import { type Finder, linkTo } from './links';

/** The class of a wikilink's text when it leads nowhere, which the page shows dimmed. */
export const unresolvedClass = 'unresolved';

/**
 * A wikilink that leads nowhere: its text, in a span of the unresolved class. A node of its own, since the replacer
 * merges a text node into the text around it.
 */
interface Unresolved extends Parent {
  type: 'unresolved';
  children: PhrasingContent[];
}

declare module 'mdast' {
  interface PhrasingContentMap {
    unresolved: Unresolved;
  }
  interface RootContentMap {
    unresolved: Unresolved;
  }
}

/**
 * Replaces each wikilink in the note's text, "[[Note]]", with a link to what it names, or its text, dimmed, when it
 * leads nowhere: to a note the vault doesn't have, or to a file that isn't a note. An embed of an image, "![[x.png]]",
 * shows the image; any other embed reads as a wikilink, so "![[Note]]" is a link, for now. Code and a link's text keep
 * their wikilinks.
 */
export function replaceWikilinks(tree: Root, find: Finder) {
  // A wikilink holds no newline, "[" or "]".
  const wikilink = /!?\[\[([^[\]\n]+)\]\]/g;
  findAndReplace(tree, [wikilink, (match: string, inside: string) => toNode(match.startsWith('!'), inside, find)], {
    ignore: ['link', 'linkReference'],
  });
}

// The wikilink's node: an embed's image, or a link; false, which keeps the wikilink as written, for one that names
// nothing.
function toNode(embed: boolean, inside: string, find: Finder): PhrasingContent | false {
  const image = embed ? imageOf(inside, find) : undefined;
  if (image !== undefined) {
    return { type: 'image', url: image.src, alt: image.alt };
  }
  const link = linkTo(inside, find);
  if (link === undefined) {
    return false;
  }
  const children: PhrasingContent[] = [{ type: 'text', value: link.text }];
  if (link.href === undefined) {
    return { type: 'unresolved', children, data: { hName: 'span', hProperties: { className: [unresolvedClass] } } };
  }
  return { type: 'link', url: link.href, children };
}
