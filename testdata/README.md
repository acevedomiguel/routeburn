# Synthetic fixtures

This directory holds synthetic, sanitized API payloads used by tests and by
`ROUTEBURN_DEMO=1 routeburn` for screenshots.

Rules:

- Never commit a real OpenRouter response. Real payloads carry account-wide
  spend, key labels, and usage history that identify the account owner.
- Fixtures are hand-written or generated from the shapes documented in
  `IMPLEMENTATION_PLAN.md` section 6, then reviewed for identifying values.
- Dates, model names, spend figures, and key labels are arbitrary. Any
  resemblance to a live account is coincidental.
- No credentials, no `Authorization` headers, no token-like strings. The
  credential-leak CI job scans for these.
- Keep the file count and size small; fixtures exist to pin the decoding
  contract, not to reproduce volume.

Current fixtures:

| File | Covers |
| --- | --- |
| `internal/domain` table tests | Time ranges, aggregation, runway branches |
| `internal/openrouter/*_test.go` | Endpoint decoding, null limit fields, analytics envelopes |
| `internal/app/fixture_test.go` | Snapshot builder composition |
| `internal/ui/fixture_test.go` | Deterministic rendering fixtures |

If a fixture must change because the upstream contract changed, update the test
that pins it and note the contract change in the commit message. Do not loosen an
assertion to make a changed payload fit.
