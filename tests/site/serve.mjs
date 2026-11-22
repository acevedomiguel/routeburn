// Local static server that emulates GitHub Pages for the project site:
// the contents of site/ are mounted at /routeburn/, and missing paths are
// answered with site/404.html and a real 404 status, like Pages does.
//
// Run directly (node tests/site/serve.mjs [port]) or import { serveSite }.
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { extname, join, resolve, sep } from 'node:path';

const SITE_ROOT = new URL('../../site/', import.meta.url).pathname;
const BASE = '/routeburn/';

const TYPES = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.webp': 'image/webp',
  '.avif': 'image/avif',
  '.txt': 'text/plain; charset=utf-8',
  '.xml': 'application/xml; charset=utf-8',
  '.json': 'application/json',
};

export function serveSite(port = 8321) {
  const siteRoot = resolve(SITE_ROOT);

  const server = createServer(async (req, res) => {
    const url = new URL(req.url, 'http://localhost');
    let path;
    try {
      path = decodeURIComponent(url.pathname);
    } catch {
      res.writeHead(400, { 'Content-Type': TYPES['.txt'] });
      res.end('malformed request path');
      return;
    }

    if (path === '/' || path === '/routeburn') {
      res.writeHead(302, { Location: BASE });
      res.end();
      return;
    }
    // Crawlers only read robots.txt from the domain root. Serve the project's
    // file there too so lab checks (e.g. Lighthouse) can validate its content;
    // on the real user-site domain it is owned by the .github.io repository.
    if (path === '/robots.txt') {
      const robots = await readFile(join(SITE_ROOT, 'robots.txt'));
      res.writeHead(200, { 'Content-Type': TYPES['.txt'] });
      res.end(robots);
      return;
    }
    if (!path.startsWith(BASE)) {
      res.writeHead(404, { 'Content-Type': TYPES['.txt'] });
      res.end('outside the project base path');
      return;
    }

    const rel = path.slice(BASE.length);
    let file = resolve(join(siteRoot, rel));
    if (rel === '' || path.endsWith('/')) {
      file = join(file, 'index.html');
    }
    if (file !== siteRoot && !file.startsWith(siteRoot + sep)) {
      res.writeHead(404, { 'Content-Type': TYPES['.txt'] });
      res.end('outside the project base path');
      return;
    }

    try {
      const body = await readFile(file);
      res.writeHead(200, { 'Content-Type': TYPES[extname(file)] || 'application/octet-stream' });
      res.end(body);
    } catch {
      const notFound = await readFile(join(SITE_ROOT, '404.html'));
      res.writeHead(404, { 'Content-Type': TYPES['.html'] });
      res.end(notFound);
    }
  });
  return new Promise((resolve) => {
    server.listen(port, () => resolve(server));
  });
}

if (process.argv[1] && process.argv[1].endsWith('serve.mjs')) {
  const port = Number(process.argv[2]) || 8321;
  serveSite(port).then(() => {
    console.log(`serving site/ at http://localhost:${port}${BASE}`);
  });
}
