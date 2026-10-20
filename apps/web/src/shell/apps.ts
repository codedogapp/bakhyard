/** An app's id: the name of its service, and of its page (plan §4.3). */
export type AppId = 'notes' | 'todos' | 'pulls' | 'time' | 'github' | 'games' | 'planner';

export interface App {
  id: AppId;
  /** Its short name in the nav. */
  label: string;
  name: string;
  description: string;
  /** Its place in the list, as two digits. Apps have no icons: the number and the name identify them (plan §7). */
  number: string;
}

const catalogue: readonly Omit<App, 'number'>[] = [
  { id: 'notes', label: 'Notes', name: 'Notes', description: 'The Obsidian vault, rendered and editable' },
  { id: 'todos', label: 'Todos', name: 'Todos', description: 'Todos linked to Jira tickets' },
  { id: 'pulls', label: 'PRs', name: 'Pull requests', description: 'Bitbucket PRs and reviewer nudges' },
  { id: 'time', label: 'Time', name: 'Time', description: 'Hours logged on the Jira tickets' },
  { id: 'github', label: 'GitHub', name: 'GitHub', description: 'PRs, reviews and issues across owners' },
  { id: 'games', label: 'Games', name: 'Games', description: 'Backloggd and Exophase' },
  { id: 'planner', label: 'Planner', name: 'Planner', description: 'Copilot plans saved to the vault' },
];

/** Every app, in the order the nav and the Home list them. */
export const apps: readonly App[] = catalogue.map((app, index) => ({
  ...app,
  number: String(index + 1).padStart(2, '0'),
}));

/** The path of an app's page. */
export function appHref(id: AppId): string {
  return `/${id}`;
}
