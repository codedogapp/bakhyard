import { appHref, apps } from './apps';

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
