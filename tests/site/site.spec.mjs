import { test, expect } from '@playwright/test';
import { AxeBuilder } from '@axe-core/playwright';
import { readFileSync, statSync } from 'node:fs';
import { gzipSync } from 'node:zlib';

const BASE = '/routeburn/';
const PUBLISHED = 'https://acevedomiguel.github.io/routeburn/';

// --- Accessibility --------------------------------------------------------

for (const width of [375, 1280]) {
  for (const pagePath of [BASE, `${BASE}404.html`]) {
    test(`axe: no serious violations on ${pagePath} at ${width}px`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.goto(pagePath);
      const results = await new AxeBuilder({ page }).analyze();
      const blocking = results.violations.filter(
        (v) => v.impact === 'serious' || v.impact === 'critical',
      );
      expect(blocking, JSON.stringify(blocking, null, 2)).toEqual([]);
    });
  }
}

// --- Responsive layout -----------------------------------------------------

for (const width of [320, 768, 1440]) {
  test(`no horizontal scroll at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto(BASE);
    const fits = await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    );
    expect(fits).toBe(true);
  });
}

test('main navigation, install commands, and footer are usable at 320px', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 700 });
  await page.goto(BASE);
  await expect(page.getByRole('link', { name: 'Features' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Install RouteBurn' })).toBeVisible();
  await page.locator('#install').scrollIntoViewIfNeeded();
  await expect(page.locator('#install pre').first()).toBeVisible();
  await page.locator('.site-footer').scrollIntoViewIfNeeded();
  await expect(page.locator('.site-footer')).toBeVisible();
});

test('reduced motion disables smooth scrolling', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto(BASE);
  const behavior = await page.evaluate(
    () => getComputedStyle(document.documentElement).scrollBehavior,
  );
  expect(behavior).toBe('auto');
});

// --- Metadata and security -------------------------------------------------

test('homepage metadata is complete', async ({ page }) => {
  await page.goto(BASE);
  await expect(page).toHaveTitle(/RouteBurn/);
  const description = await page.getAttribute('meta[name="description"]', 'content');
  expect(description?.length).toBeGreaterThan(50);
  const canonical = await page.getAttribute('link[rel="canonical"]', 'href');
  expect(canonical).toBe(PUBLISHED);
  for (const property of ['og:type', 'og:title', 'og:description', 'og:url', 'og:image']) {
    const content = await page.getAttribute(`meta[property="${property}"]`, 'content');
    expect(content, property).toBeTruthy();
  }
  const ogImage = await page.getAttribute('meta[property="og:image"]', 'content');
  expect(ogImage).toBe(`${PUBLISHED}assets/images/social-preview.png`);
  expect(await page.getAttribute('link[rel="icon"]', 'href')).toBeTruthy();
});

test('CSP is restrictive on both pages', async ({ page }) => {
  for (const pagePath of [BASE, `${BASE}404.html`]) {
    await page.goto(pagePath);
    const csp = await page.getAttribute('meta[http-equiv="Content-Security-Policy"]', 'content');
    expect(csp, pagePath).toBeTruthy();
    expect(csp).not.toContain("'unsafe-inline'");
    expect(csp).not.toContain("'unsafe-eval'");
    expect(csp).not.toMatch(/(script-src|style-src|img-src)[^;]*\*/);
    expect(csp).toContain("connect-src 'none'");
  }
});

test('no third-party requests and no console errors', async ({ page }) => {
  const external = [];
  page.on('request', (request) => {
    if (!request.url().startsWith('http://localhost:8321/')) {
      external.push(request.url());
    }
  });
  const consoleErrors = [];
  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(msg.text());
  });
  page.on('pageerror', (err) => consoleErrors.push(String(err)));
  await page.goto(BASE);
  await page.waitForLoadState('networkidle');
  expect(external).toEqual([]);
  expect(consoleErrors).toEqual([]);
});

test('content contains no credential-like strings', async ({ page }) => {
  await page.goto(BASE);
  const html = await page.content();
  expect(html).not.toMatch(/sk-or-[a-z0-9-]{8,}/i);
});

test('terminal preview has explicit dimensions and alt text', async ({ page }) => {
  await page.goto(BASE);
  const img = page.locator('.terminal-figure img');
  await expect(img).toHaveAttribute('width', '1280');
  await expect(img).toHaveAttribute('height', '800');
  const alt = await img.getAttribute('alt');
  expect(alt?.length).toBeGreaterThan(40);
});

// --- 404 behavior ----------------------------------------------------------

test('missing nested path serves the styled 404 with working assets', async ({ page }) => {
  const css = [];
  page.on('response', (response) => {
    if (response.url().endsWith('site.css')) css.push(response.status());
  });
  const response = await page.goto(`${BASE}no/such/page`);
  expect(response.status()).toBe(404);
  await expect(page.getByRole('heading', { name: /page not found/i })).toBeVisible();
  expect(css).toEqual([200]);
  await expect(page.getByRole('link', { name: 'Back to the RouteBurn homepage' })).toHaveAttribute(
    'href',
    BASE,
  );
});

// --- Progressive enhancement -------------------------------------------------

test('copy button copies the command and announces it', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.goto(BASE);
  const button = page.locator('.copy-button').first();
  await button.click();
  await expect(button).toHaveText('copied');
  const clip = await page.evaluate(() => navigator.clipboard.readText());
  expect(clip).toContain('brew tap acevedomiguel/routeburn');
});

// --- Performance budgets (deterministic, spec section 11) --------------------

const KiB = 1024;
const read = (path) => readFileSync(new URL(`../../site/${path}`, import.meta.url));
const gzSize = (path) => gzipSync(read(path)).length;

test('asset size budgets', () => {
  expect(read('index.html').length).toBeLessThanOrEqual(50 * KiB);
  expect(gzSize('assets/css/site.css')).toBeLessThanOrEqual(30 * KiB);
  expect(gzSize('assets/js/site.js')).toBeLessThanOrEqual(15 * KiB);
  expect(statSync(new URL('../../site/assets/images/terminal-demo.avif', import.meta.url)).size)
    .toBeLessThanOrEqual(180 * KiB);

  const initial =
    gzSize('index.html') +
    gzSize('assets/css/site.css') +
    gzSize('assets/js/site.js') +
    statSync(new URL('../../site/assets/images/terminal-demo.avif', import.meta.url)).size;
  expect(initial).toBeLessThanOrEqual(350 * KiB);
});

// --- Local server containment (regression: path traversal review finding) ---

const rawGet = (path) =>
  new Promise((resolve, reject) => {
    import('node:http').then(({ get }) => {
      get({ host: 'localhost', port: 8321, path }, (res) => {
        res.resume();
        res.on('end', () =>
          resolve({ status: res.statusCode, location: res.headers.location || '' }),
        );
      }).on('error', reject);
    });
  });

test('server rejects traversal outside the site root', async () => {
  // SITE_ROOT/../package.json must never be served.
  expect((await rawGet('/routeburn/%2e%2e/package.json')).status).toBe(404);
  expect((await rawGet('/routeburn/..%2f..%2fpackage.json')).status).toBe(404);

  // The URL parser resolves %2e%2e before we see it; a redirect back into
  // the base path is safe, serving anything outside it is not.
  const dotdot = await rawGet('/routeburn/%2e%2e');
  if (dotdot.status === 302) {
    expect(dotdot.location.startsWith('/routeburn')).toBe(true);
  } else {
    expect(dotdot.status).toBe(404);
  }
});

test('server answers malformed percent-encoding with 400, not a crash', async () => {
  expect((await rawGet('/routeburn/%zz')).status).toBe(400);
});
