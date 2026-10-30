import type { Conversation } from '../src/planner/conversation';

/** A made-up conversation of the demo's newest plan, as its stream first sends it while the agent works. */
export const conversation: Conversation = {
  messages: [
    { id: 'u1', from: 'user', text: 'Add a dark theme to the docs site.\nKeep the light one as it is.' },
    { id: 'a1', from: 'assistant', text: 'Should the code blocks and the diagrams get dark colours too?' },
    { id: 'u2', from: 'user', text: 'Yes, all of it.' },
    {
      id: 'a2',
      from: 'assistant',
      text: "Here's what I found:\n\n1. The colours live in one file, styles/colours.css.\n2. The code blocks have a theme of their own.\n3. The diagrams are SVG files with fixed colours.",
    },
  ],
  writing: [],
  working: true,
  failure: '',
};

/** A stream's body: each event with its data, after the time the browser should wait before it opens it again. */
export function stream(retryMs: number, ...events: [name: string, data: unknown][]): string {
  const sent = events.map(([name, data]) => `event: ${name}\ndata: ${JSON.stringify(data)}\n\n`);
  return [`retry: ${retryMs}\n\n`, ...sent].join('');
}
