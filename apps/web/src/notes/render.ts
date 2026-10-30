import 'katex/dist/katex.min.css';

import type { Root } from 'mdast';
import { toString as plainText } from 'mdast-util-to-string';
import rehypeKatex from 'rehype-katex';
import rehypeSanitize, { defaultSchema, type Options as Schema } from 'rehype-sanitize';
import rehypeStringify from 'rehype-stringify';
import remarkBreaks from 'remark-breaks';
import remarkFrontmatter from 'remark-frontmatter';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import { unified } from 'unified';

import { headingsIn, shiftHeadings } from '../markdown/headings';
import { type Finder, finder, headingId, linkTo } from './links';
import { noteProperties, type Property } from './properties';
import type { Note } from './tree';
import { replaceWikilinks, unresolvedClass } from './wikilinks';

/** A note as its page shows it. */
export interface RenderedNote {
  /** The rows of its Properties block. */
  properties: Property[];
  /** Its Markdown as HTML that's safe to show, without an h1. */
  html: string;
}

// GitHub's rules, and one more class, a wikilink's that leads nowhere, which only the renderer adds: raw HTML is dropped.
const schema: Schema = {
  ...defaultSchema,
  attributes: { ...defaultSchema.attributes, span: [['className', unresolvedClass]] },
};

// Markdown as Obsidian reads it: GFM, a line break for each newline, YAML frontmatter, and math between `$` or `$$`.
// Raw HTML is dropped, and the HTML is sanitized with the schema above. KaTeX renders the math after that, since the
// schema would drop KaTeX's styles and MathML. KaTeX's `trust` stays off, so a formula can't add a link, an image or
// HTML attributes of its own. Frozen, so no one can change it.
const processor = unified()
  .use(remarkParse)
  .use(remarkFrontmatter)
  .use(remarkGfm)
  .use(remarkMath)
  .use(remarkBreaks)
  .use(remarkRehype)
  .use(rehypeSanitize, schema)
  .use(rehypeKatex)
  .use(rehypeStringify)
  .freeze();

/**
 * The note, whose file holds markdown, as its page shows it, under a head whose h1 is its title, its wikilinks found
 * among the vault's files.
 */
export function renderNote(markdown: string, note: Note, files: readonly string[]): RenderedNote {
  const tree = processor.parse(markdown);
  const find = finder(files, note.path);
  const properties = takeProperties(tree).map((property) => linkProperty(property, find));
  dropTitle(tree, note.title);
  shiftHeadings(tree);
  replaceWikilinks(tree, find);
  nameHeadings(tree);
  return { properties, html: processor.stringify(processor.runSync(tree)) };
}

// Takes the note's frontmatter, its first block, out of the tree, as its Properties block.
function takeProperties(tree: Root): Property[] {
  const [first] = tree.children;
  if (first?.type !== 'yaml') {
    return [];
  }
  tree.children.shift();
  return noteProperties(first.value);
}

// A property whose value is a wikilink alone, such as album: "[[X]]", shows as the note's wikilinks do.
function linkProperty(property: Property, find: Finder): Property {
  const inside = typeof property.value === 'string' ? /^\[\[([^[\]\n]+)\]\]$/.exec(property.value)?.[1] : undefined;
  const link = inside === undefined ? undefined : linkTo(inside, find);
  return link ? { ...property, value: link } : property;
}

// Drops the note's first block when it's an H1 that repeats the title, which the head shows.
function dropTitle(tree: Root, title: string) {
  const [first] = tree.children;
  if (first?.type === 'heading' && first.depth === 1 && plainText(first) === title) {
    tree.children.shift();
  }
}

// Gives each heading an ID from its text, for the links to it, unless an earlier heading has that ID: a link leads to
// the first, as in Obsidian.
function nameHeadings(tree: Root) {
  const named = new Set<string>();
  for (const heading of headingsIn(tree)) {
    const id = headingId(plainText(heading));
    if (id !== '' && !named.has(id)) {
      named.add(id);
      heading.data = { hProperties: { id } };
    }
  }
}
