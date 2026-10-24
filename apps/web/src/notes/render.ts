import type { Heading, Nodes, Root } from 'mdast';
import { toString as plainText } from 'mdast-util-to-string';
import rehypeSanitize from 'rehype-sanitize';
import rehypeStringify from 'rehype-stringify';
import remarkBreaks from 'remark-breaks';
import remarkFrontmatter from 'remark-frontmatter';
import remarkGfm from 'remark-gfm';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import { unified } from 'unified';

import { noteProperties, type Property } from './properties';

/** A note as its page shows it. */
export interface RenderedNote {
  /** The rows of its Properties block. */
  properties: Property[];
  /** Its Markdown as HTML that's safe to show, without an h1. */
  html: string;
}

// Markdown as Obsidian reads it: GFM, a line break for each newline, and YAML frontmatter. Raw HTML is dropped, and
// the HTML is sanitized with GitHub's rules. Frozen, so no one can change it.
const processor = unified()
  .use(remarkParse)
  .use(remarkFrontmatter)
  .use(remarkGfm)
  .use(remarkBreaks)
  .use(remarkRehype)
  .use(rehypeSanitize)
  .use(rehypeStringify)
  .freeze();

/** The note titled title, whose file holds markdown, as its page shows it, under a head whose h1 is the title. */
export function renderNote(markdown: string, title: string): RenderedNote {
  const tree = processor.parse(markdown);
  const properties = takeProperties(tree);
  dropTitle(tree, title);
  shiftHeadings(tree);
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

// Drops the note's first block when it's an H1 that repeats the title, which the head shows.
function dropTitle(tree: Root, title: string) {
  const [first] = tree.children;
  if (first?.type === 'heading' && first.depth === 1 && plainText(first) === title) {
    tree.children.shift();
  }
}

// Moves every heading one level down when an H1 is left, so the head's h1 is the page's only one; an h6 stays an h6.
function shiftHeadings(tree: Root) {
  const headings = headingsIn(tree);
  if (!headings.some((heading) => heading.depth === 1)) {
    return;
  }
  for (const heading of headings) {
    heading.depth = Math.min(heading.depth + 1, 6) as Heading['depth'];
  }
}

function headingsIn(node: Nodes): Heading[] {
  if (node.type === 'heading') {
    return [node];
  }
  return 'children' in node ? node.children.flatMap(headingsIn) : [];
}
