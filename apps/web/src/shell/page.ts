import { type Note, noteAt } from '../notes/tree';
import { type App, appHref, apps } from './apps';
import { isCurrent, moreHref } from './nav';

/** What a path shows. */
export type Page =
  | { kind: 'home' }
  | { kind: 'notes' }
  | { kind: 'note'; note: Note }
  | { kind: 'app'; app: App }
  | { kind: 'more' }
  | { kind: 'missing' };

const notesHref = appHref('notes');

/**
 * The page at path. The Notes app's path shows the notes, and the paths under it their notes; another app's paths show
 * its page, as the nav marks it.
 */
export function pageAt(path: string): Page {
  if (path === '/') {
    return { kind: 'home' };
  }
  if (path === notesHref || path === `${notesHref}/`) {
    return { kind: 'notes' };
  }
  if (isCurrent(notesHref, path)) {
    return notePage(path);
  }
  if (isCurrent(moreHref, path)) {
    return { kind: 'more' };
  }
  const app = apps.find((candidate) => isCurrent(appHref(candidate.id), path));
  return app ? { kind: 'app', app } : { kind: 'missing' };
}

// The page of the note at path; missing when no note's page can be at it.
function notePage(path: string): Page {
  const note = noteAt(path);
  return note ? { kind: 'note', note } : { kind: 'missing' };
}

const siteName = 'Bakhyard';

/** The document's title on every path while the user is signed out. */
export const signedOutTitle = `Signed out · ${siteName}`;

/** The document's title on the page. */
export function pageTitle(page: Page): string {
  switch (page.kind) {
    case 'home':
      return siteName;
    case 'notes':
      return `Notes · ${siteName}`;
    case 'note':
      return `${page.note.title} · ${siteName}`;
    case 'app':
      return `${page.app.name} · ${siteName}`;
    case 'more':
      return `More · ${siteName}`;
    case 'missing':
      return `Page not found · ${siteName}`;
  }
}
