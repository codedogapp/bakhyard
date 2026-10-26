/** A todo's priority; a new todo is Normal. */
export type Priority = 'high' | 'normal' | 'low';

/** A todo, as the todos service gives it. */
export interface Todo {
  id: number;
  title: string;
  done: boolean;
  priority: Priority;
}

/** The body of GET /api/todos. */
interface Listing {
  todos: Todo[];
}

const url = '/api/todos';

/**
 * Lists the todos in the service's order, newest first until the user moves them; fails when the gateway can't be
 * reached or answers with an error.
 */
export async function fetchTodos(): Promise<Todo[]> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`GET ${url}: ${response.status}`);
  }
  const body: Listing = await response.json();
  return body.todos;
}

/** The service's reason for refusing a title. */
export type Refused = { kind: 'refused'; reason: string };

/** How an add ended: the new todo, or the service's reason for refusing its title. */
export type Added = { kind: 'added'; todo: Todo } | Refused;

/** What the row's editor changes in a todo: its title, its priority or both. */
export type Edit = Partial<Pick<Todo, 'title' | 'priority'>>;

/** How an edit ended: the edited todo, or the service's reason for refusing the new title. */
export type Edited = { kind: 'edited'; todo: Todo } | Refused;

async function refusal(response: Response): Promise<Refused> {
  return { kind: 'refused', reason: (await response.text()).trim() };
}

/** Adds a todo with the title as typed, which the service trims and checks; fails on any answer but a 201 or a 400. */
export async function addTodo(title: string): Promise<Added> {
  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title }),
  });
  if (response.status === 400) {
    return refusal(response);
  }
  if (response.status !== 201) {
    throw new Error(`POST ${url}: ${response.status}`);
  }
  const todo: Todo = await response.json();
  return { kind: 'added', todo };
}

/** Where a move puts a todo: right before another, that is above it in the list, or right after it, below. */
export type Place = { before: number } | { after: number };

/** Sends the change to the todo; the service changes only what the change names. */
function patchTodo(id: number, change: Partial<Omit<Todo, 'id'>> | Place): Promise<Response> {
  return fetch(`${url}/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(change),
  });
}

/** Ticks the todo done, or unticks it, and gives the todo as the service stored it; fails on any answer but a 200. */
export async function tickTodo(id: number, done: boolean): Promise<Todo> {
  const response = await patchTodo(id, { done });
  if (response.status !== 200) {
    throw new Error(`PATCH ${url}/${id}: ${response.status}`);
  }
  return response.json();
}

/**
 * Changes the todo's title, as typed, which the service trims and checks, its priority, or both; the service keeps
 * what the edit leaves out. Fails on any answer but a 200 or a 400.
 */
export async function editTodo(id: number, edit: Edit): Promise<Edited> {
  const response = await patchTodo(id, edit);
  if (response.status === 400) {
    return refusal(response);
  }
  if (response.status !== 200) {
    throw new Error(`PATCH ${url}/${id}: ${response.status}`);
  }
  const todo: Todo = await response.json();
  return { kind: 'edited', todo };
}

/**
 * Moves the todo right before or right after another, and into the priority's group if given, in one request; fails
 * on any answer but a 200.
 */
export async function moveTodo(id: number, place: Place, priority?: Priority): Promise<void> {
  const response = await patchTodo(id, priority === undefined ? place : { ...place, priority });
  if (response.status !== 200) {
    throw new Error(`PATCH ${url}/${id}: ${response.status}`);
  }
}

/**
 * Deletes the todo for good. A 404 counts as deleted, since the todo is gone either way, say deleted from another
 * device; fails on any other answer but a 204.
 */
export async function deleteTodo(id: number): Promise<void> {
  const response = await fetch(`${url}/${id}`, { method: 'DELETE' });
  if (response.status !== 204 && response.status !== 404) {
    throw new Error(`DELETE ${url}/${id}: ${response.status}`);
  }
}
