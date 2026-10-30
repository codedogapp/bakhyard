/** The vault's list, from the body of GET /api/notes or POST /api/notes/sync. */
export interface Listing {
  /** The vault's files, by their paths in it, such as "Daily/To do.md". */
  files: string[];
  /** Whether the notes service saves notes, which it says once a store is set up (decision 400). */
  editable: boolean;
}

/** Lists the vault's files as of its last scan; fails when the gateway can't be reached or answers with an error. */
export function fetchNotes(): Promise<Listing> {
  return listing('/api/notes', 'GET');
}

/** Scans the vault again and lists its files as they are now; fails as fetchNotes does. */
export function syncNotes(): Promise<Listing> {
  return listing('/api/notes/sync', 'POST');
}

async function listing(url: string, method: string): Promise<Listing> {
  const response = await fetch(url, { method });
  if (!response.ok) {
    throw new Error(`${method} ${url}: ${response.status}`);
  }
  // Only the fields the app reads; editable is left out until a store is set up.
  const body: { files: string[]; editable?: boolean } = await response.json();
  return { files: body.files, editable: body.editable === true };
}

/** The URL of the file at path in the vault, such as "Work/Plans/Q3.md", each name in the path escaped. */
export function fileUrl(path: string): string {
  return `/api/notes/files/${path.split('/').map(encodeURIComponent).join('/')}`;
}

/** A note's Markdown, and its version, which a save sends back (decision 400). */
export interface NoteFile {
  markdown: string;
  /** The note's ETag, the SHA-256 of its bytes in quotes; blank when the answer had none. */
  version: string;
}

/**
 * Loads the note at path in the vault, such as "Work/Plans/Q3.md": its Markdown and version, or undefined when the
 * vault has no such note. Fails when the gateway can't be reached or answers with another error.
 */
export async function fetchNote(path: string): Promise<NoteFile | undefined> {
  const url = fileUrl(path);
  const response = await fetch(url);
  if (response.status === 404) {
    return undefined;
  }
  if (!response.ok) {
    throw new Error(`GET ${url}: ${response.status}`);
  }
  return { markdown: await response.text(), version: response.headers.get('ETag') ?? '' };
}

/**
 * Saves markdown as the note at path, unless the note has changed since version: the saved note's version, or
 * undefined when the note has changed. Fails when the gateway can't be reached or answers with another error.
 */
export async function saveNote(path: string, markdown: string, version: string): Promise<string | undefined> {
  const url = fileUrl(path);
  const response = await fetch(url, {
    method: 'PUT',
    headers: { 'Content-Type': 'text/markdown; charset=utf-8', 'If-Match': version },
    body: markdown,
  });
  if (response.status === 412) {
    return undefined;
  }
  if (!response.ok) {
    throw new Error(`PUT ${url}: ${response.status}`);
  }
  return response.headers.get('ETag') ?? '';
}
