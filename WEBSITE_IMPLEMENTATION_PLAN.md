# RouteBurn Static Project Website — Implementation Plan

## 1. Purpose and scope

Build and publish RouteBurn's public open-source project website using GitHub Pages. The site will explain the project, show a synthetic terminal preview, document installation and first-run setup, explain credit-runway estimation, and direct visitors to source code, releases, contribution guidelines, and the license.

This plan is **only for the public static website**. It does not replace or modify the RouteBurn application implementation plan.

### Repository configuration

- Repository root: `/Users/mike/projects/github/routeburn/`
- Git remote: `git@github.com:acevedomiguel/routeburn`
- GitHub repository: `https://github.com/acevedomiguel/routeburn`
- Published project-site URL: `https://acevedomiguel.github.io/routeburn/`
- Default branch: `main`
- Website source directory: `site/`
- Deployment: GitHub Actions to GitHub Pages
- Website implementation: plain HTML, CSS, and minimal vanilla JavaScript

The owner, repository, and default branch were read from the existing local Git configuration.

### Explicit non-goals

- Do not build a browser version of the RouteBurn TUI dashboard.
- Do not add a backend, database, CMS, JavaScript framework, or static-site generator.
- Do not request, collect, transmit, store, or process OpenRouter API keys or other credentials.
- Do not display live OpenRouter account data on GitHub Pages.
- Do not add analytics, behavioral tracking, advertising, external fonts, CDN-hosted scripts, or third-party embeds.
- Do not claim that RouteBurn, a release, a Homebrew formula, or an unfinished feature is already available.
- Do not add a version-substitution or compilation step merely to display a release number.

## 2. Desired outcome

After implementation:

1. A visitor can understand what RouteBurn does within the first screen.
2. The site clearly identifies RouteBurn as a Go TUI using Charmbracelet Bubbles components.
3. Screenshots contain only obviously synthetic data and no credentials.
4. Installation and quick-start commands match the application's actual released behavior.
5. Every page works at the project-site base path `/routeburn/`.
6. The site is responsive, keyboard accessible, and oriented toward WCAG 2.2 AA.
7. Pull requests validate site changes automatically.
8. Merges to `main` deploy through the official GitHub Pages Actions workflow.
9. No account secrets or privileged API access are needed by the website or its build.

## 3. Information architecture

Use a single primary landing page plus a custom not-found page. Keep the first version small and maintainable.

### `index.html` section order

1. **Header and navigation**
   - RouteBurn wordmark and small terminal-style mark.
   - Links to Features, Install, Quick start, FAQ, and GitHub.
   - A skip link targeting the main content.
   - Mobile navigation must not require JavaScript; wrapping or a simple accessible disclosure is acceptable.

2. **Hero**
   - Suggested headline: **Know when your OpenRouter credits will run out.**
   - Supporting copy: RouteBurn is a terminal dashboard for tracking OpenRouter credit balance, recent usage cost, remaining token limits where available, and estimated credit runway from recent consumption.
   - Primary action: view installation.
   - Secondary action: view the GitHub repository.
   - Do not use unverified claims such as “real time,” “accurate predictions,” or “official OpenRouter client.”

3. **Terminal preview**
   - Optimized synthetic screenshot or deterministic HTML/CSS terminal mockup.
   - Show balance, last-30-days cost, refresh status, and estimated days remaining.
   - Label the image as a synthetic example.
   - Never use real account identifiers, keys, balances, or usage history.

4. **Features**
   - Credit balance and recent cost monitoring.
   - Last-30-days usage view.
   - Token/limit information only when exposed by the relevant OpenRouter API.
   - Credit-runway estimate based on recent usage.
   - TUI charts built with Bubbles-compatible components.
   - Local configuration under the user's configuration directory.
   - Configurable refresh interval with API-safe minimum enforcement.

5. **Installation**
   - Homebrew command, only if the tap/formula is actually published:
     ```sh
     brew install acevedomiguel/tap/routeburn
     ```
   - Go installation command, adjusted to the actual command package:
     ```sh
     go install github.com/acevedomiguel/routeburn/cmd/routeburn@latest
     ```
   - Link to GitHub releases using:
     `https://github.com/acevedomiguel/routeburn/releases/latest`
   - Do not introduce a build step to inject the current version into the HTML.

6. **Quick start**
   - Run `routeburn`.
   - Explain that the first run asks for an OpenRouter token interactively.
   - Document the actual configuration path implemented by the application.
   - Explain how to change the refresh interval in the configuration file.
   - State that values below the application's supported OpenRouter rate-limit floor are rejected or clamped, matching actual application behavior.
   - Never place a token in a URL, command example, screenshot, website form, issue template, or log example.

7. **Understanding credit runway**
   - Explain the concept rather than promising a guaranteed prediction.
   - Display the formula only if it matches the application:
     `average daily cost = total eligible usage cost over observation window / elapsed days represented`
     `estimated days remaining = current usable credit / average daily cost`
   - State the observation window explicitly; for the initial product this is intended to be the last 30 days.
   - Explain zero-usage, insufficient-history, changing-usage, and unavailable-balance behavior.
   - Call the result an estimate, not a guarantee.

8. **Security and privacy**
   - The public website is static and does not receive OpenRouter credentials.
   - The application is separate from the website.
   - Token storage, file permissions, redaction, and network behavior must be described exactly as implemented in the application.
   - Link to security-reporting instructions if `SECURITY.md` exists; otherwise do not invent a policy URL.

9. **FAQ**
   - Is RouteBurn affiliated with OpenRouter? Answer: no, unless an affiliation actually exists.
   - Does the website see my token? Answer: no; the static site has no credential flow.
   - How often does the app refresh? Describe the implemented default and minimum.
   - How is runway calculated? Summarize the estimate and its limitations.
   - Where is configuration stored? Use the actual platform-aware path.
   - Can I use it without Homebrew? Link to Go install and releases when available.

10. **Open source and contributing**
    - Link to the repository, `CONTRIBUTING.md`, issue tracker, discussions if enabled, and license.
    - Link to the releases page rather than duplicating release notes.
    - Include the actual license name only after the repository license is confirmed.

11. **Footer**
    - Project name, repository link, license link, and a short independent-project notice.
    - No fake copyright holder or unavailable social links.

### `404.html`

- Match the main site's visual style.
- Explain that the page was not found.
- Link back to the published project homepage.
- Avoid a client-side redirect as the only navigation mechanism.
- Test asset and home links from nested nonexistent URLs, not just `/routeburn/404.html`.

## 4. Visual and interaction design

### Design direction

Use a restrained terminal-inspired dark aesthetic:

- Near-black page background with slightly lighter panels.
- High-contrast off-white primary text.
- One warm accent for burn/credit/runway emphasis and one cool accent for informational states.
- System monospace for terminal examples and metrics; system sans-serif or monospace for prose.
- Thin borders, subtle grid details, generous spacing, and clear typographic hierarchy.
- No animation is required for the core message.

Define colors as CSS custom properties. Verify normal text contrast of at least 4.5:1 and large text/UI contrast required by WCAG 2.2 AA. Do not use color alone to communicate state.

### Responsive behavior

Design mobile-first and verify at minimum:

- 320 px
- 375 px
- 768 px
- 1024 px
- 1440 px

Requirements:

- No horizontal page scrolling at 320 px.
- Long commands wrap or scroll within their code container without moving the page.
- Navigation remains operable at every width.
- Screenshots scale without distortion and retain explicit dimensions.
- Touch targets meet WCAG 2.2 target-size expectations where practical.
- Content remains usable at 200% browser zoom.

### Motion

- Respect `prefers-reduced-motion: reduce`.
- Disable nonessential transitions and animations under reduced motion.
- Do not autoplay a typing effect, flashing cursor, or continuously updating fake metrics.
- No content may flash more than three times per second.

### Minimal JavaScript

JavaScript is optional for core reading and navigation. Appropriate uses include:

- Accessible copy buttons for installation commands.
- A small navigation disclosure if CSS/wrapping alone is insufficient.
- Enhancing external-link or section behavior only when necessary.

Rules:

- Keep all JavaScript in local `.js` files; no inline scripts.
- Use no JavaScript for rendering essential content.
- Provide a text selection fallback if copy buttons are unavailable.
- Announce copy success without moving keyboard focus unexpectedly.
- Handle clipboard denial without throwing an uncaught error.

## 5. File layout

Create only the files needed by the site and its validation:

```text
routeburn/
├── .github/
│   └── workflows/
│       ├── pages.yml
│       └── site-check.yml
├── site/
│   ├── .nojekyll
│   ├── index.html
│   ├── 404.html
│   ├── robots.txt
│   ├── sitemap.xml
│   ├── favicon.svg
│   ├── assets/
│   │   ├── css/
│   │   │   └── site.css
│   │   ├── js/
│   │   │   └── site.js
│   │   └── images/
│   │       ├── terminal-demo.avif
│   │       ├── terminal-demo.webp
│   │       └── social-preview.png
│   └── README.md
├── tests/
│   └── site/
│       ├── site.spec.js
│       └── check-links.mjs
├── package.json
├── package-lock.json
├── playwright.config.js
└── IMPLEMENTATION_PLAN.md
```

Notes:

- `site/.nojekyll` disables Jekyll processing and preserves all files as authored.
- `site/README.md` may document local preview and asset conventions; it is not deployed navigation.
- If copy buttons are omitted, omit `site.js` rather than shipping an empty script.
- Do not place credentials, real exports, or application configuration examples containing secrets in `site/`.

## 6. URL, asset, and metadata rules

Because this is a project site served below `/routeburn/`, do not assume the website is hosted at `/`.

### Internal URLs

- Prefer relative asset references from the homepage, for example:
  - `assets/css/site.css`
  - `assets/js/site.js`
  - `assets/images/terminal-demo.webp`
- Use fragment links such as `#install` for same-page sections.
- Avoid hardcoding a development hostname.
- Ensure `404.html` works from nested missing paths; use project-root-aware links there if relative paths cannot resolve reliably.
- Test the published base path, not only a local root server.

### Canonical and social metadata

For `index.html`, include:

- A concise, unique `<title>`.
- A factual meta description.
- Canonical URL: `https://acevedomiguel.github.io/routeburn/`.
- Open Graph `og:type`, `og:title`, `og:description`, `og:url`, and `og:image`.
- Twitter/X card metadata if desired, without adding a tracker or embed.
- Absolute Open Graph image URL under the published site.
- A useful favicon and theme color.

The social-preview image must contain only synthetic data. Include meaningful alternative text for content images; decorative images use empty alternative text.

### `robots.txt`

Allow normal crawling and declare the absolute sitemap URL:

```txt
User-agent: *
Allow: /

Sitemap: https://acevedomiguel.github.io/routeburn/sitemap.xml
```

### `sitemap.xml`

Initially list the canonical homepage only. Add `404.html` to neither the sitemap nor navigation. Add a last-modified value only when it can be maintained accurately; otherwise omit it.

## 7. Security and privacy requirements

### Content Security Policy

Use a restrictive meta CSP compatible with static GitHub Pages hosting. Start with:

```html
<meta http-equiv="Content-Security-Policy" content="default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; font-src 'self'; connect-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; media-src 'self'; worker-src 'none'">
```

Implementation constraints:

- Keep scripts and styles external so `'unsafe-inline'` is unnecessary.
- Do not add remote image, font, script, or API origins.
- Do not add `frame-ancestors` to the meta policy and assume it works; that directive requires an HTTP header, which this plan does not rely on.
- If an inline SVG favicon or data URL is introduced, adjust only the necessary directive and revalidate.
- Test browser consoles for CSP violations on the deployed site.

### Additional checks

- No inline event handlers such as `onclick`.
- No `eval`, `new Function`, or dynamic script injection.
- Use `textContent`, not `innerHTML`, for JavaScript status messages.
- External links that open a new tab require `rel="noopener noreferrer"`; prefer same-tab navigation.
- Do not put sensitive values in query strings or fragments.
- Do not add service workers in the initial version.
- Treat GitHub Actions tokens as privileged: grant only the permissions shown in the workflows.

## 8. Terminal imagery

Preferred source: an actual RouteBurn screenshot after the application exists, populated entirely with fixture data. Until then, use a clearly labeled design mockup and do not present it as a working release.

Required synthetic values and behavior:

- Use fictional balances, dates, token counts, and costs.
- Show a plausible runway label such as “estimated runway,” not “days guaranteed.”
- Ensure no string resembles a live OpenRouter secret.
- Do not use a real user's terminal history or account export.
- Include the synthetic-data label in nearby visible text.

Asset pipeline:

1. Keep an editable source outside the deployed asset directory if needed.
2. Export AVIF and WebP with the same visual content.
3. Provide WebP as fallback if AVIF is unsupported.
4. Set intrinsic `width` and `height` to prevent layout shift.
5. Lazy-load imagery below the hero; do not lazy-load a genuine LCP hero image.
6. Keep the social preview at a conventional 1200 × 630 px size.
7. Optimize without making terminal text unreadable.

## 9. Local development and preview

From the repository root:

```sh
python3 -m http.server 8000 --directory site
```

Open:

```text
http://localhost:8000/
```

Then verify:

- Homepage and all section links.
- `404.html` directly and through a nested missing-path test on the deployed site.
- Browser console has no JavaScript, asset, mixed-content, or CSP errors.
- Site works with JavaScript disabled except for explicitly enhanced controls.
- No network request goes to OpenRouter or a third-party origin.

Localhost does not reproduce the `/routeburn/` base path by itself. Before deployment, run the internal-link checker against the intended public base path and inspect the Actions-built artifact.

## 10. Pull-request validation

Add `.github/workflows/site-check.yml`. It must run for pull requests that change:

- `site/**`
- `tests/site/**`
- `package.json`
- `package-lock.json`
- `playwright.config.js`
- `.github/workflows/site-check.yml`
- `.github/workflows/pages.yml`

### Pinned tooling policy

- Use `actions/checkout@v6` and `actions/setup-node@v6`.
- Select one supported Node.js LTS version and record it explicitly in the workflow.
- Commit `package.json` and `package-lock.json` with exact dependency versions.
- Install with `npm ci`; never use an unpinned `npx <tool>@latest` in CI.
- Pin browser test tooling through the same lockfile.
- Cache npm using the lockfile through `actions/setup-node@v6`.
- Update pins deliberately through reviewed pull requests.

### Validation stages

1. **HTML validation**
   - Validate document structure, duplicate IDs, image alternatives, labels, and invalid attributes.
   - Treat validation errors as failures.

2. **CSS and JavaScript syntax**
   - Parse every stylesheet.
   - Run `node --check site/assets/js/site.js` when the file exists.
   - Run syntax checks for test and configuration files.

3. **Internal-link and asset checks**
   - Start a local static server.
   - Check every local `href`, `src`, `srcset`, canonical, sitemap, and manifest reference used by the site.
   - Verify fragment targets exist and IDs are unique.
   - Account for the `/routeburn/` publishing base.
   - Do not fail a build solely because an external third-party site is temporarily unavailable; validate repository/release URL syntax and check external availability separately when appropriate.

4. **Accessibility automation**
   - Use Playwright with axe-core at mobile and desktop widths.
   - Fail on serious and critical violations; aim to resolve all reported violations.
   - Test both `index.html` and `404.html`.
   - Automation supplements, not replaces, keyboard and screen-reader review.

5. **Responsive rendering**
   - Capture homepage snapshots at 320, 768, and 1440 px.
   - Assert `document.documentElement.scrollWidth <= window.innerWidth`.
   - Check that main navigation, install commands, images, and footer remain visible and usable.
   - Emulate reduced motion and verify animations are disabled.

6. **Metadata and security**
   - Require title, description, canonical URL, Open Graph fields, favicon, and image dimensions.
   - Verify no HTTP asset references and no unexpected third-party origins.
   - Scan HTML/CSS/JS for credential-like test fixtures and reject real account exports.
   - Verify the CSP does not contain `'unsafe-inline'`, `'unsafe-eval'`, or a wildcard source.

7. **Performance smoke checks**
   - Run Lighthouse against the locally served static site in CI or as a repeatable pre-merge command.
   - Save reports as workflow artifacts.
   - Use deterministic local asset-size checks as hard budgets; treat lab timing scores as signals rather than universal user guarantees.

## 11. Performance budgets

Apply budgets to the initial homepage load with an empty cache:

| Resource or metric | Budget |
| --- | ---: |
| HTML | ≤ 50 KiB uncompressed |
| CSS | ≤ 30 KiB gzip |
| JavaScript | ≤ 15 KiB gzip |
| Initial font transfer | 0 KiB third-party; system fonts preferred |
| LCP image | ≤ 180 KiB where practical |
| Total initial transfer | ≤ 350 KiB excluding an explicitly documented optional below-fold asset |
| Cumulative Layout Shift | ≤ 0.1 |
| Lighthouse LCP in the agreed lab profile | ≤ 2.5 s |
| Lighthouse accessibility/best-practices/SEO | Target 100; investigate any regression |

If a budget is exceeded, optimize the asset or remove unnecessary code rather than silently increasing the budget. Record any justified exception in the pull request.

## 12. GitHub Pages deployment workflow

Create `.github/workflows/pages.yml` with the following template. This uploads `site/` directly; there is no compilation step.

```yaml
name: Deploy static project site to GitHub Pages

on:
  push:
    branches:
      - main
    paths:
      - site/**
      - .github/workflows/pages.yml
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: pages
  cancel-in-progress: false

jobs:
  build:
    name: Validate and upload site artifact
    runs-on: ubuntu-latest
    steps:
      - name: Check out repository
        uses: actions/checkout@v6

      - name: Configure GitHub Pages
        uses: actions/configure-pages@v5

      - name: Check required static files
        shell: bash
        run: |
          set -euo pipefail
          test -f site/index.html
          test -f site/404.html
          test -f site/.nojekyll
          test -f site/robots.txt
          test -f site/sitemap.xml
          test -f site/favicon.svg

      - name: Upload GitHub Pages artifact
        uses: actions/upload-pages-artifact@v4
        with:
          path: site

  deploy:
    name: Deploy to GitHub Pages
    needs: build
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pages: write
      id-token: write
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - name: Deploy Pages artifact
        id: deployment
        uses: actions/deploy-pages@v4
```

### Workflow behavior

- Pushes deploy only when `site/**` or the Pages workflow changes on `main`.
- Maintainers can run the workflow manually with **Run workflow**.
- The deploy job cannot run unless build and artifact upload succeed.
- The artifact contains exactly the static site directory.
- The `github-pages` environment exposes the deployed URL.
- Concurrent deployments queue; an active deployment is not cancelled.
- No OpenRouter token or application secret is required.

If application release automation also modifies site content, either allow that automation to commit under `site/**` or invoke this workflow deliberately through a maintained mechanism. Do not broaden triggers without considering duplicate deployments.

## 13. One-time GitHub configuration

Perform these steps after the repository and workflow exist:

1. Push `.github/workflows/pages.yml` and the initial `site/` content to the default branch.
2. In the repository, open **Settings → Pages**.
3. Under **Build and deployment → Source**, select **GitHub Actions**.
4. Open **Settings → Environments → `github-pages`** after the environment is created.
5. Restrict deployment to the default branch, or otherwise ensure only trusted default-branch workflows can deploy.
6. Require relevant site validation checks on pull requests through branch protection or rulesets.
7. Run the Pages workflow manually if the initial configuration does not start it automatically.
8. Confirm the workflow finishes successfully and reports the Pages URL.
9. Visit `https://acevedomiguel.github.io/routeburn/` and verify the deployed artifact, not just local files.

Do not switch Pages back to branch-based publishing; the custom Actions workflow is the publishing source.

## 14. Release and content-maintenance strategy

- Use generic release links such as `https://github.com/acevedomiguel/routeburn/releases/latest`.
- Do not hardcode a current release number in static HTML.
- Update installation commands only when the corresponding distribution channel works.
- Keep screenshots synchronized with major UI changes, but never automate capture using a real account.
- Review FAQ statements whenever configuration paths, refresh limits, or runway logic change.
- Add a documentation page only when the landing page becomes difficult to scan; the initial site should remain one page.
- Keep website deployment independent from application binary releases unless a deliberate content update is needed.

## 15. Rollback and recovery

### Bad deployed content

1. Revert the offending site commit on `main` using the project's normal reviewed Git workflow.
2. Push the revert; the `site/**` path trigger starts a new Pages deployment.
3. If no content path changes but redeployment is required, use **Run workflow** on the last known-good revision through the repository's supported workflow-dispatch process.
4. Confirm the deployed page and key assets after completion.

### Failed deployment

- A failed build or deploy must not be described as published.
- Inspect the failed job and preserve the current live site.
- Do not cancel an in-progress known-good deployment merely to force a new one.
- Fix the workflow or content in a pull request and rerun validation.

### Broken release or installation instructions

- Correct the documentation immediately if the published commands do not work.
- If the application release itself is faulty, follow the application release-runbook separately; do not hide the issue with website-only edits.

## 16. Implementation phases

### Phase 0 — Confirm facts and placeholders

- [ ] Confirm GitHub owner and repository name.
- [ ] Confirm default branch is `main`.
- [ ] Confirm actual application command name and Go package path.
- [ ] Confirm Homebrew tap/formula availability before publishing its command.
- [ ] Confirm configuration path, refresh behavior, runway formula, and license.
- [ ] Mark unreleased functionality honestly or withhold it until available.

### Phase 1 — Static content and structure

- [ ] Create `site/` and `.nojekyll`.
- [ ] Build semantic `index.html` with the sections in this plan.
- [ ] Add `404.html`, favicon, robots, and sitemap.
- [ ] Add metadata and canonical/Open Graph URLs.
- [ ] Use only local static resources.
- [ ] Verify the site contains no credential fields or API calls.

### Phase 2 — Styling and imagery

- [ ] Implement the dark terminal-inspired design in `site.css`.
- [ ] Add responsive layouts and visible focus states.
- [ ] Add reduced-motion and forced-colors considerations.
- [ ] Produce optimized synthetic terminal and social images.
- [ ] Verify performance budgets and image dimensions.

### Phase 3 — Optional progressive enhancement

- [ ] Add only necessary local JavaScript.
- [ ] Make copy/navigation controls keyboard and screen-reader usable.
- [ ] Confirm essential content works without JavaScript.
- [ ] Check the browser console and CSP.

### Phase 4 — Automated validation

- [ ] Add exact-version development dependencies and commit the lockfile.
- [ ] Add HTML, link, syntax, accessibility, responsive, metadata, and security checks.
- [ ] Add Playwright tests and local-server fixtures.
- [ ] Add `site-check.yml` with pinned actions and `npm ci`.
- [ ] Prove that a deliberately broken link or accessibility violation fails CI, then remove the deliberate failure.

### Phase 5 — Pages deployment

- [ ] Add `pages.yml` exactly with least-privilege permissions and the official Pages actions.
- [ ] Select GitHub Actions as the Pages publishing source.
- [ ] Restrict the `github-pages` environment to trusted default-branch deployment.
- [ ] Merge the initial site and confirm deployment succeeds.
- [ ] Verify the reported page URL and `/routeburn/` asset loading.

### Phase 6 — Manual launch review

- [ ] Test keyboard-only navigation, skip link, focus order, and visible focus.
- [ ] Test at 320, 375, 768, 1024, and 1440 px plus 200% zoom.
- [ ] Test reduced motion and high-contrast/forced-colors behavior where supported.
- [ ] Perform a screen-reader smoke test of headings, landmarks, links, images, and FAQ.
- [ ] Test 404 behavior from a nested nonexistent path.
- [ ] Validate canonical, sitemap, robots, Open Graph, and social image URLs.
- [ ] Verify no unexpected network requests or console errors.
- [ ] Confirm every product statement matches the actual application state.

## 17. Acceptance criteria

The website is ready only when all of the following are true:

1. `site/` runs with the documented one-command local preview and no install/build step.
2. The deployed site loads correctly at `https://acevedomiguel.github.io/routeburn/` with no missing assets.
3. The homepage communicates purpose, features, installation, quick start, runway limitations, privacy, contribution, releases, and licensing without false availability claims.
4. All terminal imagery uses synthetic data and contains no credentials or real account details.
5. The site makes no OpenRouter requests and includes no third-party trackers, fonts, scripts, or embeds.
6. HTML, link, syntax, accessibility, responsive, metadata, security, and performance checks pass.
7. Keyboard navigation, reduced motion, 320 px layout, and 200% zoom are usable.
8. Pull requests run the site-validation workflow.
9. Default-branch site changes deploy through `pages.yml`; deployment depends on a successful artifact build/upload.
10. The workflow uses only `contents: read`, `pages: write`, and `id-token: write` as needed, does not cancel active Pages deployments, and reports the deployed URL.
11. Rollback by reverting a site commit and redeploying has been documented and is workable.
12. No real token or live account information appears in source, screenshots, workflow logs, test fixtures, issues, or deployed output.

## 18. Official references

- GitHub Pages publishing source configuration:
  https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site
- GitHub Pages custom Actions workflows:
  https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages
- WCAG 2.2:
  https://www.w3.org/TR/WCAG22/
- Open Graph protocol:
  https://ogp.me/

Use current official GitHub documentation if action versions or platform requirements change; update workflow pins through reviewed changes rather than silently switching to unpinned tooling.
