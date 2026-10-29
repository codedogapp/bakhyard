// The service worker for the app's notifications. It shows every push as a notification, because Safari revokes a
// subscription whose pushes show nothing, and tells the open pages, so that one on screen shows the question at once.

const fallback = { title: 'Bakhyard', body: 'A new question', url: '/nudge' };

// A new worker takes over the open pages at once.
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (event) => event.waitUntil(self.clients.claim()));

self.addEventListener('push', (event) => {
  const { title, body, url } = message(event.data);
  event.waitUntil(Promise.all([self.registration.showNotification(title, { body, data: { url } }), tell()]));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  event.waitUntil(open(event.notification.data?.url ?? fallback.url));
});

// A push's payload is JSON with a title and a body, and perhaps the path of a page of this site.
function message(data) {
  try {
    const { title, body, url } = data.json();
    if (typeof title === 'string' && typeof body === 'string') {
      return { title, body, url: typeof url === 'string' && url.startsWith('/') ? url : fallback.url };
    }
  } catch {
    // A push without a payload, or with one that isn't JSON.
  }
  return fallback;
}

async function tell() {
  for (const page of await self.clients.matchAll({ type: 'window' })) {
    page.postMessage({ kind: 'push' });
  }
}

// Focuses a window of the site and takes it to the page, or opens one.
async function open(url) {
  const [page] = await self.clients.matchAll({ type: 'window' });
  if (!page) {
    await self.clients.openWindow(url);
    return;
  }
  await page.focus();
  await page.navigate(url);
}
