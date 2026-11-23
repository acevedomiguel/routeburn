// Lighthouse smoke check against the locally served site. Uses the Chromium
// installed by Playwright so no separate Chrome is needed. Writes reports to
// lighthouse-report/ (CI uploads them as artifacts). Lab timing scores are
// signals; the hard budgets live in tests/site/site.spec.js.
import { writeFile, mkdir } from 'node:fs/promises';
import { chromium } from '@playwright/test';
import { launch } from 'chrome-launcher';
import lighthouse from 'lighthouse';
import { serveSite } from '../tests/site/serve.mjs';

const server = await serveSite(8323);
const url = 'http://localhost:8323/routeburn/';

const chrome = await launch({
  chromePath: chromium.executablePath(),
  chromeFlags: ['--headless=new', '--no-sandbox'],
});

const result = await lighthouse(url, {
  port: chrome.port,
  output: 'html',
  logLevel: 'error',
  // robots-txt loads through the page context, which our strict
  // connect-src 'none' CSP blocks by design; the audit cannot run in the
  // lab. robots.txt content is asserted by tests/site/check-links.mjs.
  skipAudits: ['robots-txt'],
});

await chrome.kill();
await mkdir('lighthouse-report', { recursive: true });
await writeFile('lighthouse-report/index.html', result.report);
server.close();

const scores = Object.fromEntries(
  Object.entries(result.lhr.categories).map(([key, cat]) => [key, cat.score]),
);
console.log('lighthouse scores:', scores);

// Accessibility, best-practices, and SEO target 100; investigate any drop.
for (const key of ['accessibility', 'best-practices', 'seo']) {
  if ((scores[key] ?? 0) < 1) {
    console.error(`lighthouse ${key} below target: ${scores[key]}`);
    process.exit(1);
  }
}
