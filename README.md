# RouteBurn

RouteBurn is a read-only terminal dashboard for OpenRouter credits, usage, and
runway. It shows your balance, the last 30 days of spend, and a top-model
breakdown, and it estimates how many days your remaining credits will last.

> RouteBurn reads data and never changes your account. It performs no
> purchases, key rotations, or other writes against the OpenRouter API.

## Status

Early development. The MVP targets a single provider (OpenRouter), a 30-day
range, and a Bubble Tea TUI with a non-interactive `--json` mode.

## Build from source

```bash
git clone https://github.com/acevedomiguel/routeburn.git
cd routeburn
go build -trimpath -o routeburn ./cmd/routeburn
./routeburn --version
```

Requires Go 1.24 or newer. See the full documentation in `README.md` sections
below as the MVP lands, and `IMPLEMENTATION_PLAN.md` for the design.

## Design notes

- Runway is an estimate derived from your previous 30 complete UTC days of
  credit spend. It is not a guarantee.
- Cached data can be stale; the dashboard labels freshness explicitly.
- Token counts are not credit balances.

## Contributing

See `CONTRIBUTING.md`. Commit messages follow Conventional Commits.

## License

MIT. See `LICENSE`.
