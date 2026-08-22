# Plan Audit — task-260-kafka-precreate-go-tool

**Plan Path:** docs/tasks/task-260-kafka-precreate-go-tool/plan.md
**Audit Date:** 2026-08-22
**Branch:** task-260-kafka-precreate-go-tool
**Base Branch:** main (merge base `c47469811`)
**Range Audited:** all 8 plan tasks (full plan)

## Executive Summary

All 8 plan tasks were implemented and are supported by direct evidence in the diff: the `discover`/`kafkaops`/`topics`/`groups` packages, `main.go`'s five-phase pipeline, the Dockerfile/README, service registration, bake target, base Job rewrite, all three overlay wirings, and the shell script's deletion. Both deviations the dispatcher flagged were verified against the code and are correctly described, not defects: the cold-start retry contract (commit `fbff7c891`) replaces the plan's originally-fatal partition-level `LeaderNotAvailable`/`NotLeaderForPartition`/coordinator-election codes with bounded retries, backed by live-broker evidence and new passing tests; and Task 6's two-commit split (`8c06feb43` deletion-only, `23ecfa0f3` modification-only) touches disjoint file sets with no double-counted or missing content. Module-local `go build ./...`, `go vet ./...`, and `go test ./... -count=1` all pass cleanly at HEAD. Two informational, previously-deferred cosmetic notes remain (a message-wording inconsistency and a config-resource ordering comment whose only covering test is pre-sorted input); neither is a functional defect.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Module scaffold and `discover` package | DONE | `services/atlas-kafka-precreate/go.mod`, `internal/discover/discover.go` (125 lines), `discover_test.go` (189 lines); `go.work:53` adds the module. `go mod tidy` correctly stripped unused requires per the ledger's ruling (not a defect). |
| 2 | `kafkaops` seam and coordinator retry | DONE | `internal/kafkaops/ops.go:17-25` — 7-method `AdminClient` interface, `var _ AdminClient = (*kafka.Client)(nil)` static assertion at :27. `RetryConfig`/`DefaultRetryConfig`/`WithCoordinatorRetry` at :29-92 (later generalized in the Task 8 fix, see below). |
| 3 | `topics` package — create, settle, end offsets | DONE | `internal/topics/topics.go`: `Ensure` (:45), `Settle` (:157), `EndOffsets` (:235), `missingTopicsError`/`firstLeaderError` helpers. `IncrementalAlterConfigs` used (not legacy `AlterConfigs`), per plan's OQ-1 note at topics.go:90-101. |
| 4 | `groups` package — seed and verify | DONE | `internal/groups/groups.go`: `Seed` (:46), `Verify` (:218), `probeState` (:167), `firstCoordinatorError`/`firstFetchCoordinatorError`. Error wraps name the group (`groups.go:90,213,216` per ledger, confirmed present in current file at equivalent lines post-fix). |
| 5 | `main.go`, Dockerfile, README | DONE | `main.go:32` `run()` drives discover→create/alter→settle→seed→verify in `main.go:62-161`, reading every field of `SeedResult`/`VerifyReport`. `services/atlas-kafka-precreate/Dockerfile` and `README.md` present. |
| 6 | Cutover — build registration, Job manifest, delete shell | DONE | `.github/config/services.json:225-230` new `support-image` entry; `docker-bake.hcl:142-148` target + `:155` added to `all-services`; `deploy/k8s/base/atlas-kafka-precreate.yaml` rewritten to `envFrom`-only container, no `command`/volumes; `deploy/k8s/base/kafka-precreate.sh` and `atlas-kafka-precreate_test.sh` deleted (confirmed absent on disk). Landed as two commits (`8c06feb43` deletion-only 682 lines removed, `23ecfa0f3` modification-only across 4 other files, 40+/35-) — disjoint file sets, no double-count. Ledger records this as a mechanical `git add` staging issue; verified not a content defect. |
| 7 | Overlay wiring and dangling references | DONE | `deploy/k8s/overlays/main/kustomization.yaml`, `pr/kustomization.yaml:436-437`, `pr-sparse/kustomization.yaml:571-572` all carry the new `images:` entry. `pr-sparse/kustomization.yaml:272-274` comment correctly names real symbols `groups.Seed`/`groups.Verify` (`internal/groups/groups.go:46,191`) after the `df0f420d1` fix — confirmed by direct read, no invented names remain. |
| 8 | Full gate | DONE | Flagless `tools/verify.sh` PASS at final HEAD `79fdb0649` per ledger (not re-run here per dispatcher instruction). `docker buildx bake atlas-kafka-precreate` PASS per ledger. Manual acceptance against a live KRaft broker: cold run 1130 ms, steady-state 31 ms, both exit 0 (NFR-1 <5s satisfied). Module-local `go build ./...`, `go vet ./...`, `go test ./... -count=1` all pass in this audit (confirmed independently, see Build & Test Results). |

**Completion Rate:** 8/8 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Deviations Verified

1. **Cold-start retriable-error contract (commit `fbff7c891`).** The plan's test tables at `plan.md:438` and `plan.md:593` originally modeled `kafka.LeaderNotAvailable`/`kafka.NotLeaderForPartition` (and the analogous coordinator-election codes on `OffsetCommit`/`OffsetFetch`) as fatal. Live-broker acceptance testing (`docs/tasks/task-260-kafka-precreate-go-tool/bug-cold-start-retriable-errors.md`) found these are transient cluster-warmup states. The fix commit generalizes `withRetry` behind a predicate (`ops.go:92`, confirmed present), adds `WithLeaderRetry` alongside the unchanged-signature `WithCoordinatorRetry`, wraps `topics.EndOffsets`'s `ListOffsets` call in the leader retry, and makes `groups.Seed`/`Verify` surface retriable per-partition errors through the existing retry `fn` rather than escaping as fatal. `kafka.UnknownMemberId` correctly retains its distinct meaning (commit-race → skip) and was deliberately not folded into the retriable set — confirmed by reading `groups.go:108-120`, which still branches on `UnknownMemberId` separately from the fatal wrap. Post-fix re-run against a brand-new broker: cold path now passes on the first attempt (1130 ms), matching the description. Module-local tests pass. This matches the dispatcher's description exactly and is not a defect.

2. **Task 6 two-commit split.** `8c06feb43` removes only `deploy/k8s/base/atlas-kafka-precreate_test.sh` and `deploy/k8s/base/kafka-precreate.sh` (682 lines deleted, 0 added). `23ecfa0f3` modifies only `.github/config/services.json`, `deploy/k8s/base/atlas-kafka-precreate.yaml`, `deploy/k8s/base/kustomization.yaml`, and `docker-bake.hcl` (40 insertions, 35 deletions). The two commits' file sets are fully disjoint; combined they equal the task's described scope with no gaps or duplication. Matches the dispatcher's description exactly and is not a defect.

## Skipped / Deferred Tasks

None. All 8 tasks are `DONE`. Two pre-existing minor/cosmetic items were carried forward by the ledger during implementation and remain informational only, not blocking:

- `internal/topics/topics.go:101-114` — `IncrementalAlterConfigs` resources are built by iterating `t.Compact` in given order rather than sorting by `ResourceName` in-package; correctness relies on `discover.Topics.Compact`'s documented sorted invariant, and the only covering test supplies already-sorted input. No functional defect observed; flagged for awareness only.
- `internal/groups/groups.go:114-115` (pre-fix line numbers, content confirmed still present post-fix) — singular/plural wording inconsistency between two commit-error messages. Cosmetic only.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-kafka-precreate | PASS | PASS | `go build ./...`, `go vet ./...` clean; `go test ./... -count=1`: `internal/discover`, `internal/groups`, `internal/kafkaops`, `internal/topics` all `ok`. Root package has no test files (expected — `main.go` is a thin wiring layer per the plan). |

Full-repo flagless `tools/verify.sh` was not re-run per dispatcher instruction; the ledger records it PASSed at final HEAD `79fdb0649` with all guards green (go analyzer, skill/job id, scope, producer seam, service registration, service name, env domain, env bootstrap, LB port drift, routes drift, version coverage, overlay env drift, tenant tables drift + generator tests, pr-sparse mirror drift, sparse baseline scoping) and the docker bake completing for every `go-service` target plus the new `atlas-kafka-precreate` target.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. Both flagged deviations were verified to match their stated descriptions and are not defects. The two carried-forward cosmetic notes (message wording, config-resource ordering comment) may optionally be addressed in a follow-up but do not block merge.
