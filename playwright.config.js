import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/site',
  timeout: 30000,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? 'github' : 'list',
  webServer: {
    command: 'node tests/site/serve.mjs 8321',
    port: 8321,
    reuseExistingServer: !process.env.CI,
  },
  use: {
    baseURL: 'http://localhost:8321',
  },
});
