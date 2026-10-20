import { type App, appHref, apps } from './apps';

interface NavItem {
  label: string;
  href: string;
}

/** Home, then every app. */
export const navItems: readonly NavItem[] = [
  { label: 'Home', href: '/' },
  ...apps.map((app) => ({ label: app.label, href: appHref(app.id) })),
];

/** Whether the page at path belongs to the item at href, its own pages included. Home is "/" alone. */
export function isCurrent(href: string, path: string): boolean {
  return href === '/' ? path === '/' : path === href || path.startsWith(`${href}/`);
}

/** The More tab's page. */
export const moreHref = '/more';
/** Home, Notes, Todos and PRs get a tab of their own. */
const ownTabs = 4;

/** The bottom tab bar: the first nav items, then More, which lists the rest (plan §7). */
export const tabItems: readonly NavItem[] = [...navItems.slice(0, ownTabs), { label: 'More', href: moreHref }];

/** The apps without a tab of their own, which More lists. */
export const moreApps: readonly App[] = apps.filter((app) => !tabItems.some((item) => item.href === appHref(app.id)));

/** How a tab is current, as its aria-current: "page" for its own pages, "true" for a page it only stands for. */
type TabCurrent = 'page' | 'true';

/** The tab at href is current on its own pages, as in the header; More is too on the apps it lists, as on iOS. */
export function tabCurrent(href: string, path: string): TabCurrent | undefined {
  if (isCurrent(href, path)) {
    return 'page';
  }
  if (href === moreHref && moreApps.some((app) => isCurrent(appHref(app.id), path))) {
    return 'true';
  }
  return undefined;
}
