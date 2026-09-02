# task-288 — findings

## Diagnosis

`services/atlas-tenants` declared its status topic as an untyped constant
holding a Kafka topic *name*, `EventTopicTenantStatus = "tenant.status"`. It
reached `Buffer.Put(t topic.Token, ...)` by Go's implicit untyped-constant
conversion, so the type checker saw a well-typed call, and `topic.EnvProvider`
then looked up an environment variable literally named `tenant.status`. No
overlay declares such a variable. Before PR #1608 (task-276, merged
2026-09-01) `EnvProvider` fell back silently to the token string, so the value
resolved to the topic name `tenant.status` and worked; #1608 correctly made an
unset token an error, and every tenant emit began failing.

Observed live in `atlas-pr-1566`: `create_tenant` returned HTTP 500 with
`topic token [tenant.status] has no value in the environment` (11:04:51,
11:05:03, 11:05:24, 11:06:05 UTC); the `atlas-pr-bootstrap` Job failed all four
attempts at step `tenant-create`; with no tenant, `atlas-configurations` never
published a projection snapshot, so `atlas-login` fatalled with
`configuration: projection snapshot not yet published` and `atlas-drops` got a
500 from `GET /api/configurations/services/...` and exited 1 — both
CrashLoopBackOff, Argo `Degraded`.

Because the constant was never typed `topic.Token`, `libs/atlas-kafka/gen`'s
scanner (which matches on that exact named type, `gen/scan.go:20-24`) never saw
it, so `EVENT_TOPIC_TENANT_STATUS` was absent from `topics.yaml` and every
generated deploy surface, and the topic was never pre-created anywhere.

## The guard gap

`tools/topicguard`'s `bare-token-literal` diagnostic already inspected untyped
constants reaching a `topic.Token` parameter, but only reported when the
constant's *value* matched `rawEnvTopicPattern` =
`^[A-Z0-9_]*TOPIC[A-Z0-9_]*$` (`analyzer.go:70,194`). That inverted the intent:
a legacy lowercase topic name — exactly the shape still needing migration —
could never match. Confirmed against unmodified `main`:
`GUARD_SKIP_SELFTEST=1 GUARD_SERVICE_MODULES="services/atlas-tenants/atlas.com/tenants" ./tools/go-analyzer-guards.sh`
→ `go-analyzer-guards: PASS`.

## Experiment: how wide can the filter be removed?

PRD open question 1 asked whether widening would surface a manageable or
unmanageable number of fleet-wide violations. Measured rather than designed:

| Variant | Change | Fleet-wide result |
|---|---|---|
| A | Drop the shape filter on the untyped-constant path (`reportIfUntypedConstRef`) only | 0 new violations (69 services/ + 24 libs/ modules) |
| B | Drop it on the bare-string-literal path too | `go-analyzer-guards: PASS` — 0 violations (69 + 24 modules) |

Variant B — the stronger one — is clean, so it was taken. No allowlist entries
were added; the `allowlist.txt` remains scoped to diagnostic 2. PRD open
questions 1 and 3 are closed by this result.

## Decisions

- Token is `EVENT_TOPIC_TENANT_STATUS`, matching the fleet convention.
- `cleanup: delete`, not `compact`. The `compact` list in
  `libs/atlas-kafka/gen/policies.yaml` is justified by consumers that replay
  from first offset at boot to rebuild config state; the tenant status topic has
  no consumer at all (repo-wide grep for `"tenant.status"` returns only the
  declaration).
- The wire topic name changes from `tenant.status` to
  `EVENT_TOPIC_TENANT_STATUS-<env>`. Safe for the same reason — nothing reads
  it. The orphaned legacy topic in long-lived clusters is left to age out
  (PRD open question 2).
- `deploy/compose/` needed no hand edit: `atlas-tenants` inherits
  `env_file: .env` through the `x-atlas-infra` anchor, and `.env.example` is
  generated.
- `testmain_test.go` previously set the token to its own value, which would let
  this whole class of defect pass the suite. It now sets a resolved name
  distinct from the token.

## Next step

Post-merge only. The ephemeral overlays pin `atlas-tenants` to `:latest`
(not a per-PR tag), so once this lands on `main` and `latest` rebuilds, each
environment re-resolves its `bot/pr-<n>-resolved` branch and picks up the
regenerated configmap. The exhausted `atlas-pr-bootstrap` Job must then be
deleted per namespace — a `Failed` Job past its backoff limit does not retry
on its own.
