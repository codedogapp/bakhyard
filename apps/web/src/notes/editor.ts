import type { Attachment } from 'svelte/attachments';
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
import { markdownKeymap, markdownLanguage, pasteURLAsLink } from '@codemirror/lang-markdown';
import { yamlFrontmatter } from '@codemirror/lang-yaml';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { EditorView, keymap } from '@codemirror/view';
import { tags } from '@lezer/highlight';

// A note's parts in ink only, as §7 asks: weight, slant and the muted ink set them apart, never a colour.
const inkOnly = HighlightStyle.define([
  { tag: [tags.heading, tags.strong], fontWeight: '600' },
  { tag: tags.emphasis, fontStyle: 'italic' },
  { tag: tags.strikethrough, textDecoration: 'line-through' },
  { tag: tags.link, textDecoration: 'underline' },
  { tag: tags.monospace, fontFamily: 'var(--font-mono)' },
  {
    tag: [tags.processingInstruction, tags.meta, tags.url, tags.quote, tags.contentSeparator],
    color: 'var(--ink-muted)',
  },
]);

/**
 * Shows text, a note's Markdown, in a CodeMirror editor in the element, and gives the editor the focus (decision 401):
 * Markdown as GitHub writes it, after the note's YAML frontmatter, with its lines wrapped and undo. Enter carries a list
 * or a quote on to the next line, and a URL pasted over text links it. Tab leaves the editor, as it leaves any field.
 * The editor goes with the element.
 *
 * The bare Markdown language, without lang-markdown's markdown(), which would add HTML's, CSS's and JavaScript's for
 * the HTML in a note: 150 kB more in the editor's chunk.
 */
export function noteEditor(text: string): Attachment<HTMLElement> {
  return (parent) => {
    const view = new EditorView({
      parent,
      doc: text,
      extensions: [
        history(),
        keymap.of([...markdownKeymap, ...defaultKeymap, ...historyKeymap]),
        yamlFrontmatter({ content: markdownLanguage }),
        pasteURLAsLink,
        syntaxHighlighting(inkOnly),
        EditorView.lineWrapping,
        EditorView.contentAttributes.of({ 'aria-label': 'Markdown' }),
      ],
    });
    view.focus();
    return () => view.destroy();
  };
}
