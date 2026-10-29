import type { Push } from './push';

const states: Record<Push['kind'], string> = {
  on: 'On',
  off: 'Off',
  blocked: 'Blocked',
  unsupported: 'Unavailable',
  failed: 'Failed',
};

/** The bell's name: what it is, then the state of this device's notifications once the page knows it. */
export function bellName(push?: Push): string {
  return push ? `Notifications: ${states[push.kind]}` : 'Notifications';
}

/** What the panel says of this device's notifications. Off says nothing: the panel's button turns them on. */
export function pushStatus(push?: Push): string {
  switch (push?.kind) {
    case undefined:
      return 'Checking notifications…';
    case 'on':
      return 'Notifications on';
    case 'off':
      return '';
    case 'blocked':
      return 'Notifications are blocked in the browser settings';
    case 'unsupported':
      return 'Notifications need the Home Screen app';
    case 'failed':
      return `Notifications failed: ${push.reason}`;
  }
}
