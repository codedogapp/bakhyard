import { svelte } from '@sveltejs/vite-plugin-svelte';
import { svelteTesting } from '@testing-library/svelte/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [
    svelte({
      compilerOptions: {
        // Runes mode for our components, so Svelte's deprecated legacy syntax fails; libraries keep their own mode.
        runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true),
      },
    }),
    svelteTesting(),
  ],
  server: {
    proxy: { '/api': 'http://127.0.0.1:8080' },
  },
  build: {
    // The web service embeds the app from here; Vite empties a directory outside its root only when told to.
    outDir: '../../services/web/dist/app',
    emptyOutDir: true,
    // The note renderer's chunk holds KaTeX, about 540 kB in all; it loads only with a note's page (decision 99).
    chunkSizeWarningLimit: 600,
  },
  test: {
    // The end-to-end specs in e2e/ are Playwright's.
    include: ['src/**/*.test.ts', 'overrides/**/*.test.ts'],
    environment: 'jsdom',
    // A fixed zone that isn't UTC and has no daylight saving, so a test that mixes up local and UTC time fails everywhere.
    env: { TZ: 'Asia/Tokyo' },
    setupFiles: ['./src/test-setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      include: ['src/**/*.{ts,svelte}', 'overrides/**/*.ts'],
      exclude: ['**/*.test.ts', 'src/test-setup.ts', 'src/main.ts'],
      thresholds: {
        lines: 80,
        statements: 80,
        functions: 80,
        branches: 80,
        perFile: { lines: 70, statements: 70, functions: 70, branches: 70 },
      },
    },
  },
});
