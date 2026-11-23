# RouteBurn project website

Static site published to GitHub Pages at
<https://acevedomiguel.github.io/routeburn/> by
`.github/workflows/pages.yml`. Plain HTML/CSS with optional vanilla
JavaScript — no build step, framework, or third-party resources.

## Local preview

```sh
python3 -m http.server 8000 --directory site
```

Open <http://localhost:8000/>. Localhost serves the site at `/`, while
production serves it at `/routeburn/`; run `node tests/site/check-links.mjs`
to validate links against the production base path before merging.

## Conventions

- Keep everything self-hosted: no external fonts, scripts, images, or embeds.
- Keep scripts and styles in external files so the Content Security Policy in
  each page's `<head>` needs no `'unsafe-inline'`.
- All imagery must use synthetic data (see `assets-src/README.md` for how the
  terminal capture and social preview are generated). Never commit real
  account exports, tokens, or identifiers.
- `404.html` uses absolute `/routeburn/`-prefixed asset links because GitHub
  Pages serves it for missing paths at any depth; the homepage uses relative
  links.
