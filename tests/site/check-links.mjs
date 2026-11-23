// Internal link and asset checker for the project site. Serves site/ at the
// production base path /routeburn/ and verifies every local href, src, and
// srcset reference, every same-page fragment, unique IDs, the canonical URL,
// the sitemap, and robots.txt. External URLs are syntax-checked only — a
// temporarily unreachable third party must not fail the build.
import { readFile } from 'node:fs/promises';
import { serveSite } from './serve.mjs';

const ORIGIN = 'http://localhost:8322';
const BASE = '/routeburn/';
const PUBLISHED = 'https://acevedomiguel.github.io/routeburn/';
const SITE = new URL('../../site/', import.meta.url).pathname;

const failures = [];
const fail = (msg) => failures.push(msg);

const read = (name) => readFile(`${SITE}${name}`, 'utf8');
const attrs = (html, attr) =>
  [...html.matchAll(new RegExp(`${attr}="([^"]*)"`, 'g'))].map((m) => m[1]);

const pages = ['index.html', '404.html'];
const idsByPage = new Map();
let totalIds = 0;

async function get(path) {
  const res = await fetch(`${ORIGIN}${path}`);
  await res.arrayBuffer();
  return res.status;
}

const server = await serveSite(8322);

for (const page of pages) {
  const html = await read(page);

  // Collect IDs for fragment and uniqueness checks (uniqueness is per page).
  const pageIds = new Set();
  for (const id of attrs(html, 'id')) {
    if (pageIds.has(id)) fail(`duplicate id "${id}" in ${page}`);
    pageIds.add(id);
  }
  idsByPage.set(page, pageIds);
  totalIds += pageIds.size;

  const refs = [
    ...attrs(html, 'href'),
    ...attrs(html, 'src'),
    ...attrs(html, 'srcset').flatMap((s) => s.split(',').map((part) => part.trim().split(' ')[0])),
  ];

  for (const ref of refs) {
    if (!ref || ref.startsWith('data:')) continue;

    if (ref.startsWith('#')) {
      if (!idsByPage.get(page).has(ref.slice(1))) fail(`${page}: fragment ${ref} has no matching id`);
      continue;
    }

    if (/^https?:\/\//.test(ref)) {
      try {
        new URL(ref);
      } catch {
        fail(`${page}: malformed external URL ${ref}`);
      }
      continue;
    }

    // Internal reference: resolve relative to the page, or as an absolute
    // path under the site origin (used by 404.html).
    let path;
    if (ref.startsWith('/')) {
      path = ref;
      if (!path.startsWith(BASE)) fail(`${page}: absolute path ${ref} is outside ${BASE}`);
    } else {
      path = BASE + (page === 'index.html' ? '' : '') + ref;
    }
    const status = await get(path);
    if (status !== 200) fail(`${page}: ${ref} → ${path} returned ${status}`);
  }
}

// Canonical and sitemap consistency.
const index = await read('index.html');
const canonical = attrs(index, 'href').find((h) => h === PUBLISHED);
if (!canonical) fail(`index.html: canonical ${PUBLISHED} missing`);

const sitemap = await read('sitemap.xml');
if (!sitemap.includes(`<loc>${PUBLISHED}</loc>`)) fail('sitemap.xml: homepage URL missing');
if (sitemap.includes('404')) fail('sitemap.xml: must not list 404.html');

const robots = await read('robots.txt');
if (!robots.includes(`Sitemap: ${PUBLISHED}sitemap.xml`)) fail('robots.txt: sitemap URL missing');

// Required static files must exist and be served.
for (const file of ['index.html', '404.html', 'robots.txt', 'sitemap.xml', 'favicon.svg', '.nojekyll']) {
  const status = await get(BASE + file);
  if (status !== 200) fail(`${file} not served at ${BASE}${file} (${status})`);
}

server.close();

if (failures.length) {
  console.error('link check failed:');
  for (const f of failures) console.error(`  - ${f}`);
  process.exit(1);
}
console.log(`link check passed (${pages.length} pages, ${totalIds} ids)`);
