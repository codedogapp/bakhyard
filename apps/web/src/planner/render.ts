import rehypeSanitize from 'rehype-sanitize';
import rehypeStringify from 'rehype-stringify';
import remarkBreaks from 'remark-breaks';
import remarkGfm from 'remark-gfm';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import { unified } from 'unified';

import { shiftHeadings } from '../markdown/headings';

// GFM, as the agent writes it, with a line break for each newline as on a note's page, since the plan goes to the
// vault; no frontmatter or math, which a plan doesn't have. Raw HTML is dropped, and the HTML is sanitized with
// GitHub's rules. Frozen, so no one can change it.
const processor = unified()
  .use(remarkParse)
  .use(remarkGfm)
  .use(remarkBreaks)
  .use(remarkRehype)
  .use(rehypeSanitize)
  .use(rehypeStringify)
  .freeze();

/** The plan, whose markdown the agent wrote, as HTML that's safe to show under a head whose h1 is the page's. */
export function renderPlan(markdown: string): string {
  const tree = processor.parse(markdown);
  shiftHeadings(tree);
  return processor.stringify(processor.runSync(tree));
}
