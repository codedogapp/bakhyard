import { type App, appHref, apps } from './apps';
import { isCurrent, moreHref } from './nav';

/** What a path shows. */
export type Page =
  | { kind: 'home' }
  | { kind: 'notes' }
  | { kind: 'app'; app: App }
  | { kind: 'more' }
  | { kind: 'missing' };

const notesHref = appHref('notes');

/** The page at path. The Notes app's path shows the notes; an app's other paths show its page, as the nav marks it. */
export function pageAt(path: string): Page {
  if (path === '/') {
    return { kind: 'home' };
  }
  if (path === notesHref || path === `${notesHref}/`) {
    return { kind: 'notes' };
  }
  if (isCurrent(moreHref, path)) {
    return { kind: 'more' };
  }
  const app = apps.find((candidate) => isCurrent(appHref(candidate.id), path));
  return app ? { kind: 'app', app } : { kind: 'missing' };
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
    case 'app':
      return `${page.app.name} · ${siteName}`;
    case 'more':
      return `More · ${siteName}`;
    case 'missing':
      return `Page not found · ${siteName}`;
  }
}
