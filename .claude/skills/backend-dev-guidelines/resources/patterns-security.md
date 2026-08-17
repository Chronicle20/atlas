---
title: Security Patterns
description: Token, redirect, and secret-handling rules for auth-related Atlas services.
---

# Security Patterns

Applies to services that handle authentication, authorization, token issuance
or revocation, OAuth-style callbacks, or secrets — `atlas-login` and
`atlas-account` are the usual subjects. Rule IDs are defined in
[audit-checklist.md](audit-checklist.md).

Services with none of those responsibilities dispose of the whole `SEC-*`
family as `N/A`, naming the absent trigger as the evidence.

## Verified token parsing (SEC-01)

A token is only trustworthy after its signature and claims have been verified
against the configured key. `ParseUnverified` decodes without checking
anything; it is acceptable only for inspecting a token you are about to
discard (e.g. logging an issuer for diagnostics), never for a decision.

**How to verify.** Grep the service for `ParseUnverified` and for every
`Parse(` / `ParseWithClaims(` call site. Follow each result to the decision it
feeds.

**Pass criteria.** Every authorization or identity decision reads claims from a
token parsed with a key and validated claims. Any `ParseUnverified` result
reaching a trust decision is a FAIL.

## Revocation reads validated tokens (SEC-02)

**How to verify.** Read the logout and revocation handlers.

**Pass criteria.** Claims used to identify *which* session or token to revoke
come from a validated token. Extracting a subject from an unvalidated token
lets an attacker revoke someone else's session — a FAIL.

## No open redirect (SEC-03)

**How to verify.** Read the callback and redirect handlers; find where the
redirect target originates.

**Pass criteria.** Redirect targets are validated against an allowlist or
constrained to a known host/path prefix. Reflecting a caller-supplied absolute
URL into a `Location` header is a FAIL.

## Secrets are not hardcoded (SEC-04)

**How to verify.** Grep the changed source for embedded keys, passwords, tokens,
and connection strings.

**Pass criteria.** Secrets come from the environment or a mounted secret (the
`db-credentials` secret is the established pattern for DB creds — see
`deploy/k8s/base/`), never from a Go literal or a checked-in config file. Test
fixtures with obviously fake values are fine; cite the file:line and say why.
