/**
 * Whether the focus was lost: it's on the page's body, or it stayed in a closed dialog, as WebKit leaves it after a
 * click, which focuses no button (plan §6.8).
 */
export function focusLost(): boolean {
  const focused = document.activeElement;
  return focused === document.body || Boolean(focused?.closest('dialog:not([open])'));
}
