# Tenant Status Topic Token — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-02
---

## 1. Overview

`services/atlas-tenants` emits tenant created/updated/deleted events through a
`topic.Token`-keyed message buffer, but declares its token as an **untyped**
string constant holding a literal topic name:

```go
// services/atlas-tenants/atlas.com/tenants/tenant/kafka.go:12
EventTopicTenantStatus = "tenant.status"
```

`topic.Token` is documented (`libs/atlas-kafka/topic/token.go`) as "the NAME OF
THE ENVIRONMENT VARIABLE that carries a topic's per-environment name -- never
the topic name itself." Because the constant is untyped, Go implicitly converts
it at the `Buffer.Put(t topic.Token, ...)` call site
(`services/atlas-tenants/atlas.com/tenants/kafka/message/message.go:25`), so the
type checker sees a well-typed call. The value then reaches
`topic.EnvProvider`, which looks up an environment variable literally named
`tenant.status`.

Task-276 / PR #1608 ("generated topic manifest and repo-wide topic.Token
retype", merged 2026-09-01) changed `EnvProvider` to **return an error** when
the variable is unset or empty, where it previously fell back silently to the
token string. Under the old fallback, `"tenant.status"` resolved to the literal
topic name `tenant.status` and worked. Since #1608 every tenant emit fails.

The blast radius is the whole ephemeral-environment pipeline. Observed live in
`atlas-pr-1566` on 2026-09-02:

1. `atlas-tenants` `create_tenant` returns HTTP 500 with
   `topic token [tenant.status] has no value in the environment`
   (11:04:51, 11:05:03, 11:05:24, 11:06:05 UTC).
2. The `atlas-pr-bootstrap` Job fails all four attempts at step `tenant-create`
   (`curl: (22) The requested URL returned error: 500` → `tenant POST failed`),
   exhausting its backoff limit.
3. With no tenant, `atlas-configurations` never publishes a projection
   snapshot. `atlas-login` fatals with
   `configuration: projection snapshot not yet published`; `atlas-drops` gets
   HTTP 500 from `GET /api/configurations/services/00000000-...` and exits 1.
   Both CrashLoopBackOff.
4. The Argo application reports `Degraded`.

Because the token was never typed, `libs/atlas-kafka/gen`'s scanner — which
collects constants only when their type resolves exactly to `topic.Token`
(`gen/scan.go:20-24`) — never saw it. `EVENT_TOPIC_TENANT_STATUS` is therefore
absent from `topics.yaml`, `env-configmap.yaml`, `kafka-topics-configmap.yaml`,
every overlay, and the compose `.env.example`. The topic is also missing from
the Kafka pre-create manifest, so it is not pre-created in any environment.

`tools/topicguard` exists to prevent exactly this class and did not fire. Its
`bare-token-literal` diagnostic does inspect references to untyped string
constants reaching a `topic.Token` parameter, but only reports when the
constant's **value** matches `rawEnvTopicPattern` =
`^[A-Z0-9_]*TOPIC[A-Z0-9_]*$` (`tools/topicguard/analyzer.go:70,194`). A legacy
lowercase topic name like `tenant.status` — precisely the shape that still needs
migrating — does not match and passes silently. Verified by running
`GUARD_SKIP_SELFTEST=1 GUARD_SERVICE_MODULES="services/atlas-tenants/atlas.com/tenants" ./tools/go-analyzer-guards.sh`
against unmodified `main`: `go-analyzer-guards: PASS`.

## 2. Goals

Primary goals:

- Restore tenant creation in `atlas-tenants` so new ephemeral environments
  bootstrap successfully.
- Declare the tenant status topic as a properly typed `topic.Token` so it flows
  into the generated topic manifest and every deploy surface.
- Close the `topicguard` gap so a lowercase/legacy topic-name constant reaching
  a `topic.Token` parameter is a build failure, not a silent runtime error.

Non-goals:

- Reverting or softening #1608's `EnvProvider` erroring behavior. Erroring on an
  unset token is the correct contract; the silent fallback is what let this
  latent bug live undetected.
- Adding a consumer for the tenant status topic. Nothing in the repo consumes
  it today (verified: `"tenant.status"` appears in exactly one file).
- Fixing the unrelated `kubectl logs` timeouts observed against nodes `eos`,
  `helios`, and `theia` during triage.
- Any change to the map/channel work in PR #1566, whose environment merely
  exposed this defect.

## 3. User Stories

- As a developer opening a PR, I want the ephemeral environment to bootstrap so
  I can exercise my change instead of debugging a `Degraded` Argo app.
- As a developer adding a new Kafka topic, I want `topicguard` to reject an
  improperly declared token at build time so the failure surfaces in CI rather
  than as a 500 in a live environment.
- As an operator, I want every topic a service emits to be present in the
  generated manifest so it is pre-created with the correct cleanup policy in
  every environment.

## 4. Functional Requirements

### FR-1: Retype the tenant status token

- FR-1.1 `EventTopicTenantStatus` in
  `services/atlas-tenants/atlas.com/tenants/tenant/kafka.go` MUST be declared
  as `EventTopicTenantStatus topic.Token = "EVENT_TOPIC_TENANT_STATUS"`,
  matching the repo-wide idiom (`services/atlas-configurations/.../tenants/processor.go:27`).
- FR-1.2 The constant MUST move out of the `const (...)` block it currently
  shares with the untyped `EventTypeCreated`/`Updated`/`Deleted` string
  constants, or that block must be restructured, so the type applies only to
  the token.
- FR-1.3 The three `mb.Put(EventTopicTenantStatus, ...)` call sites in
  `services/atlas-tenants/atlas.com/tenants/tenant/processor.go` (lines 88, 164,
  230) MUST continue to compile unchanged.
- FR-1.4 `services/atlas-tenants/atlas.com/tenants/tenant/testmain_test.go:10`
  currently does `os.Setenv(string(tenant.EventTopicTenantStatus), string(tenant.EventTopicTenantStatus))`,
  which masks the defect by making the lookup succeed with the token as its own
  value. It MUST be updated to set the new token to a realistic resolved topic
  name distinct from the token itself (e.g. `EVENT_TOPIC_TENANT_STATUS` →
  `tenant-status-test`), so a future regression of this class cannot pass the
  suite.

### FR-2: Regenerate deploy surfaces

- FR-2.1 `tools/gen-topics.sh` MUST be run (not hand-edited) to regenerate the
  derived surfaces. It is the single source of truth for:
  `libs/atlas-kafka/gen/topics.yaml`, `deploy/k8s/base/env-configmap.yaml`,
  `deploy/k8s/base/kafka-topics-configmap.yaml`,
  `deploy/k8s/overlays/{main,pr,pr-sparse}/kustomization.yaml`, and
  `deploy/compose/.env.example`.
- FR-2.2 After regeneration, `EVENT_TOPIC_TENANT_STATUS` MUST appear in
  `topics.yaml` attributed to package `atlas-tenants/tenant`.
- FR-2.3 The token MUST appear in `env-configmap.yaml` and in each overlay with
  that overlay's suffix convention (`-main`, `-PLACEHOLDER_ATLAS_ENV`,
  `-PLACEHOLDER_BASELINE_ENVIRONMENT`).
- FR-2.4 The topic MUST appear in `kafka-topics-configmap.yaml` with
  `cleanup: delete`. It MUST NOT be added to
  `libs/atlas-kafka/gen/policies.yaml`'s `compact` list: that list is justified
  by consumers that replay from first offset at every boot to rebuild config
  state, and the tenant status topic has no such consumer.
- FR-2.5 `./tools/gen-topics.sh --check` MUST exit 0 after the change (the
  drift guard `tools/verify.sh` runs when any of these paths is touched).
- FR-2.6 Non-generated compose files under `deploy/compose/` that enumerate
  topic env vars per service (e.g. `docker-compose.socket.yml`) MUST be checked
  and updated if `atlas-tenants` appears in them.

### FR-3: Close the topicguard gap

- FR-3.1 `topicguard`'s `bare-token-literal` diagnostic MUST fire on an untyped
  string constant reaching a `topic.Token` parameter regardless of whether the
  constant's value matches `^[A-Z0-9_]*TOPIC[A-Z0-9_]*$`. The value-shape filter
  is what let `"tenant.status"` through and MUST be removed from, or inverted
  for, the untyped-constant path (`reportIfUntypedConstRef`).
- FR-3.2 The bare *string literal* path (`*ast.BasicLit`) SHOULD be widened the
  same way for consistency; if it is not, the design MUST record why. Note the
  raised false-positive risk: any string literal argument at a `topic.Token`
  position becomes a diagnostic.
- FR-3.3 `_test.go` files MUST remain excluded, per the existing rationale in
  `checkBareTokenLiteral`.
- FR-3.4 A test fixture under `tools/topicguard/testdata/` MUST cover a
  lowercase, dotted legacy topic name (the `tenant.status` shape) reaching a
  `topic.Token` parameter, and assert the diagnostic fires.
- FR-3.5 If widening produces violations elsewhere in the fleet, each MUST be
  fixed at the source. Adding entries to `tools/topicguard/allowlist.txt` is not
  an acceptable resolution — that allowlist is scoped to diagnostic 2
  (`raw-env-topic-read`) only.
- FR-3.6 `tools/go-analyzer-guards.sh` MUST pass fleet-wide after the change.

### FR-4: Regression coverage

- FR-4.1 A test MUST exist that fails on the pre-fix code: assert that the
  tenant status emit path resolves its token through `topic.EnvProvider`
  successfully when `EVENT_TOPIC_TENANT_STATUS` is set, and returns an error
  when it is not. Per repo convention this is a real test, not a
  `*_testhelpers.go` construct, and uses the Builder pattern for setup.
- FR-4.2 The `topicguard` fixture from FR-3.4 is itself a regression test for
  the guard gap and MUST be present.

## 5. API Surface

No REST surface changes. The externally visible behavior change is that
`POST /api/tenants` (and the update/delete equivalents) stops returning HTTP 500
and returns its normal success response.

## 6. Data Model

No schema change, no migration.

**Wire-format note:** the Kafka topic name changes from the literal
`tenant.status` to `EVENT_TOPIC_TENANT_STATUS-<env>` (per-overlay suffix). This
is safe because no consumer of `tenant.status` exists anywhere in the repo —
verified by a repo-wide grep for the literal across `*.go`, `*.yaml`, and
`*.yml`, which returns exactly one hit: the declaration itself. Any messages
sitting on a legacy `tenant.status` topic in a long-lived environment are
therefore unread and can be abandoned.

## 7. Service Impact

| Area | Change |
|---|---|
| `services/atlas-tenants` | Retype the token constant (FR-1); fix the `testmain_test.go` env hack; add regression coverage. Behavior: tenant create/update/delete emits succeed again. |
| `libs/atlas-kafka/gen` | `topics.yaml` regenerated to include the new token. `policies.yaml` unchanged. |
| `deploy/k8s/base` | `env-configmap.yaml` and `kafka-topics-configmap.yaml` regenerated. |
| `deploy/k8s/overlays/*` | All four `kustomization.yaml` literal lists regenerated. |
| `deploy/compose` | `.env.example` regenerated; per-service compose files checked. |
| `tools/topicguard` | Analyzer widened (FR-3) plus fixtures. |
| Every other service | None directly, but each ephemeral environment recovers once this lands, since bootstrap can create its tenant again. |

## 8. Non-Functional Requirements

- **Multi-tenancy:** unchanged. The token resolves per-environment via the
  overlay suffix, identical to every other topic.
- **Observability:** the `atlas-tenants` 500 currently logs
  `topic token [tenant.status] has no value in the environment` at `error` with
  a trace id — adequate. No new logging required.
- **Security:** none.
- **Performance:** none.
- **Deployment ordering:** the code change and the deploy-manifest change must
  land in the same commit/PR. Shipping the retype without the configmap entry
  would leave the token unset and reproduce the same 500 under a new name.

## 9. Open Questions

1. **FR-3.2 scope.** Widening `bare-token-literal` to all string literals at a
   `topic.Token` position may surface false positives in non-test production
   code that legitimately passes a computed or placeholder string. The design
   phase should run the widened analyzer fleet-wide first and let the actual
   violation count decide between "widen both paths" and "widen the
   untyped-constant path only."
2. **Legacy topic cleanup.** Whether the now-orphaned `tenant.status` topic
   should be explicitly deleted from long-lived Kafka clusters (`atlas-main`), or
   left to age out under its retention policy. Leaning: leave it; it is unread
   and harmless.
3. **Sweep for siblings.** A grep for lowercase topic-shaped constants across
   `services/` and `libs/` found `EventTopicTenantStatus` as the only remaining
   instance. The widened guard from FR-3 is the authoritative sweep and may
   surface cases the grep heuristic missed; the design phase should report its
   fleet-wide output.

## 10. Acceptance Criteria

- [ ] `EventTopicTenantStatus` is declared `topic.Token = "EVENT_TOPIC_TENANT_STATUS"`.
- [ ] `testmain_test.go` no longer sets the token to its own name.
- [ ] A table-driven regression test pins the token's shape (FR-4.1) and fails
      against the pre-fix constant. Resolution behaviour itself is EnvProvider's
      and stays covered by `libs/atlas-kafka/topic/provider_test.go`.
- [ ] `EVENT_TOPIC_TENANT_STATUS` is present in `topics.yaml`,
      `env-configmap.yaml`, `kafka-topics-configmap.yaml` (`cleanup: delete`),
      the three topic-carrying overlay `kustomization.yaml` files (`main`,
      `pr`, `pr-sparse` — `pr-cleanup` declares no topic env vars), and
      `.env.example`, all produced by `tools/gen-topics.sh` rather than
      hand-edited.
- [ ] `./tools/gen-topics.sh --check` exits 0.
- [ ] `topicguard` fires on a lowercase dotted legacy topic name reaching a
      `topic.Token` parameter, proven by a `testdata` fixture.
- [ ] `./tools/go-analyzer-guards.sh` passes fleet-wide, with no new
      `allowlist.txt` entries.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review passes before the PR opens.

### Post-merge verification (not part of the branch's own gates)

- [ ] `atlas-tenants:latest` rebuilds from `main` — the ephemeral overlays pin
      `atlas-tenants` to `:latest`, not a per-PR tag, so no PR rebase is needed
      for the image.
- [ ] Ephemeral environments re-resolve their `bot/pr-<n>-resolved` branch and
      pick up the regenerated configmap.
- [ ] The exhausted `atlas-pr-bootstrap` Job is deleted in each affected
      namespace so Argo recreates it — a `Failed` Job past its backoff limit
      does not retry on its own.
- [ ] `atlas-pr-1566` reaches `Healthy`, with `atlas-login` and `atlas-drops`
      out of CrashLoopBackOff.
