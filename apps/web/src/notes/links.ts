import { noteHref } from './tree';

/** A wikilink, as a note writes it between "[[" and "]]": "Note#Heading|Text". */
export interface Wikilink {
  /** The path it names, such as "Work/Plan" or "x.png"; "" when it names the note it's in. */
  path: string;
  /** What follows the first "#": a heading, or "^" and a block's ID; "" when there's none. */
  subpath: string;
  /** The text it shows. */
  text: string;
}

/** A wikilink as a page shows it: its text, and the page it opens, if it leads anywhere. */
export interface Link {
  text: string;
  href?: string;
}

/** The vault path of the file that a link's path names; undefined when the vault has none. */
export type Finder = (path: string) => string | undefined;

/**
 * The wikilink whose inside is inside: the target before the first "|" and the text after it, the path before the
 * target's first "#" and the subpath after it, each trimmed. Without a text, it shows the path and the subpath's parts,
 * leaving out the empty ones, joined by " > ", as Obsidian does: "Note > Heading", or "Heading" for the note's own.
 */
export function parseWikilink(inside: string): Wikilink {
  const [target, text] = cut(inside, '|');
  const [path, subpath] = cut(target, '#');
  const parts = [path, ...subpath.split('#')].map((part) => part.trim()).filter(Boolean);
  return { path, subpath, text: text || parts.join(' > ') };
}

// The text before the first separator and the text after it, each trimmed; "" after it when there's none.
function cut(text: string, separator: string): [string, string] {
  const at = text.indexOf(separator);
  return at < 0 ? [text.trim(), ''] : [text.slice(0, at).trim(), text.slice(at + 1).trim()];
}

/**
 * The finder for the links of the note at source, in a vault of files, as Obsidian's getFirstLinkpathDest finds them:
 * the path with ".md", then as written, since a link to another kind of file names its extension; in the note's folder
 * first, then from the vault's top, then in any folder whose path ends with it, the shortest such path first, then the
 * first by name. In the file's case.
 */
export function finder(files: readonly string[], source: string): Finder {
  const vault = new Set(files);
  const folder = source.slice(0, source.lastIndexOf('/') + 1);
  return (path) => {
    const names = [`${path}.md`, path];
    const exact = [...names.map((name) => folder + name), ...names].find((name) => vault.has(name));
    return exact ?? shortest(files.filter((file) => names.some((name) => file.endsWith(`/${name}`))));
  };
}

function shortest(paths: string[]): string | undefined {
  return paths.toSorted((a, b) => a.length - b.length || (a < b ? -1 : 1))[0];
}

/** The ID of a heading whose text is text, which a link finds in any case: its words in lower case, joined by "-". */
export function headingId(text: string): string {
  return text.trim().toLowerCase().replaceAll(/\s+/g, '-');
}

// rehype-sanitize prefixes each ID with this, its default, and leaves links alone, so a link to a heading names it too.
const idPrefix = 'user-content-';

/**
 * The wikilink whose inside is inside, from the note find looks from: a link to a note's page, or to a heading of the
 * note itself; no href when the vault has no note at its path. Undefined when it names neither a path nor a heading.
 */
export function linkTo(inside: string, find: Finder): Link | undefined {
  const { path, subpath, text } = parseWikilink(inside);
  if (path !== '') {
    const file = find(path);
    return { text, href: file?.endsWith('.md') ? noteHref(file) : undefined };
  }
  return subpath === '' ? undefined : { text, href: `#${idPrefix}${encodeURIComponent(headingId(subpath))}` };
}
