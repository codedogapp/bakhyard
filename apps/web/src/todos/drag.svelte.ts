import type { Attachment } from 'svelte/attachments';
import { on } from 'svelte/events';

import type { Editor } from './editor.svelte';
import type { TodosState } from './state.svelte';
import { dropTarget, type GroupBox, type Target } from './target';

/** A drag under way: its pointer, its todo, the rows as it began, and where the todo would land. */
interface Dragging {
  pointer: number;
  id: number;
  groups: readonly GroupBox[];
  target?: Target;
}

/** How a row looks during a drag: lifted in place, or marked as where the dragged todo would land. */
export type Look = 'lifted' | 'drop-before' | 'drop-after';

/** Drags a todo by its row's grip, with a mouse, a pen or a finger, to another place or group (decision 120). */
export class Drag {
  #dragging = $state.raw<Dragging>();
  /** The rows, by their todo's ID; plain, since nothing on the page shows it. */
  readonly #rows = new Map<number, HTMLElement>();
  readonly #list: TodosState;
  readonly #editor: Pick<Editor, 'busy'>;

  constructor(list: TodosState, editor: Pick<Editor, 'busy'>) {
    this.#list = list;
    this.#editor = editor;
  }

  /** Whether a drop or the editor waits for the service, which no drag may start before. */
  get waiting(): boolean {
    return this.#list.dropping || this.#editor.busy;
  }

  /** How the todo's row looks during a drag, if it looks different. */
  look(id: number): Look | undefined {
    const place = this.#dragging?.target?.place;
    if (this.#dragging?.id === id) {
      return 'lifted';
    }
    if (place && 'before' in place) {
      return place.before === id ? 'drop-before' : undefined;
    }
    return place?.after === id ? 'drop-after' : undefined;
  }

  /** An attachment for the todo's row, which a drag measures as it begins. */
  row(id: number): Attachment<HTMLElement> {
    return (row) => {
      this.#rows.set(id, row);
      return () => {
        // A todo dropped in another group gets a new row there: keep the newer, whichever order the two come and go in.
        if (this.#rows.get(id) === row) {
          this.#rows.delete(id);
        }
      };
    };
  }

  /** An attachment for the todo's grip, which drags the todo. */
  grip(id: number): Attachment<HTMLElement> {
    return (grip) => {
      const stops = [
        on(grip, 'pointerdown', (event) => this.#start(id, grip, event)),
        on(grip, 'pointermove', (event) => this.#move(event)),
        on(grip, 'pointerup', (event) => this.#drop(event)),
        on(grip, 'lostpointercapture', () => this.#cancel(id)),
      ];
      return () => {
        this.#cancel(id);
        for (const stop of stops) {
          stop();
        }
      };
    };
  }

  #start(id: number, grip: HTMLElement, event: PointerEvent): void {
    if (!event.isPrimary || event.button !== 0 || this.#dragging || this.waiting) {
      return;
    }
    // Else a mouse would select the text of the rows it passes.
    event.preventDefault();
    grip.setPointerCapture(event.pointerId);
    this.#dragging = { pointer: event.pointerId, id, groups: this.#measure() };
  }

  #move(event: PointerEvent): void {
    const dragging = this.#dragging;
    if (dragging?.pointer === event.pointerId) {
      this.#dragging = { ...dragging, target: dropTarget(dragging.groups, dragging.id, event.pageY) };
    }
  }

  #drop(event: PointerEvent): void {
    const dragging = this.#dragging;
    if (dragging?.pointer !== event.pointerId) {
      return;
    }
    this.#dragging = undefined;
    if (dragging.target) {
      void this.#list.drop(dragging.id, dragging.target);
    }
  }

  /** Ends todo id's drag, if it's under way, with nothing sent: the browser took the pointer, or the grip went. */
  #cancel(id: number): void {
    if (this.#dragging?.id === id) {
      this.#dragging = undefined;
    }
  }

  /** The rows' boxes on the page, group by group, in the page's order. */
  #measure(): GroupBox[] {
    return this.#list.groups.map(({ priority, todos }) => ({
      priority,
      rows: todos.flatMap(({ id }) => {
        const box = this.#rows.get(id)?.getBoundingClientRect();
        return box ? [{ id, top: box.top + scrollY, bottom: box.bottom + scrollY }] : [];
      }),
    }));
  }
}
