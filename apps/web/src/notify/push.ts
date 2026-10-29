/** What the page knows of this device's notifications. */
export type Push =
  | { kind: 'unsupported' | 'blocked' | 'on' }
  | { kind: 'failed'; reason: string }
  /** Notifications are off; enable asks for the permission and subscribes. It returns the reason when it fails. */
  | { kind: 'off'; enable: () => Promise<string | undefined> };

const workerPath = '/sw.js';

/** Registers the service worker, which tells the open pages about each push, and returns its registration. */
async function register(): Promise<ServiceWorkerRegistration> {
  await navigator.serviceWorker.register(workerPath);
  return navigator.serviceWorker.ready;
}

/**
 * Looks at this device's notifications. A subscription made with the service's current key goes to the service again,
 * which keeps each device once, in case it lost this one; a subscription made with an earlier key is dropped.
 */
export async function preparePush(): Promise<Push> {
  // A Safari tab on the iPhone has no push manager: only the Home Screen app gets pushes.
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    return { kind: 'unsupported' };
  }
  if (Notification.permission === 'denied') {
    return { kind: 'blocked' };
  }
  try {
    const key = await loadKey();
    const { pushManager } = await register();
    const subscription = await pushManager.getSubscription();
    if (subscription && sameKey(subscription, key)) {
      await give(subscription);
      return { kind: 'on' };
    }
    await subscription?.unsubscribe();
    // The key is in hand before the tap: Safari allows the permission prompt only within it.
    return { kind: 'off', enable: () => subscribe(pushManager, key) };
  } catch (error) {
    return { kind: 'failed', reason: reasonOf(error) };
  }
}

async function subscribe(pushManager: PushManager, key: string): Promise<string | undefined> {
  try {
    await give(await pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: key }));
    return undefined;
  } catch (error) {
    return reasonOf(error);
  }
}

async function loadKey(): Promise<string> {
  const response = await fetch('/api/notify/key');
  if (response.status !== 200) {
    throw new Error(`the key's request failed (${response.status})`);
  }
  const { key } = await response.json();
  return key;
}

// Gives the subscription to the service. A refusal's reason is the service's text.
async function give(subscription: PushSubscription): Promise<void> {
  const response = await fetch('/api/notify/subscriptions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(subscription),
  });
  if (response.status !== 204) {
    const reason = (await response.text()).trim();
    throw new Error(reason || `status ${response.status}`);
  }
}

// Whether the subscription was made with key, a base64url string.
function sameKey(subscription: PushSubscription, key: string): boolean {
  const used = subscription.options.applicationServerKey;
  if (!used) {
    return false;
  }
  const binary = String.fromCodePoint(...new Uint8Array(used));
  const encoded = btoa(binary).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '');
  return encoded === key;
}

function reasonOf(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
