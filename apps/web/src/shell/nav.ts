interface NavItem {
  label: string;
  href: string;
}

// Each app's page is named after its service (plan §4.3).
export const navItems: readonly NavItem[] = [
  { label: 'Home', href: '/' },
  { label: 'Notes', href: '/notes' },
  { label: 'Todos', href: '/todos' },
  { label: 'PRs', href: '/pulls' },
  { label: 'Time', href: '/time' },
  { label: 'GitHub', href: '/github' },
  { label: 'Games', href: '/games' },
  { label: 'Planner', href: '/planner' },
];

/** Whether the page at path belongs to the item at href, its own pages included. Home is "/" alone. */
export function isCurrent(href: string, path: string): boolean {
  return href === '/' ? path === '/' : path === href || path.startsWith(`${href}/`);
}
