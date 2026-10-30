const url = '/api/planner/plans';

/** A plan, as the planner service lists it (decision 413). */
export interface Plan {
  id: string;
  /** The CLI's name for the plan, which may be its prompt, or "" before it has one. */
  title: string;
  started_at: string;
  /** When the CLI's record of the plan last changed. */
  updated_at: string;
}

/** The Planner page's list: loading, off when Copilot isn't set up, failed to load, or the plans. */
export type Plans = { kind: 'loading' | 'off' | 'failed' } | { kind: 'loaded'; plans: Plan[] };

/**
 * Loads the plans, the last updated first. A 404 says that Copilot isn't set up: the planner's own, or the gateway's
 * while it doesn't forward to the planner. Never fails, since a failed load is one of the outcomes.
 */
export async function loadPlans(): Promise<Plans> {
  try {
    const response = await fetch(url);
    if (response.status === 404) {
      return { kind: 'off' };
    }
    if (response.status !== 200) {
      return { kind: 'failed' };
    }
    const body: { plans: Plan[] } = await response.json();
    return { kind: 'loaded', plans: body.plans };
  } catch {
    return { kind: 'failed' };
  }
}

// What a failed start means, by its status, where the service's own text, such as "Bad Gateway", would say less.
const startFailures = new Map([
  [404, "Copilot isn't set up."],
  [413, 'The prompt is too long.'],
  [502, "Copilot couldn't start the plan."],
]);

// How a start went: started, with the new plan's ID, or failed, with the reason to show the user.
type Start = { kind: 'started'; id: string } | { kind: 'failed'; reason: string };

/**
 * Starts a plan from the prompt, as typed (decision 405). A failed start's reason is, for a 404, a 413 or a 502, what
 * it means; for any other answer, the service's reason or its status.
 */
export async function startPlan(prompt: string): Promise<Start> {
  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ prompt }),
    });
    if (response.status === 201) {
      const body: { id: string } = await response.json();
      return { kind: 'started', id: body.id };
    }
    const reason = startFailures.get(response.status);
    return {
      kind: 'failed',
      reason: reason ?? ((await response.text()).trim() || `The plan didn't start (${response.status}).`),
    };
  } catch {
    return { kind: 'failed', reason: "The plan didn't start: the planner isn't reachable." };
  }
}

// What a failed save means, by its status, where the service's own text would say less. A 404 has three meanings,
// which the service's text tells apart: no such plan, no Copilot, or no saving to the vault.
const saveFailures = new Map([
  [409, "Copilot hasn't written the plan yet."],
  [502, "Couldn't save the plan in the vault."],
]);

/** How a save went: saved, with the note's path in the vault, or failed, with the reason to show the user. */
export type Save = { kind: 'saved'; path: string } | { kind: 'failed'; reason: string };

/**
 * Saves the plan of the plan with the ID, as its page's path holds it, in the vault as a new note (decision 709). A
 * failed save's reason is, for a 409 or a 502, what it means; for any other answer, the service's reason or its status.
 */
export async function savePlan(id: string): Promise<Save> {
  try {
    const response = await fetch(`${url}/${id}/note`, { method: 'POST' });
    if (response.status === 201) {
      const body: { path: string } = await response.json();
      return { kind: 'saved', path: body.path };
    }
    const reason = saveFailures.get(response.status);
    return {
      kind: 'failed',
      reason: reason ?? ((await response.text()).trim() || `The plan wasn't saved (${response.status}).`),
    };
  } catch {
    return { kind: 'failed', reason: "The plan wasn't saved: the planner isn't reachable." };
  }
}

// What a failed post of a text means: the reasons for the statuses whose own text would say less, and what the text
// is, for the others.
interface Failures {
  what: 'message' | 'answer';
  reasons: Map<number, string>;
}

const messageFailures: Failures = {
  what: 'message',
  reasons: new Map([
    [404, 'Copilot no longer has this plan.'],
    [413, 'The message is too long.'],
    [502, "Copilot couldn't take the message."],
  ]),
};

const answerFailures: Failures = {
  what: 'answer',
  reasons: new Map([
    [404, 'Copilot no longer waits for this question.'],
    [413, 'The answer is too long.'],
    [502, "Copilot couldn't take the answer."],
  ]),
};

// Posts the text, as typed, as the body {"text"} that the planner's message and answer routes take. The failure is the
// reason to show the user, or none once the planner has the text: by its status, as failures has it, or else the
// service's reason or the status.
async function postText(path: string, text: string, failures: Failures): Promise<string | undefined> {
  try {
    const response = await fetch(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text }),
    });
    if (response.status === 204) {
      return undefined;
    }
    const reason = failures.reasons.get(response.status);
    return reason ?? ((await response.text()).trim() || `The ${failures.what} wasn't sent (${response.status}).`);
  } catch {
    return `The ${failures.what} wasn't sent: the planner isn't reachable.`;
  }
}

/**
 * Sends the text, as typed, to the plan with the ID, as its page's path holds it, as the user's next message (decision
 * 414). The failure is the reason to show the user, or none once the planner has the message.
 */
export function sendMessage(id: string, text: string): Promise<string | undefined> {
  return postText(`${url}/${id}/messages`, text, messageFailures);
}

/**
 * Sends the text, as typed, as the answer to the question of the plan with the ID, as its page's path holds it
 * (decision 411). The failure is the reason to show the user, or none once the planner has the answer.
 */
export function sendAnswer(id: string, question: string, text: string): Promise<string | undefined> {
  return postText(`${url}/${id}/questions/${encodeURIComponent(question)}/answer`, text, answerFailures);
}
