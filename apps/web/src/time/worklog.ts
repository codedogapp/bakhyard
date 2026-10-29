const url = '/api/time/worklogs';

/** The service's reason for refusing a worklog. */
export type Refused = { kind: 'refused'; reason: string };

/** How an add ended: added, or refused. */
export type Added = { kind: 'added' } | Refused;

/**
 * Adds a worklog of the minutes on the ticket, from 09:00 on the date in the time service's zone (decision 192); fails
 * on any answer but a 201 or a 400.
 */
export async function addWorklog(ticket: string, date: string, minutes: number): Promise<Added> {
  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ticket, date, minutes }),
  });
  if (response.status === 400) {
    return { kind: 'refused', reason: (await response.text()).trim() };
  }
  if (response.status !== 201) {
    throw new Error(`POST ${url}: ${response.status}`);
  }
  return { kind: 'added' };
}
