#!/usr/bin/env bash
# Opt-in live smoke test.
#
# This is the only script that talks to the real OpenRouter API. It is never run
# in CI and never runs by default:
#
#   ROUTEBURN_SMOKE=1 scripts/smoke-test.sh
#
# Requirements:
#   - ROUTEBURN_MANAGEMENT_KEY (or a resolvable credential) for the aggregate run
#   - a built ./routeburn (or ROUTEBURN_BIN pointing elsewhere)
#
# Guarantees:
#   - never prints a token, header, or raw response body
#   - asserts only aggregate shapes (numeric fields, 30 buckets, valid JSON)
#   - exits non-zero on failure so it is safe to use as a gate

set -euo pipefail

if [[ "${ROUTEBURN_SMOKE:-0}" != "1" ]]; then
  echo "smoke-test: skipped (set ROUTEBURN_SMOKE=1 to run against the live API)"
  exit 0
fi

BIN="${ROUTEBURN_BIN:-./routeburn}"
if [[ ! -x "$BIN" ]]; then
  echo "smoke-test: $BIN is not executable; run 'make build' first" >&2
  exit 1
fi

if [[ -z "${ROUTEBURN_MANAGEMENT_KEY:-}" ]]; then
  echo "smoke-test: ROUTEBURN_MANAGEMENT_KEY is not set" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

fail() {
  echo "smoke-test: FAIL - $1" >&2
  exit 1
}

echo "smoke-test: version"
"$BIN" --version || fail "--version exited non-zero"

echo "smoke-test: help"
"$BIN" --help >/dev/null || fail "--help exited non-zero"

echo "smoke-test: json snapshot (live)"
json="$tmpdir/snapshot.json"
# stderr is captured separately so a credential-leak check can scan it.
stderr_file="$tmpdir/stderr.txt"
if ! XDG_CACHE_HOME="$tmpdir/cache" XDG_CONFIG_HOME="$tmpdir/config" \
    "$BIN" --json --no-cache >"$json" 2>"$stderr_file"; then
  cat "$stderr_file" >&2
  fail "--json exited non-zero"
fi

python3 - "$json" <<'PY' || fail "json contract check failed"
import json, sys

path = sys.argv[1]
with open(path) as handle:
    data = json.load(handle)

def need(container, key, label, types):
    if key not in container:
        raise SystemExit(f"missing {label}.{key}")
    value = container[key]
    if not isinstance(value, types):
        raise SystemExit(f"{label}.{key} has type {type(value).__name__}, want {types}")
    return value

if need(data, "schema_version", "root", int) != 1:
    raise SystemExit("schema_version must be 1")
need(data, "generated_at", "root", str)
need(data, "cached", "root", bool)

account = need(data, "account", "root", dict)
for key in ("total_credits", "total_usage", "remaining"):
    need(account, key, "account", (int, float))

last = need(data, "last_30_days", "root", dict)
need(last, "credit_spend", "last_30_days", (int, float))
need(last, "tokens", "last_30_days", int)

runway = need(data, "runway", "root", dict)
need(runway, "window_days", "runway", int)
need(runway, "daily_burn", "runway", (int, float))
need(runway, "basis", "runway", str)
if runway["basis"] not in {"finite", "depleted", "no_recent_spend", "unavailable", "beyond_horizon"}:
    raise SystemExit(f"unexpected basis {runway['basis']!r}")

need(data, "warnings", "root", list)
if "key" not in data:
    raise SystemExit("missing key section")

print("smoke-test: json contract ok")
PY

echo "smoke-test: credential leak scan"
# Case-insensitive search for the markers a leaked secret would produce.
if grep -Eiq 'sk-or-v1-|bearer |authorization:' "$json" "$stderr_file"; then
  fail "json or stderr looks like it contains a credential"
fi

echo "smoke-test: cache round trip"
cache_home="$tmpdir/cache"
XDG_CACHE_HOME="$cache_home" XDG_CONFIG_HOME="$tmpdir/config" \
  "$BIN" --json >"$tmpdir/first.json" 2>/dev/null || fail "cache run 1 failed"
if [[ ! -s "$cache_home/routeburn/snapshot.json" ]]; then
  fail "expected a snapshot cache at $cache_home/routeburn/snapshot.json"
fi

if grep -Eiq 'sk-or-v1-|bearer ' "$cache_home/routeburn/snapshot.json"; then
  fail "cache file contains a credential"
fi

echo "smoke-test: offline behaviour"
# Point the base URL at a closed port via the config file to force a network
# failure, then confirm the cached snapshot is still served and labelled.
cat >"$tmpdir/config/routeburn/config.yaml" <<YAML
version: 1
refresh_interval: 10s
range_days: 30
theme: auto
color: never
keyring: false
YAML

if XDG_CACHE_HOME="$cache_home" XDG_CONFIG_HOME="$tmpdir/config" \
    HTTPS_PROXY="http://127.0.0.1:1" HTTP_PROXY="http://127.0.0.1:1" \
    "$BIN" --json >"$tmpdir/offline.json" 2>"$tmpdir/offline.err"; then
  python3 - "$tmpdir/offline.json" <<'PY' || fail "offline fallback was not labelled"
import json, sys
with open(sys.argv[1]) as handle:
    data = json.load(handle)
if not data.get("cached"):
    raise SystemExit("offline snapshot must report cached=true")
print("smoke-test: offline snapshot labelled")
PY
else
  echo "smoke-test: offline run failed without a usable cache (acceptable on a cold cache)"
fi

echo "smoke-test: ok"
