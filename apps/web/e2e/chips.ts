import type { Locator } from '@playwright/test';

/** How each chip looks: filled with the ink, or outlined in it with its text in the ink; else its background. */
export function chipLooks(chips: Locator): Promise<string[]> {
  return chips.evaluateAll((all) =>
    all.map((chip) => {
      const { backgroundColor, borderTopColor, color } = getComputedStyle(chip);
      if (backgroundColor === borderTopColor) {
        return 'filled';
      }
      return backgroundColor === 'rgba(0, 0, 0, 0)' && color === borderTopColor ? 'outlined' : backgroundColor;
    })
  );
}
