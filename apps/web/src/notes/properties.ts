import { parse } from 'yaml';

import type { Link } from './links';

/** A row of a note's Properties block: a key of its frontmatter, with its value, or its list's values, as text. */
export interface Property {
  key: string;
  /** One value, shown as text, or a list's values, shown as chips; or a wikilink's, which the renderer finds. */
  value: string | string[] | Link;
}

/**
 * The Properties block of a note whose frontmatter is source, in its order, without its title. A string, number or
 * boolean is a value; a list's strings, numbers and booleans are values too. A key whose value is null, empty or a map
 * is left out. Source that doesn't parse, or isn't a map, gives no properties.
 */
export function noteProperties(source: string): Property[] {
  let data: unknown;
  try {
    // "error" throws on errors and logs nothing; a Map keeps the keys in the file's order.
    data = parse(source, { logLevel: 'error', mapAsMap: true });
  } catch {
    return [];
  }
  if (!(data instanceof Map)) {
    return [];
  }
  const properties: Property[] = [];
  for (const [key, value] of data) {
    const name = text(key);
    const shown = Array.isArray(value) ? values(value) : text(value);
    if (name !== undefined && name !== 'title' && shown !== undefined) {
      properties.push({ key: name, value: shown });
    }
  }
  return properties;
}

// The list's values; undefined when it has none.
function values(list: unknown[]): string[] | undefined {
  const shown = list.map(text).filter((item) => item !== undefined);
  return shown.length > 0 ? shown : undefined;
}

// A string, number or boolean as text; undefined for anything else, or an empty string.
function text(value: unknown): string | undefined {
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  return typeof value === 'string' && value !== '' ? value : undefined;
}
