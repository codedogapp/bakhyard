import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, expect, test, vi } from 'vitest';

import { loadTickets, type Snapshot } from '../jira/tickets';
import { fetchTodos, type Todo } from './api';
import TodosPage from './TodosPage.svelte';

// Spies that call through, so a test can wait for the todos or the tickets to load.
vi.mock('./api', { spy: true });
vi.mock('../jira/tickets', { spy: true });

afterEach(() => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

/**
 * Stubs fetch: the todos' calls go to `todos`, the tickets' call gets `tickets`, by default the answer without Jira, and
 * the dismissed tickets' call gets `dismissals`, by default none.
 */
function stubFetch(
  todos: typeof fetch,
  tickets: Response | Promise<Response> = new Response(null, { status: 404 }),
  dismissals: Response | Promise<Response> = Response.json({ dismissals: [] })
) {
  const answers = new Map([
    ['/api/todos/tickets', tickets],
    ['/api/todos/dismissals', dismissals],
  ]);
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>((...args) => {
      const answer = answers.get(String(args[0]));
      return answer ? Promise.resolve(answer) : todos(...args);
    })
  );
}

test('lists the todos, newest first, once the list loads', async () => {
  const load = Promise.withResolvers<Response>();
  const fetchMock = vi.fn<typeof fetch>().mockReturnValue(load.promise);
  stubFetch(fetchMock);
  render(TodosPage);

  expect(screen.getByRole('heading', { level: 1, name: 'Todos' })).toBeInTheDocument();
  expect(screen.queryByText(/on the list\.$/)).not.toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
  expect(screen.queryByRole('textbox')).not.toBeInTheDocument();

  const todos = [
    { id: 3, title: 'Call <b>home</b>', done: false, priority: 'normal' },
    { id: 1, title: 'Buy milk', done: true, priority: 'normal' },
  ];
  load.resolve(Response.json({ todos }));

  expect(await screen.findByText('2 todos on the list.')).toBeInTheDocument();
  const rows = within(screen.getByRole('list')).getAllByRole('listitem');
  expect(rows.map((row) => row.querySelector('label')?.textContent?.trim())).toEqual(['Call <b>home</b>', 'Buy milk']);
  expect(screen.getByRole('button', { name: 'Edit Call <b>home</b>' })).toHaveTextContent('Edit');
  expect(screen.getByRole('textbox', { name: 'New todo' })).toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledExactlyOnceWith('/api/todos');
});

test('shows a section per priority, High first, each with its count and its todos, newest first', async () => {
  const todos: Todo[] = [
    { id: 3, title: 'Call the bank', done: false, priority: 'low' },
    { id: 2, title: 'Pay the rent', done: false, priority: 'high' },
    { id: 1, title: 'Buy milk', done: true, priority: 'high' },
  ];
  stubFetch(vi.fn<typeof fetch>().mockResolvedValue(Response.json({ todos })));
  render(TodosPage);

  const heads = await screen.findAllByRole('heading', { level: 2, name: /priority$/ });
  const sections = heads.map((head) => within(head.closest('section') as HTMLElement));
  expect(
    sections.map((section) => [
      section.getByRole('heading', { level: 2 }).textContent,
      section.getByText(/^\d+ todos?$/).textContent,
      section.getAllByRole('listitem').map((row) => row.querySelector('label')?.textContent?.trim()),
    ])
  ).toEqual([
    ['High priority', '2 todos', ['Pay the rent', 'Buy milk']],
    ['Low priority', '1 todo', ['Call the bank']],
  ]);
});

test('shows no list without todos, until the first is added', async () => {
  const fetchMock = vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(Response.json({ todos: [] }))
    .mockResolvedValueOnce(
      Response.json({ id: 1, title: 'Buy milk', done: false, priority: 'normal' }, { status: 201 })
    );
  stubFetch(fetchMock);
  render(TodosPage);

  expect(await screen.findByText('No todos on the list.')).toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
  expect(screen.queryByRole('heading', { level: 2, name: /priority$/ })).not.toBeInTheDocument();

  await fireEvent.input(screen.getByRole('textbox', { name: 'New todo' }), { target: { value: 'Buy milk' } });
  await fireEvent.click(screen.getByRole('button', { name: 'Add' }));

  expect(await screen.findByText('1 todo on the list.')).toBeInTheDocument();
  expect(within(screen.getByRole('list')).getByRole('listitem')).toHaveTextContent('Buy milk');
});

test('says when the list could not load', async () => {
  stubFetch(vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')));
  render(TodosPage);

  expect(await screen.findByText("Couldn't load the todos.")).toBeInTheDocument();
  expect(screen.queryByText(/on the list\.$/)).not.toBeInTheDocument();
  expect(screen.queryByRole('list')).not.toBeInTheDocument();
  expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
});

const ticket = {
  key: 'WEB-12',
  summary: 'Show the order history',
  status: 'In Review',
  category: 'in-progress',
  url: 'https://jira.example.com/browse/WEB-12',
} as const;
const snapshot: Snapshot = { tickets: [ticket], searched: '2026-10-26T00:41:00Z' };

test('loads my Jira tickets with the todos, with the inbox before the groups and the tickets after them', async () => {
  const todos: Todo[] = [{ id: 1, title: 'Buy milk', done: false, priority: 'normal' }];
  stubFetch(vi.fn<typeof fetch>().mockResolvedValue(Response.json({ todos })), Response.json(snapshot));
  render(TodosPage);

  const head = await screen.findByRole('heading', { level: 2, name: 'Jira tickets' });
  const heads = screen.getAllByRole('heading', { level: 2 }).map((each) => each.textContent);
  expect(heads).toEqual(['Jira inbox', 'Normal priority', 'Jira tickets']);
  expect(within(head.closest('section') as HTMLElement).getByRole('link')).toHaveAttribute('href', ticket.url);
  expect(fetch).toHaveBeenCalledWith('/api/todos/tickets');
});

test('holds the tickets back until the todos load, so the todos never push them down', async () => {
  const load = Promise.withResolvers<Response>();
  stubFetch(vi.fn<typeof fetch>().mockReturnValue(load.promise), Response.json(snapshot));
  render(TodosPage);

  await waitFor(() => expect(loadTickets).toHaveResolvedWith({ kind: 'loaded', snapshot }));
  expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
  load.resolve(Response.json({ todos: [] }));

  expect(await screen.findByRole('heading', { level: 2, name: 'Jira tickets' })).toBeInTheDocument();
});

test('shows the tickets after the line that says the todos could not load', async () => {
  stubFetch(vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch')), Response.json(snapshot));
  render(TodosPage);

  const head = await screen.findByRole('heading', { level: 2, name: 'Jira tickets' });
  const failed = screen.getByText("Couldn't load the todos.");
  expect(failed.compareDocumentPosition(head)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
});

test('shows no tickets without Jira', async () => {
  stubFetch(vi.fn<typeof fetch>().mockResolvedValue(Response.json({ todos: [] })));
  render(TodosPage);

  await waitFor(() => expect(loadTickets).toHaveResolvedWith({ kind: 'off' }));
  expect(await screen.findByText('No todos on the list.')).toBeInTheDocument();
  expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
});

test('holds the todos back until my open tickets load, so the inbox never pushes them down', async () => {
  const todos: Todo[] = [{ id: 1, title: 'Buy milk', done: false, priority: 'normal', ticket: 'WEB-12' }];
  const load = Promise.withResolvers<Response>();
  stubFetch(vi.fn<typeof fetch>().mockResolvedValue(Response.json({ todos })), load.promise);
  render(TodosPage);

  await waitFor(() => expect(fetchTodos).toHaveResolved());
  expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
  expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
  expect(screen.queryByText(/on the list\.$/)).not.toBeInTheDocument();
  load.resolve(Response.json(snapshot));

  expect(await screen.findByRole('checkbox', { name: 'Buy milk WEB-12 In Review' })).toBeInTheDocument();
  expect(screen.getByText('1 todo on the list.')).toBeInTheDocument();
  expect(screen.getAllByRole('heading', { level: 2 }).map((each) => each.textContent)).toEqual([
    'Normal priority',
    'Jira tickets',
  ]);
});

test('holds the todos back until the dismissed tickets load, and leaves those out of the inbox', async () => {
  const todos: Todo[] = [{ id: 1, title: 'Buy milk', done: false, priority: 'normal' }];
  const load = Promise.withResolvers<Response>();
  stubFetch(vi.fn<typeof fetch>().mockResolvedValue(Response.json({ todos })), Response.json(snapshot), load.promise);
  render(TodosPage);

  await waitFor(() => expect(loadTickets).toHaveResolved());
  await waitFor(() => expect(fetchTodos).toHaveResolved());
  expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
  expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
  expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
  load.resolve(Response.json({ dismissals: ['WEB-12'] }));

  expect(await screen.findByRole('checkbox', { name: 'Buy milk' })).toBeInTheDocument();
  expect(screen.getAllByRole('heading', { level: 2 }).map((each) => each.textContent)).toEqual([
    'Normal priority',
    'Jira tickets',
  ]);
  expect(fetch).toHaveBeenCalledWith('/api/todos/dismissals');
});

test('shows the todos and the tickets, but no inbox, when the dismissed tickets could not load', async () => {
  const todos: Todo[] = [{ id: 1, title: 'Buy milk', done: false, priority: 'normal' }];
  stubFetch(
    vi.fn<typeof fetch>().mockResolvedValue(Response.json({ todos })),
    Response.json(snapshot),
    new Response(null, { status: 500 })
  );
  render(TodosPage);

  expect(await screen.findByRole('checkbox', { name: 'Buy milk' })).toBeInTheDocument();
  expect(screen.getAllByRole('heading', { level: 2 }).map((each) => each.textContent)).toEqual([
    'Normal priority',
    'Jira tickets',
  ]);
});
