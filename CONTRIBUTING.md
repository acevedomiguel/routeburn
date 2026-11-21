# Contributing to RouteBurn

## Development setup

- Go 1.24 or newer (`go.mod` declares the minimum).
- Node.js as pinned in `.nvmrc`, only for release tooling.

```bash
go build ./cmd/routeburn
go test ./...
```

Useful Makefile targets: `make build`, `make test`, `make fmt`, `make vet`,
`make check`, `make snapshot`.

## Commit messages

RouteBurn releases are derived from Conventional Commits, so the message format
is load-bearing rather than cosmetic.

- `fix: ...` produces a patch release.
- `feat: ...` produces a minor release.
- `feat!: ...` or a `BREAKING CHANGE:` footer produces a major release.
- `perf:` produces a patch release.
- `docs:`, `chore:`, `ci:`, `test:`, and `style:` produce no release.

Subjects are lowercase, imperative, and under 72 characters, for example
`fix: preserve cached data after refresh failure`.

Validate a message locally:

```bash
printf 'fix: preserve cached data\n' | npx commitlint
```

Pull requests are squash-merged, so the PR title becomes the release commit
subject and is validated by the `commit-title` workflow.

## Testing expectations

- Test-first: write the failing test, watch it fail, then make it pass.
- Domain calculations are pure functions with injected clocks so tests are
  deterministic.
- No network calls in tests; use `httptest` and synthetic fixtures.
- Fixtures never contain real account data or credentials.

## Credentials

Never commit tokens. Local development reads credentials from
`ROUTEBURN_MANAGEMENT_KEY` or the OS keyring. See `SECURITY.md`.

## Release process

Merging a qualifying commit into `main` is the only release action; automation
computes the version, tags it, publishes binaries, and updates the Homebrew tap.

## License

Contributions are accepted under the MIT license in `LICENSE`.
