// @ts-check
import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright config for Kinnect UI tests.
 * Focused on mobile viewports (iPhone 14 Pro, iPhone SE) as specified.
 *
 * Run: npx playwright test frontend/tests/chat.spec.js
 *
 * The dev server must be running on port 5173 (npm run dev:fe) OR
 * set TEST_BASE_URL to an already-running URL.
 */

const BASE_URL = process.env.TEST_BASE_URL || 'http://localhost:5173';

export default defineConfig({
  testDir: './frontend/tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [['html', { outputFolder: 'playwright-report', open: 'never' }], ['list']],

  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'on-first-retry',
    // Global timeout per action
    actionTimeout: 8000,
    // Block service workers so page.route() intercepts API calls reliably.
    // The sw.js network-first handler for /api/* routes intercepts fetch() calls
    // before Playwright's route handlers can see them, causing mocked routes to
    // fall through to the (non-running) backend and return 500 during tests.
    serviceWorkers: 'block',
  },

  projects: [
    // iPhone 14 Pro — primary mobile target
    {
      name: 'iPhone 14 Pro',
      use: {
        ...devices['iPhone 14 Pro'],
        // Emulate real iOS UA so any UA-sniffing code behaves correctly
        userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1',
      },
    },
    // iPhone SE — small-screen stress test (375×667)
    {
      name: 'iPhone SE',
      use: {
        ...devices['iPhone SE'],
        userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1',
      },
    },
    // Desktop Chrome — sanity check that the modal form factor works
    {
      name: 'Desktop Chrome',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Auto-start the Vite dev server if not already running
  // Comment this out if you want to manage the server manually
  webServer: {
    command: 'cd frontend && npx vite',
    url: BASE_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
    stdout: 'ignore',
    stderr: 'pipe',
  },
});
