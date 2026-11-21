# RouteBurn

**A read-only terminal dashboard for OpenRouter credits, spend, and runway.**

```
Credits remaining                                Last 30 days
$42.10 / $100.00                                 Spend: $12.34
[######################----------] 42% remaining Tokens: 8.4M
Estimated runway                                 This API key
102 days - ~Mar 03                               $31.20 / $50.00 limit

Daily credit spend
$1.20 +
      |        #
      |     #  #  #     #
$0.00 +---#--#--#--#--#--#---
      Oct 24                 Nov 21

Cost by model
anthropic/claude-sonnet-4    $7.21  4.1M tokens
openai/gpt-5                 $3.82  2.9M tokens
meta/llama-4-70b             $1.31  1.4M tokens

[r]refresh  [j/k]scroll  [?]help  [q]quit
```

RouteBurn answers three questions without opening a browser: how much credit is
left, what was spent in the last 30 days, and how long that balance lasts at the
current burn rate.

> RouteBurn only reads data. It sends no inference requests, makes no purchases,
> and never rotates or revokes keys.

## Features

- Account credits: purchased, used, and remaining, with negative balances shown
  rather than hidden.
- Per-key credit limit when a regular API key is configured, clearly labelled as
  a different scope from the account balance.
- 30 daily UTC buckets rendered as an ASCII chart that fits the terminal.
- Model spend ranking with token totals and an `Other (remaining)` rollup.
- Runway estimate from the previous 30 complete UTC days of spend.
- Cached snapshot on startup, explicitly labelled with its age when stale.
- Automatic refresh every 10 seconds by default, plus a throttled manual refresh.
- Renders from 60 columns up, and says so plainly when the terminal is smaller.
- Versioned `--json` snapshot for scripts and dashboards.

## Requirements

- Go 1.24 or newer to build from source.
- An OpenRouter **Management API key** for account credits and analytics.
- Optionally a regular **API key** for the per-key limit card.

### Management key versus regular API key

The two keys are not interchangeable, and RouteBurn treats them separately.

| Key | Scope | Used for |
| --- | --- | --- |
| Management key | Whole account | `GET /credits`, `POST /analytics/query` |
| Regular API key | One key | `GET /key` (label, limit, remaining limit) |

A regular key alone cannot show the account balance. A management key alone
still produces every card except the per-key limit. `OPENROUTER_API_KEY` is
accepted as a compatibility fallback for the *regular* key slot only; it is
never treated as a management key.

Create a management key at
<https://openrouter.ai/settings/management_keys>.

## Installation

### Homebrew (macOS and Linux)

```bash
brew tap acevedomiguel/routeburn
brew install routeburn
routeburn --version
```

Upgrades arrive through the same tap:

```bash
brew update
brew upgrade routeburn
```

### Go

```bash
go install github.com/acevedomiguel/routeburn/cmd/routeburn@latest
```

### Prebuilt binaries

Every release publishes Linux, macOS, and Windows archives plus a `checksums.txt`
file on the GitHub Releases page. Verify a download with:

```bash
sha256sum -c checksums.txt --ignore-missing
```

### From source

```bash
git clone https://github.com/acevedomiguel/routeburn.git
cd routeburn
make build
./routeburn --version
```

## Quick start

```bash
routeburn setup     # masked prompt, validates the key, stores it, writes config
routeburn           # open the dashboard
```

`setup` never echoes the key, and stores nothing until OpenRouter accepts it. If
you prefer environment variables:

```bash
export ROUTEBURN_MANAGEMENT_KEY="..."
export ROUTEBURN_API_KEY="..."   # optional
routeburn
```

Environment credentials are used as-is and are never copied into persistent
storage automatically.

## Dashboard fields

| Field | Meaning |
| --- | --- |
| Credits remaining | `total_credits - total_usage`. Negative values are shown, never clamped. |
| Last 30 days | Sum of daily credit spend and tokens over the current UTC day plus the 29 before it. |
| Estimated runway | Days until the balance reaches zero at the recent burn rate. |
| This API key | Per-key credit limit and remaining limit, if a regular key is configured. |
| Daily credit spend | One bucket per UTC day for the 30-day window. |
| Cost by model | Model spend for the same window, highest first, top four plus `Other (remaining)`. |

Token counts are not credit balances. The number of tokens a credit buys depends
on the model mix.

## Runway formula

```text
daily_burn    = spend over the previous 30 complete UTC days / 30
days_remaining = remaining credits / daily_burn
depletion_date = start of today + ceil(days_remaining) days
```

The window deliberately excludes the current partial day, so a dashboard opened
at 02:00 cannot mistake two hours of spend for a whole day. Every branch is
labelled instead of guessed:

| Situation | Displayed |
| --- | --- |
| Balance at or below zero | `0 days`, with a top-up warning when negative |
| No spend in the window | `No recent spend` |
| Analytics unavailable | `Unavailable` |
| Estimate beyond 100 years | `Beyond horizon` (flagged low confidence) |
| Fewer than seven active days | Numeric estimate, flagged low confidence |

Runway is an estimate, not a guarantee. A sudden change in usage invalidates it
immediately. Account runway is also not the same thing as a per-key limit reset.

## Configuration

Locations follow `XDG_CONFIG_HOME` and `XDG_CACHE_HOME` when set (including on
macOS), otherwise the platform defaults:

| Platform | Config | Cache |
| --- | --- | --- |
| Linux | `~/.config/routeburn/config.yaml` | `~/.cache/routeburn/snapshot.json` |
| macOS | `~/Library/Application Support/routeburn/config.yaml` | `~/Library/Caches/routeburn/snapshot.json` |
| Windows | `%AppData%\routeburn\config.yaml` | `%LocalAppData%\routeburn\snapshot.json` |

Print the exact path with `routeburn config path`. A complete configuration file:

```yaml
version: 1
refresh_interval: 10s
range_days: 30
theme: auto
color: auto
keyring: true
```

| Setting | Values | Notes |
| --- | --- | --- |
| `refresh_interval` | `10s` or longer, or `0` | Shorter positive values are rejected. `0` disables automatic refresh only. |
| `range_days` | `30` | Fixed in the MVP. |
| `theme` | `auto`, `dark`, `light` | |
| `color` | `auto`, `always`, `never` | `--no-color` wins over this. |
| `keyring` | `true`, `false` | When `false`, the `0600` fallback file is used. |

Precedence, highest first: CLI flags, environment variables, YAML file,
built-in defaults. Environment variables: `ROUTEBURN_REFRESH_INTERVAL`,
`ROUTEBURN_THEME`, `ROUTEBURN_COLOR`, `ROUTEBURN_KEYRING`,
`ROUTEBURN_MANAGEMENT_KEY`, `ROUTEBURN_API_KEY`, `OPENROUTER_API_KEY`.

`routeburn config reset` rewrites the sample configuration.

## Credential storage

Resolution order for the management key:

1. `ROUTEBURN_MANAGEMENT_KEY`.
2. OS keyring (service `routeburn`, account `management`).
3. Fallback file at `0600` inside a `0700` directory.
4. Interactive `routeburn setup`.

The regular API key follows the same chain, with `OPENROUTER_API_KEY` accepted
between the two environment variables.

A fallback file that is group- or world-readable is refused rather than read,
and the error never prints the secret. `routeburn logout` deletes only
RouteBurn's local keyring entries and fallback files; it does not revoke anything
at OpenRouter, so do that in the OpenRouter console.

## JSON mode

```bash
routeburn --json
```

Prints one snapshot to stdout and exits. Progress, warnings, and errors go to
stderr, so `routeburn --json | jq` is safe. Missing credentials exit `3` with
setup instructions rather than prompting.

```json
{
  "schema_version": 1,
  "generated_at": "2026-11-21T15:42:00Z",
  "cached": false,
  "account": {
    "total_credits": 100,
    "total_usage": 57.9,
    "remaining": 42.1
  },
  "last_30_days": { "credit_spend": 12.34, "tokens": 8420000 },
  "runway": {
    "window_days": 30,
    "daily_burn": 0.4113,
    "days_remaining": 102,
    "depletion_date": "2027-03-03",
    "basis": "finite",
    "low_confidence": false
  },
  "key": {
    "label": "production",
    "limit": 50,
    "limit_remaining": 31.2,
    "limit_reset": "monthly",
    "usage_monthly": 18.8
  },
  "warnings": []
}
```

`basis` is one of `finite`, `depleted`, `no_recent_spend`, `unavailable`, or
`beyond_horizon`. `key` is `null` when no regular API key is configured, and
`days_remaining`/`depletion_date` are `null` whenever the estimate is not
numeric.

## Offline and cache behaviour

Every successful refresh writes a versioned, atomically replaced snapshot. On the
next launch that snapshot renders immediately and is labelled
`Cached data - updated 2h 14m ago` until a live refresh succeeds. A failed refresh
keeps the previous data on screen with a one-line banner instead of blanking the
dashboard. `--no-cache` skips both reads and writes.

Cached data can be stale. A cached runway estimate is never presented as live.

## Keyboard shortcuts

| Key | Action |
| --- | --- |
| `r` | Refresh now (throttled to one batch per 10 seconds) |
| `j` / `Down` | Scroll down on narrow terminals |
| `k` / `Up` | Scroll up |
| `PgDn` / `PgUp` | Scroll one screen |
| `?` | Toggle help |
| `Esc` | Close help |
| `q` / `Ctrl+C` | Quit |

When OpenRouter asks RouteBurn to slow down, both automatic and manual refreshes
wait until the requested time has passed and the footer explains why.

## Privacy and security

- No account data leaves the machine: RouteBurn talks only to
  `https://openrouter.ai/api/v1` and writes only to its own config and cache.
- Authorization headers, tokens, and response bodies are never logged, rendered,
  or cached. Error messages are redacted before display.
- TLS is required; a plaintext base URL is rejected unless it points at loopback
  (used by tests).
- The cache stores aggregates only, never credentials or raw responses.

See `SECURITY.md` to report a vulnerability.

## Development

```bash
make check      # gofmt + go vet + go test
make test       # unit tests only
make build      # binary with version metadata
make snapshot   # GoReleaser snapshot build (requires goreleaser)
```

Tests never touch the network, the real keyring, or your real config directory.
Release notes are generated from Conventional Commits; see `CONTRIBUTING.md`.
`ROUTEBURN_DEMO=1 routeburn` renders the dashboard with synthetic fixtures, which
is how the screenshot above is produced without spending credits.

## Contributing and license

Contributions are welcome under the MIT license in `LICENSE`. Start with
`CONTRIBUTING.md`, and read `CODE_OF_CONDUCT.md` before opening a pull request.
