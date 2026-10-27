import { defineConfig, devices } from '@playwright/test';

// The tests' own ports, apart from mise run up's (8080 to 8085), so both can run at once.
const gateway = '127.0.0.1:8180';
const web = '127.0.0.1:8182';
// Nothing listens here: the tests answer the app's API calls in the browser, so none should reach the services.
const nowhere = 'tcp://127.0.0.1:8189';

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
        BAKHYARD_PEERS: `home=${nowhere},notes=${nowhere},pulls=${nowhere},todos=${nowhere},web=tcp://${web}`,
      },
      url: `http://${gateway}/healthz`,
      gracefulShutdown: stop,
    },
  ],
});
