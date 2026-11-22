## [1.1.1](https://github.com/acevedomiguel/routeburn/compare/v1.1.0...v1.1.1) (2026-11-22)

### Bug Fixes

* generate the cask with postflight_steps ([#14](https://github.com/acevedomiguel/routeburn/issues/14)) ([d003f76](https://github.com/acevedomiguel/routeburn/commit/d003f76d2b510f579d64644f585c8ebd2bcb9de4))

## [1.1.0](https://github.com/acevedomiguel/routeburn/compare/v1.0.1...v1.1.0) (2026-11-22)

### Features

* put Cost by model and Tokens on the same row ([#13](https://github.com/acevedomiguel/routeburn/issues/13)) ([26910bc](https://github.com/acevedomiguel/routeburn/commit/26910bc58f841d250478d8e52e01e189fc3032d8)), closes [#12](https://github.com/acevedomiguel/routeburn/issues/12)

## [1.0.1](https://github.com/acevedomiguel/routeburn/compare/v1.0.0...v1.0.1) (2026-11-22)

### Bug Fixes

* **release:** clear the macOS quarantine flag in the generated cask ([#9](https://github.com/acevedomiguel/routeburn/issues/9)) ([74dd9f8](https://github.com/acevedomiguel/routeburn/commit/74dd9f8c06ae96d2245cf4b371ed4dcbc6380b1e))

## 1.0.0 (2026-11-22)

### Features

* ship the RouteBurn read-only OpenRouter dashboard ([#1](https://github.com/acevedomiguel/routeburn/issues/1)) ([83050a5](https://github.com/acevedomiguel/routeburn/commit/83050a52f1d09fbfe6635e0432609651bec5a372))

### Bug Fixes

* **release:** render changelog notes with a writer the preset accepts ([#8](https://github.com/acevedomiguel/routeburn/issues/8)) ([b0c8395](https://github.com/acevedomiguel/routeburn/commit/b0c8395d67fe2837aa175828d240a908cfcd33ed))

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
