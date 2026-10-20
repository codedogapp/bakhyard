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
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      include: ['src/**/*.{ts,svelte}'],
      exclude: ['src/**/*.test.ts', 'src/test-setup.ts', 'src/main.ts'],
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
