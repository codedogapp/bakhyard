import type { Alert } from '../src/alerts/alerts';

/**
 * A made-up laptop's alerts at the tests' time, as notify lists them: two that fire, then three that resolved, one of
 * them the day after it fired, each group the latest to fire first.
 */
export const alerts: Alert[] = [
  {
    name: 'OnBattery',
    summary: 'The laptop runs on battery',
    description: 'No charger has powered the laptop for 2 minutes.',
    startedAt: '2026-10-21T00:12:00.512304Z',
  },
  {
    name: 'FilesystemFull',
    summary: 'A filesystem is nearly full',
    description: '/dev/disk3s1 is 91.2% full.',
    startedAt: '2026-10-20T13:30:00Z',
  },
  {
    name: 'MemoryFull',
    summary: "The laptop's RAM is nearly full",
    description: '93.4% of the RAM is in use.',
    startedAt: '2026-10-20T05:05:00Z',
    resolvedAt: '2026-10-20T05:20:00Z',
  },
  {
    name: 'FluxNotReady',
    summary: 'Flux fails to reconcile',
    description: 'Kustomization flux-system/apps has not been ready for 15 minutes.',
    startedAt: '2026-10-17T14:45:00Z',
    resolvedAt: '2026-10-17T15:20:00Z',
  },
  {
    name: 'CPUHot',
    summary: 'The CPU is too hot',
    description: 'The CPU is at 92 °C.',
    startedAt: '2026-10-02T06:00:00Z',
    resolvedAt: '2026-10-02T06:15:00Z',
  },
];
