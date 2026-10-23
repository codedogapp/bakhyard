import { type App, appHref, apps } from './apps';
import { isCurrent, moreHref } from './nav';

/** What a path shows. */
export type Page = { kind: 'home' } | { kind: 'app'; app: App } | { kind: 'more' } | { kind: 'missing' };

/** The page at path. An app's own pages show the app's page, as the nav marks it on them. */
export function pageAt(path: string): Page {
  if (path === '/') {
    return { kind: 'home' };
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
    case 'app':
      return `${page.app.name} · ${siteName}`;
    case 'more':
      return `More · ${siteName}`;
    case 'missing':
      return `Page not found · ${siteName}`;
  }
}
