# RouteBurn project instructions

Common engineering, Go, GitHub Actions, and Homebrew conventions are loaded from
the tagged central documents in `opencode.jsonc`. This file contains only
RouteBurn-specific contracts.

## Product and credential boundaries

- RouteBurn is read-only. It must not send inference requests, purchase credits,
  or create, rotate, or revoke OpenRouter keys. `POST /analytics/query` is the
  documented read-only analytics operation.
- Keep management and regular API keys separate. `OPENROUTER_API_KEY` is a
  fallback for the regular key only and must never satisfy the management slot.
- Never expose credentials, authorization headers, raw API response bodies, or
  real account data in logs, errors, JSON, cache, UI, tests, fixtures, or website
  imagery. Route user-visible errors through `openrouter.Redact`, or
  `internal/app/redact.go`, which wraps it.
- Tests use synthetic fixtures, never sanitized real exports. They must not
  access the real network, keyring, user configuration, cache, or credentials.
- Preserve the documented TLS and loopback-test exceptions, redirect handling,
  15-second timeout, 4 MiB response cap, bounded retries, and `Retry-After`.
- Environment credentials are not persisted automatically. Fallback credential
  files remain `0600` inside `0700` directories; insecure files are refused
  before reading. Setup validates before storage, and logout removes only
  RouteBurn-owned local entries.
- RouteBurn is the terminal UI and CLI only. The OpenRouter engine — the
  read-only client, snapshot and runway calculation, daily aggregation, cache,
  configuration, and credential resolution — lives in
  `github.com/acevedomiguel/burnblocks/openrouter`. Do not reintroduce a local
  copy of any of it, and do not publish a public Go library from this module.

## Snapshots and application behavior

- Credits are required. Optional analytics and key failures become redacted
  warnings; failed refreshes retain usable prior data and stale responses cannot
  replace newer data.
- Cache snapshots remain versioned, aggregate-only, atomically replaced, and
  explicitly labelled by age. Cached runway must never appear live.
- Keep rendering in `internal/ui` separate from fetching and application state
  transitions. `internal/format` holds RouteBurn's own display formatting; the
  headless calculations live in burnblocks.
- Domain calculations use injected clocks and half-open UTC intervals. Display
  includes the current partial UTC day; runway uses the preceding 30 complete
  UTC days. Preserve negative balances, low-confidence states, and the documented
  no-spend, unavailable, and beyond-horizon outcomes.
- Treat the schema-2 `--json` output as a stable contract. Its projection and
  snapshot types now live in `burnblocks/openrouter`; preserve nullable limits
  and non-numeric runway values, and coordinate intentional schema changes with
  Burn Blocks, cache compatibility, docs, and fixtures.
- Preserve CLI exit codes `0/1/2/3`, non-interactive missing-credential behavior
  in JSON mode, JSON on stdout, and diagnostics on stderr.

## Terminal UI

- Use the Charm v2 dependencies already declared in `go.mod`.
- Preserve exact terminal height, no line overflow, a footer on the last row, and
  no ANSI escapes in no-color mode.
- Keep supported layouts working from 60 columns and 16 rows upward. Preserve
  table focus, scrolling, help behavior, and synthetic `internal/ui/fixture.go`
  data when changing the UI.

## Website

- `site/` is deployed directly without a build. Keep its current framework-free,
  self-hosted architecture and progressive-enhancement behavior.
- Preserve each page's CSP, including no `unsafe-inline`, no `unsafe-eval`, and
  `connect-src 'none'`.
- Homepage assets are relative; `404.html` uses absolute `/routeburn/` paths so
  nested missing URLs still load assets.
- Keep canonical URLs, sitemap, robots metadata, accessibility, responsive
  behavior, and explicit image dimensions synchronized.
- Generate imagery only from synthetic sources under `assets-src/`. Generated
  outputs under `site/assets/images/` are committed release assets, not files to
  hand-edit. CI does not regenerate them.

## Release automation

- Releases are driven by squash-merged Conventional Commits on `main`; the PR
  title is the release commit subject. Do not manually select versions, move
  tags, edit generated release sections of `CHANGELOG.md`, or hand-publish normal
  artifacts.
- Keep the unsuffixed Go module on `v1.x.y`. `scripts/check-module-version.mjs`
  runs in semantic-release `verifyRelease` and must reject any major that differs
  from the module path. A breaking change requires a coordinated `/v2` migration
  with consumers before any `v2` tag.
- `.goreleaser.yaml` intentionally publishes a Homebrew **cask** to
  `acevedomiguel/homebrew-routeburn`, not a formula. Keep generation under
  `homebrew_casks` and do not hand-edit the tap from this repository.
- Preserve `CGO_ENABLED=0`, the declared target exclusions, version injection,
  SHA-256 checksums, and empty-token snapshot behavior.
- Preserve declarative `postflight_steps`, not deprecated `postflight`. The
  current quarantine removal exists because binaries are ad-hoc signed; it is a
  documented Gatekeeper trade-off, with signing/notarization preferred.
- `scripts/smoke-test.sh` is an opt-in live OpenRouter API check requiring a real
  management credential. Do not run it for ordinary verification or without an
  explicit request.
- `release-repair.yml` is only for explicitly confirmed asset repair at an
  existing immutable tag. Code fixes require a corrective release.

## RouteBurn verification

Run checks relevant to the changed area and report anything unavailable:

- Go changes: `make check`; before a PR also run `gofmt -l .` (no output),
  `go test -race ./...`, and `go build ./cmd/routeburn`.
- CLI/build changes: `make build` and `./routeburn --version` without loading
  real credentials.
- GoReleaser changes: `goreleaser check` and `make snapshot`, followed by
  `git diff --exit-code` because its hooks run `go mod tidy`.
- Site markup/assets: `npm run check:html` and `npm run check:links`.
- Site behavior, styling, or accessibility: also `npm run test:site` and
  `npm run check:lighthouse`; CI expects accessibility, best-practices, and SEO
  scores of 100.
- Release tooling: `npm run check:release-notes` and `npm run check:module-version`.
- Use `node tests/site/serve.mjs 8321` when a preview must emulate the production
  `/routeburn/` base path and 404 behavior.
