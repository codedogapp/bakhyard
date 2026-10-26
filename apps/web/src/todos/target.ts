import type { Place, Priority } from './api';

/** A row on the page: its todo, and its top and bottom edges, in px from the page's top. */
interface RowBox {
  id: number;
  top: number;
  bottom: number;
}

/** A group's priority and its rows on the page, in order. */
export interface GroupBox {
  priority: Priority;
  rows: readonly RowBox[];
}

/** Where a dropped todo goes: the priority of the group it's dropped in, and its place in the list. */
export interface Target {
  priority: Priority;
  place: Place;
}

/** How far y is from the group's rows: 0 within them. */
function distance({ rows }: GroupBox, y: number): number {
  const [top, bottom] = [rows.at(0)?.top ?? Number.POSITIVE_INFINITY, rows.at(-1)?.bottom ?? 0];
  return Math.max(top - y, y - bottom, 0);
}

/**
 * Where todo id lands when dropped at y, the rows being as they were when its drag began: in the group whose rows are
 * nearest y, right before the first row whose middle is below y, else right after the group's last row. Undefined
 * where the todo already is.
 */
export function dropTarget(groups: readonly GroupBox[], id: number, y: number): Target | undefined {
  const [group] = groups.toSorted((one, other) => distance(one, y) - distance(other, y));
  const rows = group?.rows ?? [];
  const at = rows.filter(({ top, bottom }) => (top + bottom) / 2 < y).length;
  const own = rows.findIndex((row) => row.id === id);
  const [next, last] = [rows[at], rows.at(-1)];
  if (group === undefined || last === undefined || (own >= 0 && (at === own || at === own + 1))) {
    return undefined;
  }
  return { priority: group.priority, place: next ? { before: next.id } : { after: last.id } };
}
