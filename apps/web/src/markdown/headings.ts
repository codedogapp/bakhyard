import type { Heading, Nodes, Root } from 'mdast';

/**
 * Moves every heading one level down when an H1 is left, so the page head's h1 is the page's only one, and keeps the
 * levels when none is, so none is skipped; an h6 stays an h6 (decision 96).
 */
export function shiftHeadings(tree: Root) {
  const headings = headingsIn(tree);
  if (!headings.some((heading) => heading.depth === 1)) {
    return;
  }
  for (const heading of headings) {
    heading.depth = Math.min(heading.depth + 1, 6) as Heading['depth'];
  }
}

/** The headings in the node, itself included, in the order they come. */
export function headingsIn(node: Nodes): Heading[] {
  if (node.type === 'heading') {
    return [node];
  }
  return 'children' in node ? node.children.flatMap(headingsIn) : [];
}
