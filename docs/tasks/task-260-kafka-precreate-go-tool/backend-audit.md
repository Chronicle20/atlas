# Backend Audit — atlas-kafka-precreate

- **Service Path:** services/atlas-kafka-precreate
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-22
- **Range:** `c47469811..79fdb0649`
- **Build:** PASS
- **Tests:** all packages passed (`ok` on discover, groups, kafkaops, topics; `main` has no test files), 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ cd services/atlas-kafka-precreate && go build ./...
(no output — success)

$ cd services/atlas-kafka-precreate && go test ./... -count=1
?   	atlas.com/kafka-precreate	[no test files]
ok  	atlas.com/kafka-precreate/internal/discover	0.002s
ok  	atlas.com/kafka-precreate/internal/groups	0.006s
ok  	atlas.com/kafka-precreate/internal/kafkaops	0.005s
ok  	atlas.com/kafka-precreate/internal/topics	0.005s
```

`tools/verify.sh` (flagless) already reported PASS at HEAD per the task brief; not re-run.

## Scope note (Job, not a standard service)

`atlas-kafka-precreate` is a one-shot sync-wave-0 Kubernetes `Job`
(`deploy/k8s/base/atlas-kafka-precreate.yaml:26` — `kind: Job`, `restartPolicy:
OnFailure`), not a long-running Deployment. It has no `model.go`, no
`resource.go`, no REST surface, no database, and talks Kafka's admin protocol
directly via `segmentio/kafka-go` rather than through `libs/atlas-kafka`'s
consumer/producer manager. All four packages (`discover`, `groups`,
`kafkaops`, `topics`) plus `main` classify as **support** packages (no
`model.go`, no `resource.go`).

## Applicability

| Family | Fired? | Evidence |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | No | No package in the diff has `model.go`, `entity.go`, `rest.go`, or `provider.go` — `find services/atlas-kafka-precreate -name '{model,entity,rest,provider}.go'` returns nothing. |
| FILE placement (FILE-01..06) | Yes | Unconditional — every changed Go package runs FILE-*. |
| SUB sub-domain (SUB-01..04) | No | No package has `resource.go`. |
| REST (DOM-06..09,12..15,17..19,32) | No | No `resource.go`/`rest.go`/`processor.go`, no HTTP routes registered — `main.go` never imports `net/http` or a `server` package. |
| Constants reuse (DOM-21) | Yes | `internal/discover/discover.go:15-27` declares `commandPrefix`, `eventPrefix`, `compactVars`; `internal/topics/topics.go:22` declares `compactCleanupPolicy`. |
| Testing (DOM-10,20,24,33) | Yes | Diff adds/changes `*_test.go` in all four `internal/*` packages. |
| Cache (DOM-29) | No | No `cache.go`; no processor/struct holds cached state (the tool is stateless per-run). |
| Messaging (DOM-30) | No | No `producer.go`; no `AndEmit`/`message.Emit`/`producer.ProviderImpl` call anywhere in the diff. |
| Multi-tenancy (DOM-31) | No | No `rest.go`; no tenant/trace read or `tenantId` passed anywhere — grepped, no hits. |
| Migration hygiene (DOM-34,35) | No | Diff does not move symbols between a service and a `libs/atlas-*` module. |
| Deploy & topics (DOM-22,23) | No | No `libs/atlas-*` module added (`git diff --stat` shows no `libs/` path); no new/renamed `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` env var (`deploy/k8s/base/env-configmap.yaml` untouched in this diff). |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed (`main.go`, all of `internal/*`). |
| Channel wire values (DOM-25) | No | Diff does not touch `atlas-channel` or `atlas-packet`. |
| Resilience (DOM-27,28) | No | No DB-backed handler, no `model.Decorator`. |
| External clients (EXT-01..04) | No | No `requests.RootUrl`/`requests.GetRequest[T]`/`requests.PostRequest[T]` call — the tool talks Kafka's wire protocol directly, not another atlas service's REST API. |
| Scaffolding (SCAFFOLD-01..09) | Yes | Diff adds `services/atlas-kafka-precreate/` with a `main.go`. |
| Security (SEC-01..04) | No | Service handles no auth, tokens, redirects, or secrets. |
| Foundational — patterns-provider.md | No | No provider composition (`provider.go` absent). |
| Foundational — patterns-functional.md | No | No curried constructors/decorators/combinators. |

## Checklist Results

### main (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Every goroutine via `routine.Go`, bare `go` needs guard marker | PASS | No `go ` statement anywhere in `main.go` or `internal/**/*.go` (non-test) — grepped `^\s*go \|go func` across the module, zero hits. |
| FILE-01..05 | Processor/RestModel/requests/Entity/Builder+Model+Administrator+Provider placement | N/A | None of these symbols exist anywhere in the module — grepped `type Processor interface`, `type RestModel`, `requests\.(Root|Get|Post)`, `type entity struct`, `func Migration(`, `TableName()`, `type Builder`, `func NewBuilder(`, `database\.(Slice)?Query` across `internal/` and `main.go`: zero hits. |
| FILE-06 | No `<pkg>.go` catch-all bundling ≥2 responsibilities | PASS | `main.go` is the wiring/orchestration entry point only (five phases, no Processor/RestModel/entity/requests content); each `internal/*` package's single `.go` file (`discover.go`, `groups.go`, `ops.go`, `topics.go`) is a genuine single-purpose module (env scrape+classification; group-coordinator ops; retry primitive; topic-controller ops respectively), none bundling two of the FILE-01..05 responsibility classes. |
| DOM-21 | No redeclared type/helper/constant already in `libs/atlas-constants/` | PASS | `commandPrefix`/`eventPrefix`/`compactVars` (`internal/discover/discover.go:15-27`) and `compactCleanupPolicy` (`internal/topics/topics.go:22`) have no match in `libs/atlas-constants/` — `grep -rn "COMMAND_TOPIC_\|EVENT_TOPIC_\|BOOTSTRAP_SERVERS\|KAFKA_CONSUMER_GROUP" libs/atlas-constants/` returns nothing. |
| — env var access | os.Getenv for config in a Job's `main` (no `resource.go`, DOM-12 does not apply) | N/A | DOM-12 ("No `os.Getenv()` in handlers") triggers only on packages with `resource.go`; `main.go` isn't a handler and is a one-shot process reading its own env, matching the exact pattern used by `libs/atlas-kafka/producer/manager.go:121` (`kafka.TCP(os.Getenv("BOOTSTRAP_SERVERS"))`) — the same lib the rest of the platform's producers/consumers use for this variable. No shared "get bootstrap servers" helper exists to redeclare. |

### internal/discover (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | Responsibility placement | N/A | Package holds none of the FILE-01..05 subject matter (pure env-scrape/classification functions); no catch-all collapse — see main's evidence, same grep swept this package. |
| DOM-20 | Table-driven tests | PASS | `internal/discover/discover_test.go:9` (`TestFromEnviron`), `:104` (`TestGroups`), `:167` (`TestStateIsSeedable`) all use `tests := []struct{...}` + `t.Run`. |

### internal/groups (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | Responsibility placement | N/A | No Processor/RestModel/requests/Entity/Builder subject matter; single-purpose file `groups.go` (group-coordinator probe/seed/verify). |
| DOM-20 | Table-driven tests | PASS | `internal/groups/groups_test.go:100` `TestSeed` (table + `t.Run` at `:288`), `:457` `TestVerify` (table + `t.Run` at `:591`), plus focused non-table tests for single scenarios (`TestSeed_MixedGroups`, `TestSeed_DescribeIsRetried`, etc.) — legitimate given each isolates one behavior. |
| — consumer-group probe never fails the Job | `groups.probeState` must collapse every failure mode to "not active" rather than propagating | PASS | `internal/groups/groups.go:178-192`: `probeState` returns `""` on a transport error, a per-group `Error`, or an absent group — the function has no error return at all, so it structurally cannot fail `Seed`'s caller. `StateIsSeedable("")` returns `true` (`internal/discover/discover.go:118`), matching the doc comment's stated allowlist design. |
| — fatal-vs-tolerated exit codes | `Ensure`/`EndOffsets`/`Seed`/`Verify` must surface real broker errors while tolerating expected non-fatal outcomes | PASS | `internal/topics/topics.go:75-79` tolerates `kafka.TopicAlreadyExists` per-topic (re-apply idempotency) while joining every other per-topic error as fatal; `internal/groups/groups.go:112-135` tolerates `kafka.UnknownMemberId` as a commit-race skip while treating every other per-partition commit error as fatal; `main.go:27-30` calls `os.Exit(1)` on any `run()` error. |

### internal/kafkaops (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | Responsibility placement | N/A | Package is a narrow retry/interface primitive (`AdminClient`, `RetryConfig`, `WithCoordinatorRetry`, `WithLeaderRetry`) — none of the FILE-01..05 subject matter. |
| DOM-20 | Table-driven tests | PASS | `internal/kafkaops/ops_test.go:44` `TestWithCoordinatorRetry` (table at `:47`, `t.Run` at `:119`), `:200` `TestWithLeaderRetry` (table at `:203`, `t.Run` at `:260`), plus targeted budget/cancellation tests. |

### internal/topics (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | Responsibility placement | N/A | Package is controller-facing Kafka ops (`Ensure`, `Settle`, `EndOffsets`) — no FILE-01..05 subject matter. |
| DOM-20 | Table-driven tests | PASS | `internal/topics/topics_test.go:183` `TestEnsure_CreateErrors` (table + `t.Run` at `:234`), plus extensive `t.Run` subtests covering Settle polling states (`:386-537`) and EndOffsets retry/error paths (`:572-742`). |

### Scaffolding (new service)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SCAFFOLD-01 | `services.json` has a `go-service` entry | **FAIL (Important)** | `.github/config/services.json` (diff hunk, `.github/config/services.json:224-230`) registers `atlas-kafka-precreate` with `"type": "support-image"`, not `"go-service"`. The rule's pass criteria is explicit: "Returns a non-empty object with `type: "go-service"`." Precedent (`atlas-pr-bootstrap`, also `support-image`) is not a documented exception — it is a non-Go (bash/rpk) image with its own toolchain, a materially different case, and prevalence does not exempt a deviation per the audit mindset. |
| SCAFFOLD-02 | Base manifest exists, listed in base kustomization | PASS, with a noted structural deviation | `deploy/k8s/base/atlas-kafka-precreate.yaml` exists, has no `namespace:` key (verified: `grep -n "namespace:"` returns nothing), and is listed at `deploy/k8s/base/kustomization.yaml:33`. It is `kind: Job` (`atlas-kafka-precreate.yaml:26`), not `Deployment + Service` — the pass criteria's shape describes the common (long-running REST/consumer) case; this workload is deliberately a one-shot sync-wave-0 Job (design doc + manifest header comment), which is the documented nature of this service, not an unexplained omission. |
| SCAFFOLD-03 | Build registration: `docker-bake.hcl` `go_services` entry + `go.work` `use()`, no per-service Dockerfile | **FAIL (Important)** | Three violations of this rule's explicit pass criteria ("a new Go service needs no Dockerfile of its own, and adding one is a finding"): (1) `services/atlas-kafka-precreate/Dockerfile` exists — a per-service Dockerfile; (2) `docker-bake.hcl:142-148` adds a bespoke `target "atlas-kafka-precreate"` block with `context = "services/atlas-kafka-precreate"`, and the service is **not** added to the `go_services` list (`docker-bake.hcl:35`) — only to the ad hoc `all-services` group (`docker-bake.hcl:155`); (3) `go.work:53` does add `./services/atlas-kafka-precreate` to `use()`, so that half is correct. Additionally, `deploy/k8s/overlays/main/kustomization.yaml`'s pinned tag for this image (`newTag: main-2080dc7`, added at `deploy/k8s/overlays/main/kustomization.yaml:282`) does not exist on the registry: `docker manifest inspect ghcr.io/chronicle20/atlas-kafka-precreate/atlas-kafka-precreate:main-2080dc7` → `manifest unknown`, and neither does the pr/pr-sparse overlays' `:latest` tag (same command against `:latest` → `manifest unknown`). By contrast, every neighboring service's pinned main-overlay tag resolves (spot-checked `atlas-invites:main-24a33a2` and `atlas-login:main-848eee4`, both present), and the other `support-image` peer `atlas-pr-bootstrap:latest` does resolve. `docs/adding-a-new-service.md` §3.3 explicitly requires confirming the tag exists on ghcr before merge ("confirm the tag exists on ghcr.io, e.g. `docker manifest inspect`"); as committed, it does not. |
| SCAFFOLD-04 | Ingress location block (REST only) | N/A | No REST surface — no `resource.go`, no HTTP routes registered in `main.go` (never imports `net/http` or a `server` package). |
| SCAFFOLD-05 | `routes.conf` regenerated (only if `routes.conf` changed) | N/A | `git diff --stat` shows `deploy/shared/routes.conf` untouched in this range. |
| SCAFFOLD-06 | docker-compose entry alongside peers | N/A | `deploy/compose/docker-compose.core.yml` has no Kafka broker service at all (`grep -n "kafka" deploy/compose/docker-compose.yml deploy/compose/docker-compose.core.yml` — zero hits), so the local compose stack has no target for a topic-precreation job to run against; there is no "alongside peers" model for a wave-0 ArgoCD sync job in a stack with no sync-wave concept and no Kafka broker. The other `support-image` peer `atlas-pr-bootstrap` (an ArgoCD PostSync/PostDelete hook, same category of non-long-running infra tool) is likewise absent from compose. |
| SCAFFOLD-07 | New channel Writer/Handler seeded in tenant templates | N/A | Diff registers no `atlas-channel` Writer/Handler and adds no `libs/atlas-packet/character/{clientbound,serverbound}` package. |
| SCAFFOLD-08 | Bruno collection (REST only) | N/A | No REST surface (see SCAFFOLD-04). |
| SCAFFOLD-09 | `tools/service-registration-guard.sh` exit 0 | PASS | Ran `tools/service-registration-guard.sh` — output `service-registration-guard: clean`, exit 0. |

## Security Review

Not applicable — SEC-* trigger did not fire (no auth/token/redirect/secret handling in the diff).

## Not evaluable from the diff

- Whether `main-publish.yml`'s change-detection/build-matrix step will actually pick up and publish `atlas-kafka-precreate` once this branch merges to `main` (its `docker-services-matrix` selection is keyed on `docker_image != null`, not on `type == "go-service"`, per `.github/workflows/main-publish.yml:75`, which suggests it will — but I did not execute the workflow, only read it statically) — would need a live CI run or `act`-style dry run to confirm.
- Whether the image tags currently pinned in the three overlays (`main-2080dc7`, `latest` × 2) get corrected by a subsequent automated "bump main overlay" commit before this branch is actually deployed — would need to see the next commit(s) on this branch or the merge/deploy sequence, which is outside this diff's range.

## Summary

### Blocking (must fix)
- SCAFFOLD-01: `.github/config/services.json:224-230` registers `atlas-kafka-precreate` as `type: "support-image"`, not the `go-service` the rule requires.
- SCAFFOLD-03: `services/atlas-kafka-precreate/Dockerfile` is a per-service Dockerfile (forbidden for Go services); the service is wired into `docker-bake.hcl` via a bespoke target rather than the `go_services` list; and the tags currently pinned in all three overlays (`deploy/k8s/overlays/main/kustomization.yaml:282` `main-2080dc7`, plus `pr` and `pr-sparse` overlays' `:latest`) do not resolve on `ghcr.io` (`docker manifest inspect` → `manifest unknown` for all three, vs. every checked neighbor resolving).

### Non-Blocking (should fix)
- None identified beyond the blocking items above.

### Not evaluable
- CI publish-matrix behavior for the new `support-image` entry after merge (see above).
- Whether overlay image tags self-correct via a later bump commit before deploy (see above).

---

## Controller response (2026-08-22) — both blocking findings REJECTED

Reviewed each blocking finding against repo evidence. Both misapply a
`go-service` rule to a service that is deliberately and precedentedly not one.
No code change made. Evidence below; each claim is reproducible.

### SCAFFOLD-01 — "`type` should be `go-service`, not `support-image`" — REJECTED

`support-image` is the correct type and was an explicit design decision, not an
oversight. `design.md:55-83` gives the rationale: the tool has **no
`libs/atlas-*` dependency**, so the shared root Dockerfile's twenty-two
`COPY libs/…` layers and synthesized `go.work` are pure cost, and its runtime
stage (`alpine`, `EXPOSE 8080`, nine placeholder data dirs, root user) directly
contradicts NFR-9 for a one-shot Job.

`services/atlas-pr-bootstrap` is the established precedent — the same
`type: "support-image"`, the same flat layout, its own Dockerfile, the same
registration shape. It is not a new pattern invented by this task.

The finding also assumes `type` selects what CI builds. It does not. The docker
build matrix selects on **`docker_image != null`**
(`.github/actions/detect-changes/action.yml:332`); `type == "go-service"`
(`:330`) selects the separate **Go build/test** matrix. `atlas-kafka-precreate`
has `docker_image` set, so it is built and pushed regardless of type.

### SCAFFOLD-03 — "per-service Dockerfile forbidden" — REJECTED

Same rationale and same precedent as above. The guideline's "a new Go service
needs no Dockerfile of its own" governs services that build from the shared root
Dockerfile; this image deliberately does not, for stated reasons. The bespoke
`docker-bake.hcl` target follows from the same decision and was verified by the
Task 6 reviewer, which ran `docker buildx bake --print` itself.

### SCAFFOLD-03 — "pinned tags do not resolve on ghcr.io; will wedge the wave-0 sync on merge" — REJECTED

The observation is accurate; the conclusion drawn from it is not. The tags do
not resolve because **this is a brand-new service whose image has never been
built**. That is true of every new service before its first publish and is not a
defect.

Two facts settle it:

1. **Support images are published.** `docker manifest inspect
   ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest` resolves.
   The pipeline demonstrably publishes `support-image` targets.
2. **The pinned tag is self-correcting.** `main-publish.yml:291` ("Bump newTag
   for every built service") rewrites the main overlay's `newTag` to
   `main-<short-sha>` for every service rebuilt in the run, matching on
   `.images[] | select(.name == "${image}")`. The overlay entry's name,
   `ghcr.io/chronicle20/atlas-kafka-precreate/atlas-kafka-precreate`
   (`deploy/k8s/overlays/main/kustomization.yaml`), matches `docker_image` in
   `services.json` exactly, so the bump step will match it. The placeholder
   `main-2080dc7` is replaced by CI on merge.

The `:latest` pins on the pr and pr-sparse overlays resolve the same way once
the PR build publishes.

### What stands

The audit's substantive assessment of the code is accepted and matches the two
prior reviews: real table-driven tests, correct fatal-vs-tolerated error triage,
a consumer-group probe structurally incapable of failing the Job, and the
retriable-error fix correctly wired into `topics.EndOffsets` and
`groups.Seed`/`groups.Verify`. Zero non-blocking findings and no goroutine,
cache, messaging, multi-tenancy, EXT-client, or security findings.
