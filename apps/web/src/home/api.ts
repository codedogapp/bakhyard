/** An item that needs the user, with only the fields the app reads so far. */
export interface Item {
  late: boolean;
}

/** The Home screen's data from GET /api/home, with only the fields the app reads so far. */
export interface Home {
  name: string;
  items: Item[];
}

/** Fetches the Home; fails when the gateway can't be reached or answers with an error. */
export async function fetchHome(): Promise<Home> {
  const response = await fetch('/api/home');
  if (!response.ok) {
    throw new Error(`GET /api/home: ${response.status}`);
  }
  return response.json();
}
