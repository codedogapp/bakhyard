/** A todo, as the todos service gives it. */
export interface Todo {
  id: number;
  title: string;
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
