# Security Policy

## Reporting a vulnerability

Do not open a public issue for a security problem. Use GitHub's private
vulnerability reporting on this repository (Security tab, "Report a
vulnerability"). If that is unavailable, contact the maintainer listed in
`go.mod` / `LICENSE`.

Include:

- Affected version (`routeburn --version`) and platform.
- Steps to reproduce, and whether the issue is remotely triggerable.
- Any proof of concept, with real tokens replaced by placeholders.

Expected timeline: acknowledgement within five business days, an assessment
within ten, and a coordinated disclosure date once a fix is ready. Credit is
given in the release notes unless you ask to stay anonymous.

## Scope

In scope:

- Credential handling: leakage into config, cache, JSON output, logs, error
  messages, or crash output.
- Unsafe fallback-file permissions or acceptance of group/world-readable files.
- Redaction failures in the shared redaction helper.
- Transport problems: accepting non-TLS base URLs, following untrusted
  redirects, unbounded response reads.
- Release supply chain: workflow permissions, action pinning, artifact or
  checksum substitution.

Out of scope:

- Vulnerabilities in OpenRouter's own API or console.
- Issues that require an attacker who already controls your user account or
  shell, or already holds your API key.
- Rate-limit or billing disputes handled by OpenRouter support.
- Missing hardening that has no demonstrated impact (for example absent
  certificate pinning).

## Security design

RouteBurn is deliberately narrow.

- **Read-only.** It calls `GET /credits`, `GET /key`, `GET /analytics/meta`, and
  `POST /analytics/query`. It never purchases credits, creates or rotates keys,
  or sends inference requests.
- **One host.** Requests go only to `https://openrouter.ai/api/v1`. A base URL
  override is rejected unless the host is TLS-protected or resolves to loopback,
  which is how tests avoid the network.
- **No secrets at rest unless you ask.** The OS keyring is the default store. The
  fallback token file is created `0600` inside a `0700` directory and is refused
  if its permissions are loosened. Secrets are never written to the config file
  or the cache.
- **No secrets in output.** Only the `Authorization` header carries a key, that
  header is never logged, error bodies are replaced with their typed meaning,
  and every user-visible error passes through the shared redaction helper. JSON
  output contains aggregates only.
- **Bounded I/O.** 15-second request timeouts, a 4 MiB response cap, at most two
  retries on 429/502/503/504 or transport errors, and `Retry-After` respected.
- **Explicit offline behaviour.** A cache hit is always labelled with its age; a
  cached runway estimate is never presented as live.

`routeburn logout` deletes only RouteBurn's local keyring entries and fallback
files. It does not revoke keys at OpenRouter, so revoke there if a key may have
been exposed.

## Hardening checklist for operators

- Prefer `ROUTEBURN_MANAGEMENT_KEY` from a secret manager over stored files.
- If you use the fallback file, keep the parent directory at `0700`.
- Treat a leaked management key as an account-level incident: rotate it in the
  OpenRouter console. It can read all account usage.
- Enable two-factor authentication on your OpenRouter account.

## Supported versions

Security fixes land on the latest released minor version. Older versions may be
asked to upgrade before a fix is backported.
