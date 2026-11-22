# Changelog

All notable changes to this project are documented in this file.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions: [Semantic Versioning](https://semver.org/spec/v2.0.0.html), computed by
semantic-release from Conventional Commits.

This file is rewritten by `@semantic-release/changelog` on every release. Do not
edit released sections by hand; the next release would overwrite them.

## [Unreleased]

### Added

- Read-only OpenRouter dashboard for account credits, per-key limit, 30-day spend,
  model breakdown, and runway.
- Guided first-run setup with masked input, live credential validation, and OS
  keyring storage with a `0600` fallback file.
- Versioned `--json` snapshot for scripts.
- Atomic snapshot cache with explicit cached/stale labelling.
- Automatic refresh at 10-second-or-longer intervals, throttled manual refresh,
  and shared `Retry-After`/rate-limit backoff.
- Automated release: semantic-release derives the version, GoReleaser publishes
  archives and checksums, and the Homebrew tap updates itself.
