# Website asset sources

Editable sources for the images published under `site/assets/images/`. Nothing
in this directory is deployed; only the generated outputs in `site/` ship.

All imagery uses the synthetic fixture from `internal/ui/fixture.go` rendered
by the hidden `routeburn demo` command. Never capture a real account.

## Terminal demo

```sh
make build
vhs assets-src/terminal-demo.tape   # writes assets-src/terminal-demo.png
node scripts/build-site-images.mjs  # writes site/assets/images/terminal-demo.{avif,webp}
```

VHS is a local-only tool (`brew install vhs`); CI never regenerates images.

## Social preview

```sh
node scripts/capture-social-preview.mjs  # writes site/assets/images/social-preview.png
```

Renders `assets-src/social-preview.html` with Playwright at 1200 x 630 and
commits the result. Edit the HTML to change the design.
