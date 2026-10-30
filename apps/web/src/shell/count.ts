/**
 * A list's count, as the note beside its section's head: "1 PR" or "3 PRs" for the noun PR; none for an empty list,
 * which says so itself.
 */
export function countLabel(count: number, noun: string): string | undefined {
  if (count === 0) {
    return undefined;
  }
  return count === 1 ? `1 ${noun}` : `${count} ${noun}s`;
}
