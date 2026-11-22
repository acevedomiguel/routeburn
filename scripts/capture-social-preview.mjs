// Renders assets-src/social-preview.html to the deployed social preview.
// Run from the repository root: node scripts/capture-social-preview.mjs
import { chromium } from '@playwright/test';

const browser = await chromium.launch();
const page = await browser.newPage({
  viewport: { width: 1200, height: 630 },
  deviceScaleFactor: 1,
});
await page.goto(`file://${process.cwd()}/assets-src/social-preview.html`);
await page.screenshot({ path: 'site/assets/images/social-preview.png' });
await browser.close();
console.log('wrote site/assets/images/social-preview.png (1200x630)');
