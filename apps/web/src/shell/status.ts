import type { Connection } from '../home/poll';
import { timeLabel } from './clock';

/** The top bar's status: blank until the first load ends, then "Online", or "Offline · as of 01:10" with the time of the last good load. */
export function statusLabel(connection: Connection, asOf?: Date): string {
  if (connection === 'connecting') {
    return '';
  }
  if (connection === 'online') {
    return 'Online';
  }
  return asOf ? `Offline · as of ${timeLabel(asOf)}` : 'Offline';
}
