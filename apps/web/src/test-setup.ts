import '@testing-library/jest-dom/vitest';
import { configure } from '@testing-library/svelte';

// A note's page imports its renderer the first time it shows. On a busy CI runner, or a Mac where other sessions run
// their checks, loading the renderer's packages takes longer than a findBy query's default wait of a second.
configure({ asyncUtilTimeout: 10_000 });

// jsdom has no showModal or close yet. These stand-ins only open and close the dialog: they move no focus and make
// nothing inert, which the end-to-end tests check in real browsers.
HTMLDialogElement.prototype.showModal = function showModal(this: HTMLDialogElement) {
  this.open = true;
};
HTMLDialogElement.prototype.close = function close(this: HTMLDialogElement) {
  this.open = false;
};
