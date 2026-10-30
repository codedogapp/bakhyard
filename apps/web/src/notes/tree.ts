/** A note in the vault. */
export interface Note {
  /** Its path in the vault, such as "Work/Plans/Q3.md". */
  path: string;
  /** Its file name without ".md", which is its title (decision 91). */
  title: string;
  /** The path of its folder, such as "Work/Plans"; "" at the vault's top. */
  folder: string;
}

/** A folder of the vault, as the Notes page shows it. */
export interface Folder {
  name: string;
  /** Its path in the vault; "" for the vault itself. */
  path: string;
  folders: Folder[];
  notes: Note[];
  /** How many notes it holds, in its folders too. */
  count: number;
}

const noteExtension = '.md';

// Numbers in numeric order, so "Day 2" comes before "Day 10"; letters in any case, which only breaks ties.
const byName = new Intl.Collator('en', { numeric: true }).compare;

/**
 * The vault's notes, as a tree of the folders that hold them: in each folder, its folders, then its notes, each sorted
 * by name. Other files, such as images, are left out.
 */
export function noteTree(files: readonly string[]): Folder {
  const root = newFolder('', '');
  for (const path of files) {
    if (path.endsWith(noteExtension)) {
      addNote(root, path);
    }
  }
  return finish(root);
}

function newFolder(name: string, path: string): Folder {
  return { name, path, folders: [], notes: [], count: 0 };
}

function addNote(root: Folder, path: string) {
  const names = path.split('/');
  const file = names.pop()!;
  let folder = root;
  for (const name of names) {
    folder = subfolder(folder, name);
  }
  folder.notes.push({ path, title: withoutExtension(file), folder: folder.path });
}

function subfolder(parent: Folder, name: string): Folder {
  let folder = parent.folders.find((candidate) => candidate.name === name);
  if (!folder) {
    folder = newFolder(name, parent.path ? `${parent.path}/${name}` : name);
    parent.folders.push(folder);
  }
  return folder;
}

// Sorts the folder's contents and counts its notes, its folders' first.
function finish(folder: Folder): Folder {
  folder.folders.sort((a, b) => byName(a.name, b.name));
  folder.notes.sort((a, b) => byName(a.title, b.title));
  folder.count = folder.notes.length;
  for (const child of folder.folders) {
    folder.count += finish(child).count;
  }
  return folder;
}

/** The path without ".md": "Work/Plans/Q3" for "Work/Plans/Q3.md". */
export function withoutExtension(path: string): string {
  return path.slice(0, -noteExtension.length);
}

/** The title of the note at path, its file name without ".md": "Q3" for "Work/Plans/Q3.md" (decision 91). */
export function titleOf(path: string): string {
  return withoutExtension(path.slice(path.lastIndexOf('/') + 1));
}

const notePages = '/notes/';

/** The path of the page of the note at path: "/notes/" and the path without ".md", each name escaped (decision 94). */
export function noteHref(path: string): string {
  return `${notePages}${withoutExtension(path).split('/').map(encodeURIComponent).join('/')}`;
}

/**
 * The note whose page is at href, as noteHref writes it; undefined when href isn't under "/notes/", or a name in it is
 * empty or doesn't decode.
 */
export function noteAt(href: string): Note | undefined {
  if (!href.startsWith(notePages)) {
    return undefined;
  }
  let names: string[];
  try {
    names = href.slice(notePages.length).split('/').map(decodeURIComponent);
  } catch {
    // A "%" that doesn't start an escape of UTF-8.
    return undefined;
  }
  if (names.includes('')) {
    return undefined;
  }
  return { path: `${names.join('/')}${noteExtension}`, title: names.at(-1)!, folder: names.slice(0, -1).join('/') };
}

/** The notes whose path without ".md" holds every word of the filter, in any case, in the tree's order. */
export function matches(tree: Folder, filter: string): Note[] {
  const words = filter.toLowerCase().split(/\s+/).filter(Boolean);
  return notesIn(tree).filter((note) => {
    const name = withoutExtension(note.path).toLowerCase();
    return words.every((word) => name.includes(word));
  });
}

// Every note in the folder, as its tree lists them: its folders' notes first, then its own.
function notesIn(folder: Folder): Note[] {
  return [...folder.folders.flatMap(notesIn), ...folder.notes];
}
