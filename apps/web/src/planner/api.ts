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

/**
 * Starts a plan from the prompt, as typed (decision 405). The failure is the reason to show the user, or none once the
 * plan has started: for a 404, a 413 or a 502, what it means; for any other answer, the service's reason or its status.
 */
export async function startPlan(prompt: string): Promise<string | undefined> {
  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ prompt }),
    });
    if (response.status === 201) {
      return undefined;
    }
    const failure = startFailures.get(response.status);
    return failure ?? ((await response.text()).trim() || `The plan didn't start (${response.status}).`);
  } catch {
    return "The plan didn't start: the planner isn't reachable.";
  }
}
