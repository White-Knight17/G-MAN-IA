import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  timeout: 30000,
  retries: 1,
  use: {
    headless: true,
  },
  projects: [
    {
      name: 'sidecar',
      testMatch: '**/*.spec.ts',
    },
  ],
});
