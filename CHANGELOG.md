## [1.6.0](https://github.com/acevedomiguel/routeburn/compare/v1.5.1...v1.6.0) (2026-11-25)

### Features

* expose RouteBurn as an importable Go library ([#27](https://github.com/acevedomiguel/routeburn/issues/27)) ([04cfc5a](https://github.com/acevedomiguel/routeburn/commit/04cfc5aac6691f50ba35d6e6f9ab5d54e8312bbd))

## [1.5.1](https://github.com/acevedomiguel/routeburn/compare/v1.5.0...v1.5.1) (2026-11-24)

### Bug Fixes

* authenticate homebrew tap push with tap token ([#26](https://github.com/acevedomiguel/routeburn/issues/26)) ([e7c8db6](https://github.com/acevedomiguel/routeburn/commit/e7c8db6bef7790722b3fa4da438c6c974522ff16))

## [1.5.0](https://github.com/acevedomiguel/routeburn/compare/v1.4.1...v1.5.0) (2026-11-24)

### Features

* add a title row and tab title, refresh the readme and site ([#25](https://github.com/acevedomiguel/routeburn/issues/25)) ([b5c4ea2](https://github.com/acevedomiguel/routeburn/commit/b5c4ea2cfbbdd87d0f5b2481dcc55fd517a05568))

## [1.4.1](https://github.com/acevedomiguel/routeburn/compare/v1.4.0...v1.4.1) (2026-11-23)

### Bug Fixes

* wait for homebrew tap publication ([263a6de](https://github.com/acevedomiguel/routeburn/commit/263a6debfc3bcb0b484e909d020273c17651ee17))

## [1.4.0](https://github.com/acevedomiguel/routeburn/compare/v1.3.0...v1.4.0) (2026-11-23)

### Features

* add about section and help popup ([15bd8d1](https://github.com/acevedomiguel/routeburn/commit/15bd8d1be67f31b27e67387d0615f65abdc4045b))

## [1.3.0](https://github.com/acevedomiguel/routeburn/compare/v1.2.0...v1.3.0) (2026-11-22)

### Features

* heat-colour the daily table and mark the tab focus ([d05f15b](https://github.com/acevedomiguel/routeburn/commit/d05f15b0e78013626ed29f38a2604355cd9dccd1))
* replace the daily sparkline with a scrollable daily table ([95b5e11](https://github.com/acevedomiguel/routeburn/commit/95b5e1160687a3656ebc37df6e42676368bfb2f9))
* stack the small panels in a sidebar beside the daily table ([349bf9d](https://github.com/acevedomiguel/routeburn/commit/349bf9db56ea09142b9193653a5fca1f02b7a744))

### Bug Fixes

* keep coloured spend values right-aligned and finish the v2 import move ([3b0b47b](https://github.com/acevedomiguel/routeburn/commit/3b0b47b7fd4e2a0ba49b4a891a5414494ed386a8))

## [1.2.0](https://github.com/acevedomiguel/routeburn/compare/v1.1.1...v1.2.0) (2026-11-22)

### Features

* add static project website with GitHub Pages deployment ([#16](https://github.com/acevedomiguel/routeburn/issues/16)) ([428d541](https://github.com/acevedomiguel/routeburn/commit/428d54180ba8907df10c112efb267b9e413a2885))

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
