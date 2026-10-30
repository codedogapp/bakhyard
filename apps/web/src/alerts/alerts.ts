import { timeLabel } from '../shell/clock';

/** An alert that fired on the laptop, as notify keeps it (decision 608). */
export interface Alert {
  /** The alert's rule, such as FilesystemFull. */
  name: string;
  /** What fired, in a line; empty when its rule has no summary. */
  summary: string;
  /** What to do about it; empty when its rule has no description. */
  description: string;
  startedAt: string;
  /** When it resolved; none while it fires. */
  resolvedAt?: string;
}

/**
 * The Alerts page's lists: loading, failed to load, or the alerts that fire and those that resolved in the last 30
 * days, each in notify's order, the latest to fire first.
 */
export type Alerts = { kind: 'loading' | 'failed' } | { kind: 'loaded'; firing: Alert[]; resolved: Alert[] };

/** Loads the alerts; never fails, since a failed load is one of the outcomes. */
export async function loadAlerts(): Promise<Alerts> {
  try {
    const response = await fetch('/api/notify/alerts');
    if (response.status !== 200) {
      return { kind: 'failed' };
    }
    const body: { alerts: Alert[] } = await response.json();
    return {
      kind: 'loaded',
      firing: body.alerts.filter((alert) => alert.resolvedAt === undefined),
      resolved: body.alerts.filter((alert) => alert.resolvedAt !== undefined),
    };
  } catch {
    return { kind: 'failed' };
  }
}

/** What an alert's row calls it: its summary, or its name when it has none, as its push does (decision 601). */
export function alertTitle(alert: Alert): string {
  return alert.summary || alert.name;
}

const dayFormat = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short' });

/**
 * When an alert fired and resolved, in the device's time zone: "Since 13:05" while it fires, "13:05–13:20" when it
 * resolved the day it fired, and "13:05 – 1 Nov, 09:00" when it resolved on a later day. The day and the time have a
 * format each, since the browsers join them differently.
 */
export function spanLabel(alert: Alert): string {
  const start = new Date(alert.startedAt);
  if (alert.resolvedAt === undefined) {
    return `Since ${timeLabel(start)}`;
  }
  const end = new Date(alert.resolvedAt);
  if (start.toDateString() === end.toDateString()) {
    return `${timeLabel(start)}–${timeLabel(end)}`;
  }
  return `${timeLabel(start)} – ${dayFormat.format(end)}, ${timeLabel(end)}`;
}
