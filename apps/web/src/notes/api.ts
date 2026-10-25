/** The body of GET /api/notes and POST /api/notes/sync, with only the field the app reads. */
interface Listing {
  /** The vault's files, by their paths in it, such as "Daily/To do.md". */
  files: string[];
}

/** Lists the vault's files as of its last scan; fails when the gateway can't be reached or answers with an error. */
export function fetchNotes(): Promise<string[]> {
  return listing('/api/notes', 'GET');
}

/** Scans the vault again and lists its files as they are now; fails as fetchNotes does. */
export function syncNotes(): Promise<string[]> {
  return listing('/api/notes/sync', 'POST');
}

async function listing(url: string, method: string): Promise<string[]> {
  const response = await fetch(url, { method });
  if (!response.ok) {
    throw new Error(`${method} ${url}: ${response.status}`);
  }
  const body: Listing = await response.json();
  return body.files;
}

/** The URL of the file at path in the vault, such as "Work/Plans/Q3.md", each name in the path escaped. */
export function fileUrl(path: string): string {
  return `/api/notes/files/${path.split('/').map(encodeURIComponent).join('/')}`;
}

/**
 * Loads the note at path in the vault, such as "Work/Plans/Q3.md": its Markdown, or undefined when the vault has no
 * such note. Fails when the gateway can't be reached or answers with another error.
 */
export async function fetchNote(path: string): Promise<string | undefined> {
  const url = fileUrl(path);
  const response = await fetch(url);
  if (response.status === 404) {
    return undefined;
  }
  if (!response.ok) {
    throw new Error(`GET ${url}: ${response.status}`);
  }
  return response.text();
}
