import { defineConfig, devices } from '@playwright/test';

// The tests' own ports, apart from mise run up's (8080 and on), so both can run at once. A session that works
// beside others sets BAKHYARD_E2E_PORT to a base of its own (AGENTS.md, Parallel sessions).
const base = basePort(process.env.BAKHYARD_E2E_PORT);
const gateway = `127.0.0.1:${base}`;
const web = `127.0.0.1:${base + 2}`;
// Nothing listens here: the tests answer the app's API calls in the browser, so none should reach the services.
const nowhere = `tcp://127.0.0.1:${base + 9}`;

function basePort(value: string | undefined): number {
  if (!value) {
    return 8180;
  }
  const port = Number(value);
  if (!/^\d+$/.test(value) || port < 1024 || port > 65526) {
    throw new Error(`BAKHYARD_E2E_PORT: "${value}" is not a whole number from 1024 to 65526`);
  }
  return port;
}

// SIGTERM lets a service shut down as it does under process-compose.
const stop = { signal: 'SIGTERM', timeout: 5000 } as const;

// The end-to-end tests (decision 107): the built app, served by the web service through the gateway built without
// its locks (mise run //apps/web:e2e builds both), in a desktop Chromium and a phone WebKit.
export default defineConfig({
  testDir: 'e2e',
  forbidOnly: Boolean(process.env.CI),
  use: {
    baseURL: `http://${gateway}`,
    // Not UTC and without daylight saving, as the unit tests' zone, so a mix-up of local and UTC time shows.
    timezoneId: 'Asia/Tokyo',
  },
  projects: [
    { name: 'desktop', use: devices['Desktop Chrome'] },
    { name: 'phone', use: devices['iPhone 17'] },
  ],
  webServer: [
    {
      name: 'web',
      command: '../../bin/web',
      env: { BAKHYARD_LISTEN: `tcp://${web}` },
      url: `http://${web}/healthz`,
      gracefulShutdown: stop,
    },
    {
      name: 'gateway',
      command: '../../bin/e2e/gateway',
      env: {
        BAKHYARD_LISTEN: `tcp://${gateway}`,
        BAKHYARD_PEERS: [
          ...['home', 'notes', 'notify', 'nudge', 'pulls', 'time', 'todos'].map((name) => `${name}=${nowhere}`),
          `web=tcp://${web}`,
        ].join(','),
      },
      url: `http://${gateway}/healthz`,
      gracefulShutdown: stop,
    },
  ],
});
