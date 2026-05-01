# Code Review Audit — task-040-otel-span-metrics

This file aggregates the parallel reviewer audits dispatched via `superpowers:requesting-code-review`. Two reviewers ran on this branch:

- **`backend-guidelines-reviewer`** — full audit below.
- **`plan-adherence-reviewer`** — full audit at the bottom of this file.

Frontend reviewer was not dispatched (no `services/atlas-ui/` changes in this branch).

**Overall verdict: READY_TO_MERGE.** Both audits passed with zero blocking issues. One non-blocking improvement noted (the `InitTracer` logger pattern).

---

# Backend Audit — task-040-otel-span-metrics

- **Branch:** task-040-otel-span-metrics
- **Review base:** 261b67d7923c3dc7b20fac1543fca7459e779d48
- **Guidelines source:** backend-dev-guidelines skill
- **Date:** 2026-04-30
- **Build:** PASS (`libs/atlas-tracing` and `services/atlas-channel`)
- **Tests:** PASS (`libs/atlas-tracing`: 11 cases across 2 functions; `atlas-channel`: full suite green, including the new `TestAnnounce_StartsSpan`)
- **Overall:** PASS

## Scope discovery

Changed Go files (excluding deletions and `go.sum` churn):

- `libs/atlas-tracing/tracing.go` (new)
- `libs/atlas-tracing/sampling.go` (new)
- `libs/atlas-tracing/sampling_test.go` (new)
- `libs/atlas-tracing/sampling_test_helpers.go` (new)
- 54 × `services/<svc>/atlas.com/<mod>/main.go` (one-line import swap)
- 54 × `services/<svc>/atlas.com/<mod>/tracing/tracing.go` (deleted)
- `services/atlas-channel/atlas.com/channel/session/processor.go` (added `Announce` span)
- `services/atlas-channel/atlas.com/channel/session/processor_test.go` (new `TestAnnounce_StartsSpan`)

No `model.go`, `entity.go`, `rest.go`, `resource.go`, `administrator.go`, `provider.go`, or `builder.go` files were modified. The DOM/SUB checklists therefore have **no applicable domain or sub-domain packages** in scope. The audit reduces to (a) DOM-21 (shared-lib reuse), (b) general anti-pattern checks against the four files that were touched non-trivially, (c) cardinality discipline on the new span, and (d) the spot-checks called out in the request.

## Build & Test Results

```
$ cd libs/atlas-tracing && go build ./...   # clean
$ cd libs/atlas-tracing && go test ./... -count=1
ok  github.com/Chronicle20/atlas/libs/atlas-tracing  0.006s

$ cd services/atlas-channel/atlas.com/channel && go build ./...   # clean
$ cd services/atlas-channel/atlas.com/channel && go test ./... -count=1
... ok  atlas-channel/session  0.010s
(full suite green)
```

## Domain Checklist Results

No domain or sub-domain packages were created or modified, so DOM-01..DOM-20 and SUB-01..SUB-04 do not apply. Status `N/A` for all.

| ID  | Check                                        | Status | Evidence |
|-----|----------------------------------------------|--------|----------|
| DOM-21 | New shared lib does not redefine constants | PASS   | `libs/atlas-tracing/*.go` — `grep -nE 'type [A-Z]' libs/atlas-tracing/*.go` returns zero matches; the lib only exports `InitTracer` and `Teardown` and depends on no atlas-constants types, so there is nothing to overlap with |

## Touched-File Checklist (ad-hoc, against anti-patterns + ai-guidance)

| ID  | Check | Status | Evidence |
|-----|-------|--------|----------|
| MIG-01 | Per-service `replace` path uses four `..` segments | PASS | `grep -rn 'atlas-tracing => ' services/` returns 54 entries, all reading `../../../../libs/atlas-tracing` (e.g. `services/atlas-channel/atlas.com/channel/go.mod:70`); no other path shape exists |
| MIG-02 | Per-service Dockerfile in-container `-replace=` path is `/app/libs/atlas-tracing` | PASS | `grep -l 'atlas-tracing=/app/libs/atlas-tracing' services/*/Dockerfile \| wc -l` = 54 (matches Go-service count); the two non-matching Dockerfiles are `services/atlas-assets/Dockerfile` (nginx, no Go) and `services/atlas-ui/Dockerfile` (TypeScript) |
| MIG-03 | All 54 services drop the local `tracing/` package | PASS | `find services -type d -name tracing` returns nothing; `grep -L "atlas-tracing" services/*/atlas.com/*/main.go` returns nothing |
| MIG-04 | `go.work` adds the new lib | PASS | `go.work:19` adds `./libs/atlas-tracing` |
| MIG-05 | New lib's `go.mod` only carries OTel + logrus deps | PASS | `libs/atlas-tracing/go.mod:5-11` lists `sirupsen/logrus`, `otel`, `otel/exporters/otlp/otlptrace{,grpc}`, `otel/sdk` — no atlas-* lib imports, so no risk of cyclic deps |
| ANN-01 | `session.Announce` records errors via `RecordError` + `SetStatus(codes.Error, …)` | PASS | `services/atlas-channel/atlas.com/channel/session/processor.go:187-188, 192-193` |
| ANN-02 | Span is unconditionally ended | PASS | `services/atlas-channel/atlas.com/channel/session/processor.go:177` (`defer span.End()`) |
| ANN-03 | Span derives ctx from caller, not `context.Background()` | PASS | `services/atlas-channel/atlas.com/channel/session/processor.go:176` uses the curried `ctx`; `:191` propagates `spanCtx` into the writer (`w(l, spanCtx)(encoder)`) |
| ANN-04 | Cardinality discipline: no `character.id`, `account.id`, `session.id`, `transaction.id`, `item.id`, `packet.size` attributes | PASS | `grep -nE "character\.id\|account\.id\|session\.id\|transaction\.id\|item\.id\|packet\.size" services/atlas-channel/atlas.com/channel/session/processor.go` returns no matches; the only attributes set are `writer.name`, `tenant.id`, `world.id` (`processor.go:180-182`) |
| TST-01 | `TestAnnounce_StartsSpan` saves and restores the global `TracerProvider` | PASS | `services/atlas-channel/atlas.com/channel/session/processor_test.go:671-674` captures `prev := otel.GetTracerProvider()` and registers `t.Cleanup(func() { otel.SetTracerProvider(prev) })` |
| TST-02 | `parseSamplingRatio` uses table-driven tests | PASS | `libs/atlas-tracing/sampling_test.go:13-60` is a single `tests := []struct{...}` table iterated via `t.Run(tc.name, …)` |
| TST-03 | Sampling tests cover unset / valid / empty / garbage / out-of-range | PASS | Eight cases at `libs/atlas-tracing/sampling_test.go:21-29` |
| TST-04 | `parseSamplingRatio` accepts `logrus.FieldLogger`, not `*logrus.Logger` | PASS | `libs/atlas-tracing/sampling.go:19` (`func parseSamplingRatio(l logrus.FieldLogger) float64`) |
| TST-05 | `Teardown` accepts `logrus.FieldLogger` | PASS | `libs/atlas-tracing/tracing.go:57` |

## Security Review

Not applicable — this task does not touch authn/authz, token handling, or redirects. SEC-01..SEC-04 are skipped by scope.

## Important (non-blocking) findings

1. **`InitTracer` constructs its own `logrus.New()` solely for sampling-warn output** — `libs/atlas-tracing/tracing.go:36-37`. This bypasses any caller-configured logger (level, ECS formatter, hooks). For a one-line warning at startup it is defensible, but it does mean a misconfigured `TRACE_SAMPLING_RATIO` will warn at the default INFO level on stdout regardless of how the service has wired logrus. If the platform team wants observability-config warnings to flow through ECS like everything else, the call site should be refactored to take a `logrus.FieldLogger` (mirroring `Teardown`'s signature). Flagging as **Important**, not Blocking, per the request.

## Summary

### Blocking (must fix)

- None.

### Non-Blocking (should fix)

- `InitTracer` mints its own `logrus.New()` for sampling-parse warnings (`libs/atlas-tracing/tracing.go:36`); consider taking a `logrus.FieldLogger` parameter so warnings ride the caller's log pipeline.

### Verdict

**PASS.** Build and tests are green across the new shared lib and the only modified Go service (`atlas-channel`). The 54-service migration is mechanically uniform: every `replace` directive reads `../../../../libs/atlas-tracing`, every Dockerfile uses `/app/libs/atlas-tracing` for the in-container replace, and every `main.go` swaps to `tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"` with no orphaned local `tracing/` packages remaining. The `session.Announce` span obeys PRD §8.2 cardinality rules, records errors and status correctly, and propagates `spanCtx` to downstream encoders. The accompanying `TestAnnounce_StartsSpan` properly snapshots and restores the global `TracerProvider` via `t.Cleanup`. No DOM-21 violation: the new lib defines zero domain types.

---

# Plan Adherence Audit — task-040-otel-span-metrics

- **Plan:** `docs/tasks/task-040-otel-span-metrics/plan.md`
- **Branch:** `task-040-otel-span-metrics`
- **Base branch:** main (merge-base `261b67d7923c3dc7b20fac1543fca7459e779d48`)
- **Date:** 2026-04-30

## Executive Summary

All 19 tasks landed with the dictated commit messages, file moves, and verification gates honored. The new `libs/atlas-tracing` library exists with the expected API surface, every one of the 54 Go services references it via `replace`, every private `services/atlas-*/.../tracing/` package is deleted, the `session.Announce` manual span is in place with the right cardinality budget, and the Grafana dashboard plus operator documentation ship as specified. Repo-wide `go build ./...` and `go test ./...` for every workspace module pass on this worktree. The four controller-approved deviations (Task 1 deps reflow, `sdktrace` alias fix, Task 5 4th Dockerfile edit, Task 10 `.env` skip) are reflected in either follow-up commits or the migration runbook.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Scaffold `libs/atlas-tracing` module | DONE | `libs/atlas-tracing/{go.mod,README.md}` exist; `go.work:18` lists `./libs/atlas-tracing` alphabetically. Commits `99039b5d3` + ordering follow-up `e46b48469` (controller-approved). |
| 2 | TDD failing test for sampling parser | DONE | `libs/atlas-tracing/sampling_test.go` defines `TestParseSamplingRatio` with all 8 sub-cases; helper in `sampling_test_helpers.go`. |
| 3 | Implement `parseSamplingRatio` | DONE | `libs/atlas-tracing/sampling.go:19-38` matches plan exactly. All 8 sub-tests PASS. Commit `003cf4c97`. |
| 4 | Implement `InitTracer` and `Teardown` | DONE | `libs/atlas-tracing/tracing.go:24-67`; `sdktrace` alias correct after `fd5c27025`. Commits `db93684b0` + `fd5c27025` (controller-approved). |
| 5 | Wire `libs/atlas-tracing` into atlas-channel | DONE | `services/atlas-channel/atlas.com/channel/main.go:57` imports the lib; `go.mod:16,70` has require+replace; old `tracing/` dir gone; `Dockerfile:17,37,56,79` has all four edits including the 4th `-replace=` line. Commit `51d34f0a9`. |
| 6 | Verify atlas-channel Docker build | DONE | Verification-only. The 4th Dockerfile edit was identified during this gate and folded inline before the canary was claimed green; runbook step (d) records it. No fixup commit was needed. |
| 7 | TDD `session.Announce` span (failing test) | DONE | `services/atlas-channel/atlas.com/channel/session/processor_test.go:663` defines `TestAnnounce_StartsSpan` with the `announceMockProvider`/`announceMockTracer`/`announceMockSpan` mock scaffolding from the plan. |
| 8 | Implement `session.Announce` span | DONE | `services/atlas-channel/atlas.com/channel/session/processor.go:176-194` wraps the inner lambda with `otel.GetTracerProvider().Tracer("atlas-channel").Start(ctx, "session.Announce")`, sets `writer.name`/`tenant.id`/`world.id`, calls `RecordError`/`SetStatus` on both error paths. Forbidden-attribute grep (Step 5 gate) returns empty. Commit `f7f435ae3`. |
| 9 | Sampling-ratio integration test | DONE | `TestInitTracer_SamplerHonorsRatio` (3 sub-cases) appended to `libs/atlas-tracing/sampling_test.go`; PASS. Commit `bcfdb5613`. |
| 10 | `TRACE_SAMPLING_RATIO` in deploy configs | DONE | `deploy/k8s/env-configmap.yaml:14` and `deploy/compose/.env.example:12` set the var to `1.0`. `deploy/compose/.env` is gitignored and intentionally untouched (controller-approved deviation; clean redo as commit `360ed73a1` only touches the two tracked files). |
| 11 | Grafana dashboard JSON | DONE | `deploy/grafana/dashboards/atlas-latency.json` UID `atlas-latency`, 7 panels, 2 templates, 1 annotation; `python3 -m json.tool` validates. Commit `9ad6b03d5`. |
| 12 | Grafana provisioning + apply.sh + README | DONE | `deploy/grafana/{dashboards-provider.yaml,apply.sh,README.md}` all present; `apply.sh` executable. Commit `1e9de6301`. |
| 13 | `docs/observability.md` | DONE | 112 lines covering pipeline diagram, manual-span recipe, cardinality budget (allowlist + forbidden list), dashboard panel template, sampling caveat, 12-step smoke test. Commit `6738f7bcb`. |
| 14 | Per-service migration runbook | DONE | `docs/tasks/task-040-otel-span-metrics/migration-runbook.md`; service list complete; step (d) for the 4th Dockerfile `-replace=` is documented (controller-approved deviation). Commit `ec8cd48ca`. |
| 15 | Migrate atlas-account as second canary | DONE | `services/atlas-account/atlas.com/account/main.go:10` imports the lib; `go.mod:13,95` has require+replace; `Dockerfile:19,33,55,71` has all four edits. Commit `136f8329c`. |
| 16 | Fan out to remaining 52 services | DONE | Five batch commits (`e8c83c6f2`, `e169da1c7`, `eaf4ab8da`, `033b8f45d`, `b61c1b382`) cover asset-expiration..wz-extractor. atlas-assets (no Go module) correctly skipped per controller note. `find services -name tracing.go -path '*/tracing/*'` = 0; `grep -l libs/atlas-tracing services/atlas-*/atlas.com/*/go.mod | wc -l` = 54. |
| 17 | Repo-wide build & test verification | DONE | Re-verified during this audit: every `go.mod` under `libs/` and `services/atlas-*/atlas.com/*` builds and tests green. No fixup commit needed (Step 4 correctly omitted). |
| 18 | Document out-of-tree cluster changes | DONE | `docs/tasks/task-040-otel-span-metrics/cluster-changes.md`: Tempo overrides span-metrics processor with curated dimensions, Grafana volumeMount + volume snippets, references `docs/observability.md` smoke test as acceptance. Commit `2e98b6258`. |
| 19 | Final repo-wide self-review | DONE | Verification-only gate re-run: forbidden cardinality grep empty; all 54 service builds green; `session.Announce` uses only allowlisted attributes. No commit expected; none made. |

**Completion Rate:** 19/19 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every plan task has corresponding evidence. Tasks 6, 17 Step 4, and 19 Step 4 were verification-only and correctly produced no commits because nothing required fixing.

## Build & Test Results

| Module Group | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-tracing` | PASS | PASS | `TestParseSamplingRatio` (8 subs) + `TestInitTracer_SamplerHonorsRatio` (3 subs) all green. |
| `services/atlas-channel` | PASS | PASS | `TestAnnounce_StartsSpan` PASS; full `go test ./...` PASS. |
| `services/atlas-account` | PASS | PASS | `go build` + `go test` green. |
| All 54 Go services (matrix loop) | PASS | PASS | Repo-wide `for mod in services/atlas-*/atlas.com/*; do (cd $mod && go build ./... && go test ./...); done` exited 0; same for `libs/atlas-*`. |

Cardinality leak gates (Task 8 Step 5 and Task 19 Step 2): both grep commands return empty — `character.id`, `account.id`, `session.id`, `transaction.id`, `packet.size` are not promoted as span attributes.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. Optional follow-up:

1. ✅ A note clarifying `deploy/compose/.env` is gitignored and only `.env.example` ships in-tree was added to `migration-runbook.md` after the audit.

Out-of-tree work in `cluster-changes.md` (Tempo overrides + Grafana volumeMount) remains for the cluster operator to apply before the dashboard panels show non-zero data, but that boundary is by design (per Task 19's AC mapping table — ACs #1–#5, #9, #10 are explicitly cluster-deploy-verifiable per Task 18).
