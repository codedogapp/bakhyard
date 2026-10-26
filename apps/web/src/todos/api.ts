/** A todo, as the todos service gives it. */
export interface Todo {
  id: number;
  title: string;
  done: boolean;
}

/** The body of GET /api/todos. */
interface Listing {
  todos: Todo[];
}

const url = '/api/todos';

/** Lists the todos, newest first; fails when the gateway can't be reached or answers with an error. */
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

/** How a rename ended: the renamed todo, or the service's reason for refusing the new title. */
export type Renamed = { kind: 'renamed'; todo: Todo } | Refused;

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

/** Sends the change to the todo; the service changes only what the change names. */
function patchTodo(id: number, change: Partial<Omit<Todo, 'id'>>): Promise<Response> {
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

/** Renames the todo with the title as typed, which the service trims and checks; fails on any answer but a 200 or a 400. */
export async function renameTodo(id: number, title: string): Promise<Renamed> {
  const response = await patchTodo(id, { title });
  if (response.status === 400) {
    return refusal(response);
  }
  if (response.status !== 200) {
    throw new Error(`PATCH ${url}/${id}: ${response.status}`);
  }
  const todo: Todo = await response.json();
  return { kind: 'renamed', todo };
}
