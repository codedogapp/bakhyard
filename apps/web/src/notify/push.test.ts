import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import { preparePush } from './push';

// "AQID" is the bytes 1, 2, 3 in base64url.
const key = 'AQID';
const keyBytes = new Uint8Array([1, 2, 3]).buffer;

let subscription: { options: unknown; toJSON: () => unknown; unsubscribe: ReturnType<typeof vi.fn> };
let pushManager: { getSubscription: ReturnType<typeof vi.fn>; subscribe: ReturnType<typeof vi.fn> };
let fetchMock = vi.fn<typeof fetch>();

beforeEach(() => {
  subscription = {
    options: { applicationServerKey: keyBytes },
    toJSON: () => ({ endpoint: 'https://push.example/1' }),
    unsubscribe: vi.fn(async () => true),
  };
  pushManager = { getSubscription: vi.fn(async () => null), subscribe: vi.fn(async () => subscription) };
  const register = vi.fn(async () => undefined);
  vi.stubGlobal('navigator', { serviceWorker: { register, ready: Promise.resolve({ pushManager }) } });
  vi.stubGlobal('PushManager', class {});
  vi.stubGlobal('Notification', { permission: 'default' });
  fetchMock = vi.fn<typeof fetch>(async (url) =>
    String(url).endsWith('/key') ? Response.json({ key }) : new Response(null, { status: 204 })
  );
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test('is unsupported without a push manager, as in a Safari tab on the iPhone', async () => {
  vi.stubGlobal('PushManager', undefined);
  // `'PushManager' in window` is about the property, so it has to be gone.
  Reflect.deleteProperty(window, 'PushManager');

  expect(await preparePush()).toEqual({ kind: 'unsupported' });
});

test('is blocked when the browser denied the permission', async () => {
  vi.stubGlobal('Notification', { permission: 'denied' });

  expect(await preparePush()).toEqual({ kind: 'blocked' });
});

test('is on, and gives the service the subscription again, when it was made with the current key', async () => {
  pushManager.getSubscription.mockResolvedValue(subscription);

  expect(await preparePush()).toEqual({ kind: 'on' });
  const [, init] = fetchMock.mock.calls[1];
  expect(fetchMock.mock.calls[1][0]).toBe('/api/notify/subscriptions');
  expect(init?.body).toBe('{"endpoint":"https://push.example/1"}');
});

test('drops a subscription made with another key, and offers to subscribe', async () => {
  subscription.options = { applicationServerKey: new Uint8Array([9]).buffer };
  pushManager.getSubscription.mockResolvedValue(subscription);

  const push = await preparePush();

  expect(subscription.unsubscribe).toHaveBeenCalledOnce();
  expect(push.kind).toBe('off');
});

test('drops a subscription that has no key', async () => {
  subscription.options = { applicationServerKey: null };
  pushManager.getSubscription.mockResolvedValue(subscription);

  expect((await preparePush()).kind).toBe('off');
  expect(subscription.unsubscribe).toHaveBeenCalledOnce();
});

test('subscribes with the service key, and gives the service the subscription', async () => {
  const push = await preparePush();
  if (push.kind !== 'off') {
    throw new Error(`want off, got ${push.kind}`);
  }

  expect(await push.enable()).toBeUndefined();
  expect(pushManager.subscribe).toHaveBeenCalledWith({ userVisibleOnly: true, applicationServerKey: key });
  expect(fetchMock).toHaveBeenLastCalledWith('/api/notify/subscriptions', expect.objectContaining({ method: 'POST' }));
});

test.each([
  {
    name: 'the browser refuses',
    fail: () => pushManager.subscribe.mockRejectedValue(new Error('denied')),
    want: 'denied',
  },
  { name: 'it throws a non-error', fail: () => pushManager.subscribe.mockRejectedValue('no'), want: 'no' },
  {
    name: 'the service refuses',
    fail: () => fetchMock.mockResolvedValueOnce(new Response('bad subscription\n', { status: 400 })),
    want: 'bad subscription',
  },
  {
    name: 'the service refuses without a reason',
    fail: () => fetchMock.mockResolvedValueOnce(new Response('', { status: 500 })),
    want: 'status 500',
  },
])('says why subscribing failed when $name', async ({ fail, want }) => {
  const push = await preparePush();
  if (push.kind !== 'off') {
    throw new Error(`want off, got ${push.kind}`);
  }
  fail();

  expect(await push.enable()).toBe(want);
});

test('fails, with the reason, when the key cannot be loaded', async () => {
  fetchMock.mockResolvedValueOnce(new Response('', { status: 503 }));

  expect(await preparePush()).toEqual({ kind: 'failed', reason: "the key's request failed (503)" });
});
