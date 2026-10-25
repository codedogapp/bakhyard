import type { Parent, PhrasingContent, Root } from 'mdast';
import { findAndReplace } from 'mdast-util-find-and-replace';

import { type Finder, type Link, linkTo } from './links';

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
 * leads nowhere: to a note the vault doesn't have, or to a file that isn't a note. An embed, "![[Note]]", is a link
 * too, for now, and so leads nowhere when it names an image. Code and a link's text keep their wikilinks.
 */
export function replaceWikilinks(tree: Root, find: Finder) {
  // A wikilink holds no newline, "[" or "]".
  const wikilink = /!?\[\[([^[\]\n]+)\]\]/g;
  findAndReplace(tree, [wikilink, (_: string, inside: string) => toNode(linkTo(inside, find))], {
    ignore: ['link', 'linkReference'],
  });
}

// The link's node; false, which keeps the wikilink as written, for one that names nothing.
function toNode(link: Link | undefined): PhrasingContent | false {
  if (link === undefined) {
    return false;
  }
  const children: PhrasingContent[] = [{ type: 'text', value: link.text }];
  if (link.href === undefined) {
    return { type: 'unresolved', children, data: { hName: 'span', hProperties: { className: [unresolvedClass] } } };
  }
  return { type: 'link', url: link.href, children };
}
