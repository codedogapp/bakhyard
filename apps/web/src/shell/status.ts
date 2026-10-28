import type { Connection } from '../home/poll';
import { timeLabel } from './clock';

/**
 * The top bar's status: blank until the first load ends, then "Online", "Signed out", or "Offline · as of 01:10"
 * with the time of the last good load. Online, it also says how many sources the Home couldn't reach, if any.
 */
export function statusLabel(connection: Connection, asOf?: Date, unavailable = 0): string {
  if (connection === 'connecting') {
    return '';
  }
  if (connection === 'online') {
    return onlineLabel(unavailable);
  }
  if (connection === 'signed-out') {
    return 'Signed out';
  }
  return asOf ? `Offline · as of ${timeLabel(asOf)}` : 'Offline';
}

function onlineLabel(unavailable: number): string {
  if (unavailable === 0) {
    return 'Online';
  }
  return `Online · ${unavailable} ${unavailable === 1 ? 'source' : 'sources'} unavailable`;
}
