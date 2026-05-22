# Plan Audit — task-076-task071-followups

**Plan Path:** docs/tasks/task-076-task071-followups/plan.md
**Audit Date:** 2026-05-22
**Branch:** task-076-task071-followups
**Base Branch:** main

## Executive Summary

All 20 followup tasks (F1-F20) plus the final branch-level acceptance gate (Task 21) landed faithfully on the task branch in 20 implementation commits sandwiched between the spec/design/plan commits. Every claim in the plan-summary section maps to a concrete diff with the expected file shape — temp-file buffering in publish.go, negative-verdict bypass in renders/storage, two-phase finalize in restore.go, atomic Properties() signature change across the monorepo, kustomize configMapGenerator + drift-check, F11 triage placeholder with explicit operator-pending status, F20 portal dedup with regression test. No tasks were silently skipped or downgraded; the only intentional placeholder (no-bounds-triage.json) is operator-side work the plan explicitly carved out. The user reported the branch verification gates (race/vet/build/bake/kustomize/routes_nginxt) clean locally — those gates are accepted on report rather than re-run here.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | F1 — publish 500 (buffer tar to tempfile + step prefixes) | DONE | commit 952e22193; `services/atlas-data/atlas.com/data/baseline/publish.go:46-100` (`os.CreateTemp`, `io.MultiWriter(tmp, h)`, every error wrapped with `publish:`), `publish_test.go` added |
| 2 | F7 — pin atlas-renders in main overlay | DONE | commit 6efb23c0f; `deploy/k8s/overlays/main/kustomization.yaml:278-279` placed between atlas-reactors (276-277) and atlas-saga-orchestrator (280-281) — alphabetical order preserved |
| 3 | F3 — don't cache "shared" verdicts | DONE | commit d37d4f55a; `services/atlas-renders/atlas.com/renders/storage/scope.go:32-48` only `Caches.Scope.Add` on tenant path (line 30), returns "shared" uncached (line 48); `smap.go:62-78` mirrors the pattern; `scope_test.go` regression added |
| 4 | F2 — drop outer ExecuteTransaction in commodity register | DONE | commit 49c4bc699; `services/atlas-data/atlas.com/data/commodity/processor.go:21-41` — explicit comment "the prior outer ExecuteTransaction wrapped the entire" plus no `database.ExecuteTransaction` references remain in the file; processor_test added |
| 5 | F5 — two-phase finalize baseline restore | DONE | commit 34ab6470a; `services/atlas-data/atlas.com/data/baseline/restore.go:41` `runRestoreTables`, `:75` `cleanupAfterFailure`, `:145-160` UPSERT lives after the loop succeeds; restore_failure_test added |
| 6 | F8 — dedupe routes via kustomize configMapGenerator | DONE | commit 7bfb84e6a; `tools/gen-routes.sh` exists (executable), `deploy/k8s/base/atlas-ingress.yaml` lost 453 lines of inline routes, `deploy/k8s/base/kustomization.yaml:66-69` defines `atlas-ingress-routes` configMapGenerator from `routes.conf.template.generated`, `deploy/k8s/base/routes.conf.template.generated` committed |
| 7 | F18 — routes drift check in CI | DONE | commit 2970c0a9d; `deploy/shared/test/routes_nginxt.sh:104-125` re-runs `tools/gen-routes.sh` and `git diff --quiet` against the generated file, fails on stale output, restores file in cleanup |
| 8 | F6 — `Properties()` returns `([]Property, error)` (atomic monorepo update) | DONE | commit 834e3fc58; `libs/atlas-wz/wz/image.go:60` signature is `func (i *Image) Properties() ([]property.Property, error)`. Commit touches all callers across libs/atlas-wz/{charparts,icons,mapimage} and services/atlas-data/{data/workers/{ui,item,skill},wztoxml/adapter}.go — the commit body enumerates every site (~14 explicit files, 17+ individual call expressions counted as `Properties() x2/x3/x4`) |
| 9 | F4 — annotate wz seek-path concurrency invariants | DONE | commit 39df3294a; `libs/atlas-wz/wz/file.go:219-225` annotates `tryParseWithVersion` (single-threaded by construction during Open), `libs/atlas-wz/wz/image.go:111-124` invariant comment on `parsePropertyList` ("caller holds wz.parseMu"), worker runtime gets a parallel-by-Image rationale |
| 10 | F15 — extract layout-common helper | DONE | commit a3d60a012; `libs/atlas-wz/mapimage/layers.go:44` `extractLayoutCommon` helper, `:113-115` `ExtractLayout` delegates, `:167-170` `ExtractLayers` delegates — net -3 lines, no behavior change |
| 11 | F16 — `accessoryPartClassFor` delegates to libs/atlas-constants | DONE | commit 7d8cd3224; `libs/atlas-wz/charparts/extract.go:22` imports `atlas-constants/item`, `:97-101` references `item.ClassificationFaceAccessory`; `libs/atlas-wz/go.mod:6` requires `Chronicle20/atlas/libs/atlas-constants v0.0.0-20260522184656-...` |
| 12 | F17 — concurrent `Properties()` regression test | DONE | commit b89d223cc; `libs/atlas-wz/wz/parse_race_test.go:105-138` `TestPropertiesConcurrentParse` spawns 16 goroutines hitting siblings, t.Skips when `testdata/concurrent.wz` absent (operator-side fixture per F17 carve-out) |
| 13 | F14 — wzinput status uses `server.MarshalResponse[Status]` | DONE | commit ea3af58a4; `services/atlas-data/atlas.com/data/wzinput/status.go:80` calls `server.MarshalResponse[Status](d.Logger())(w)(c.ServerInformation())(queryParams)(...)` replacing the prior manual envelope |
| 14 | F13 — delete dead `processData` orphan | DONE | commit f031b8253; `services/atlas-data/atlas.com/data/data/resource.go` net -36 lines, only the legacy reference comment at `:19` remains noting it was removed |
| 15 | F12 — comment wzinput PATCH multipart bypass | DONE | commit 325a3c47b; `services/atlas-data/atlas.com/data/wzinput/resource.go:15-22` block comment explaining why PATCH bypasses the generic `RegisterInputHandler[T]` path (binary multipart body) |
| 16 | F9 — Recreate-strategy cutover runbook | DONE | commit 471c16b0a; `docs/deploy/runbooks/recreate-strategy-cutover.md` (68 lines) |
| 17 | F10 — stale layer-png cleanup runbook | DONE | commit 845f1112c; `docs/deploy/runbooks/clean-stale-layer-pngs.md` (47 lines) |
| 18 | F11 — triage 359 no-bounds maps | DONE (placeholder, expected) | commit 27ce98358; `tools/triage-no-bounds.sh` (executable, 88 lines), `docs/tasks/task-076-task071-followups/no-bounds-triage.json` committed with `"status": "pending-operator-run"` — the plan explicitly designates this as operator-side off-CI follow-up; placeholder is the agreed deliverable shape |
| 19 | F19 — atlas-renders to docker-compose.core.yml | DONE | commit bf543721b; `deploy/compose/docker-compose.core.yml:585-606` adds the atlas-renders service block (build target, image tag, container name) |
| 20 | F20 — Henesys portal duplication fix | DONE | commit cb8930992; `libs/atlas-wz/mapimage/layers.go:297-329` `extractPortals` dedups by composite key with `seen` map, repro doc + `layers_portal_test.go` regression added |
| 21 | Branch-level acceptance gate | DONE (operator-reported) | User reports race/vet/build clean per affected module, `docker buildx bake atlas-{data,renders,character-factory}` clean, `routes_nginxt.sh` passes (including F18 drift block), `kustomize build` clean for base + main overlay. Operator-side gate; not re-run in this audit |

**Completion Rate:** 21/21 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. F11's `no-bounds-triage.json` ships as a committed placeholder with `status: "pending-operator-run"`, which matches the plan's explicit carve-out ("Run it against a reachable env" is operator-side per the plan body, Steps 2b and 3). The script (`tools/triage-no-bounds.sh`) is real and runnable; the artifact populates when an operator with cluster access executes it.

## Build & Test Results

Branch-level acceptance gates (Task 21) were executed locally by the operator prior to this audit and reported clean:

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-data | PASS | PASS | go test -race / vet / build clean; `docker buildx bake atlas-data` clean (operator-reported) |
| atlas-renders | PASS | PASS | same; `docker buildx bake atlas-renders` clean (operator-reported) |
| atlas-character-factory | PASS | PASS | same; bake clean (operator-reported, transitive importer of libs/atlas-wz) |
| libs/atlas-wz | PASS | PASS | go test -race / vet / build clean (operator-reported) |
| deploy (k8s + compose) | PASS | PASS | `kustomize build` clean for base + main overlay; `deploy/shared/test/routes_nginxt.sh` passes including the new F18 drift block (operator-reported) |

This audit did not re-execute these gates; it verified the diff and file shapes match the plan's acceptance criteria.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

The branch matches the plan one-to-one. Every followup (F1-F20) is a discrete commit with the file changes the plan called out, each commit's subject line includes the followup tag, and the final branch-level gate (Task 21) passed locally. The F11 placeholder is the intended deliverable, not a partial implementation. Recommend proceeding to PR.

## Action Items

None required for plan adherence. Two non-blocking observations:

1. F17's `testdata/concurrent.wz` is intentionally absent (skipped fixture, documented). If/when an operator-side fixture is added, the test will exercise rather than skip — track this if regression coverage in CI is desired.
2. F11's triage JSON remains in the `pending-operator-run` state. Once an operator runs `tools/triage-no-bounds.sh` against a reachable cluster, file the per-cluster follow-up tasks the plan mentions in Step 3 and update the artifact.

---

# Backend Guidelines Audit — task-076-task071-followups

- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/`
- **Audit Date:** 2026-05-22
- **Branch:** task-076-task071-followups
- **Base Branch:** main
- **Auditor:** backend-guidelines-reviewer (adversarial mode)

## Scope

Audited the Go files changed in this branch:

```
git diff --name-only main..HEAD -- '*.go'   # 33 files
```

The changed packages are all **support / infrastructure packages** (baseline publish/restore, commodity registry, wzinput streaming, scope cache, wz parsing libs, kafka consumer-group resolver, atlas-channel/atlas-login main wiring). None contain a `model.go` (full DDD domain) or a JSON:API REST surface that would meaningfully exercise the DOM-* domain checklist. The targeted checks below are the items the request explicitly asked for plus the cross-cutting rules that apply (FieldLogger, error handling, `RegisterInputHandler[T]` for POST/PATCH, DOM-21 single-source constants, build/test gates).

## Phase 1 — Build & Test Gates

| Module | `go build ./...` | `go vet ./...` | `go test -race -count=1 ./...` |
|---|---|---|---|
| `libs/atlas-wz` | PASS | PASS | PASS (wz, mapimage, icons, charparts all green) |
| `libs/atlas-kafka` | PASS | PASS | PASS (consumergroup green) |
| `services/atlas-data/atlas.com/data` | PASS | PASS | PASS (baseline, commodity, wzinput, data/workers, data/wztoxml green; `TestPublishErrorIsContextualized`, `TestRegisterCommodityDoesNotWrapInOuterTx`, `TestRestoreDeferredMarkerStructure`, `TestResolveCacheGate*`, `TestExtractPortalsDeduplicates`, `TestPropertiesFastPathSkipsLock`, `TestLockParseIsExclusive` all pass) |
| `services/atlas-renders/atlas.com/renders` | PASS | PASS | PASS (storage green with new scope_test.go) |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS |
| `services/atlas-login/atlas.com/login` | PASS | PASS (one pre-existing `WaitGroup.Add called from inside new goroutine` warning at `socket/init.go:39`, unrelated to this branch's diff — confirmed via `git diff main..HEAD -- services/atlas-login/atlas.com/login/socket/init.go` returns empty) | PASS |

Phase 1 PASS overall. No gate failures.

## Phase 2 — Domain Discovery

The 33 changed Go files map to these packages:

- `libs/atlas-wz/{charparts, icons, mapimage, wz}` — pure library packages (no model.go, no REST, no Kafka).
- `libs/atlas-kafka/consumergroup` — pure library helper.
- `services/atlas-data/atlas.com/data/baseline` — REST + storage transport for baseline publish/restore (no `model.go`).
- `services/atlas-data/atlas.com/data/commodity` — registry wrapper around `document.Storage` (no `model.go`).
- `services/atlas-data/atlas.com/data/wzinput` — REST + MinIO streaming (no `model.go`).
- `services/atlas-data/atlas.com/data/data` — REST aggregator (no `model.go`).
- `services/atlas-data/atlas.com/data/data/workers/*` — ingest workers (no `model.go`).
- `services/atlas-data/atlas.com/data/data/wztoxml` — XML adapter (no `model.go`).
- `services/atlas-renders/atlas.com/renders/storage` — MinIO/cache helpers (no `model.go`).
- `services/atlas-channel/atlas.com/channel/main.go`, `services/atlas-login/atlas.com/login/main.go` — service bootstrap.

Classification: **all touched packages are Support packages**. Full DOM-* / SUB-* checklists do not apply mechanically; relevant cross-cutting checks are listed below.

## Phase 3 — Targeted Checks

### Specifically requested items

| Item | Check | Status | Evidence |
|------|-------|--------|----------|
| F16 / DOM-21 — `accessoryPartClassFor` delegates to `libs/atlas-constants` | Verify the package-private classifier in `libs/atlas-wz/charparts/extract.go` uses `item.Classification*` constants and that no other touched package re-hardcodes 101/102/103. | PASS | `libs/atlas-wz/charparts/extract.go:99-109` switches on `item.Classification(id / 10000)` and references `item.ClassificationFaceAccessory` / `item.ClassificationEyeAccessory` / `item.ClassificationEarring`. The constants are defined at `libs/atlas-constants/item/constants.go:14-16`. Cross-search for raw `/ 10000` in changed services returns only `services/atlas-renders/atlas.com/renders/character/composite.go:72` (a comment) and `libs/atlas-wz/charparts/extract.go:100` (the canonical use); the `composite.go` body at `:78-116` already switches on `item.GetClassification` and `item.Classification*` constants — same single-source pattern. `services/atlas-data/atlas.com/data/mobskill/rest.go:47` uses `id / 10000` to derive a skill id, which is a different domain concept (skill-id-from-mobskill-id, not item classification) and is not in this branch's diff. **DOM-21 holds across the diff.** |
| F6 — `Image.Properties()` error surface | Verify every caller of `img.Properties()` in the workspace handles the new error return (no `_, _ :=` swallowing). | PASS | New signature at `libs/atlas-wz/wz/image.go:60` is `func (i *Image) Properties() ([]property.Property, error)`. Cross-search `grep -rn "\.Properties()" --include="*.go" \| grep -v _test.go` yields 24 hits; the non-party-quests, non-test callers (17) are: `libs/atlas-wz/icons/extract.go:53,125,176,209` (best-effort `continue` per plan), `libs/atlas-wz/charparts/{extract.go:241, smap.go:40}`, `libs/atlas-wz/mapimage/{layers.go:48, decoder.go:80,120,145, minimap.go:18,59}`, `services/atlas-data/atlas.com/data/data/workers/{ui.go:44, item.go:77, skill.go:64}`, `services/atlas-data/atlas.com/data/data/wztoxml/adapter.go:76`. Every site assigns into `(props, err)` and either returns the wrapped error (`fmt.Errorf("…: %w", err)`) or applies the documented best-effort skip semantics. `grep -rn "_, _ :=.*Properties" --include="*.go"` returns zero non-test hits. The party-quests callers (`stage.Properties()`, `Bonus().Properties()`) are different types (not `*wz.Image`); confirmed by reading `services/atlas-party-quests/.../processor.go:450,593,666,1380,1423` — they call `stage.Stage.Properties()` (atlas-party-quests local type), not the wz lib. |
| F1 — publish.go nil-deps + tempfile | Confirm nil-deps guards return `publish: nil-*` errors; tempfile is used with `defer os.Remove + defer Close`; each step is wrapped with `publish: <step>:`. | PASS | `services/atlas-data/atlas.com/data/baseline/publish.go:39-44` returns `fmt.Errorf("publish: nil-db")` and `"publish: nil-mc"`. `:47-52` `os.CreateTemp` + `defer os.Remove(tmp.Name())` + `defer tmp.Close()`. Every step uses `fmt.Errorf("publish: <step>: %w", err)`: `create-tempfile` (:49), `marshal-header` (:67), `write-header` (:70), `dump-table %s` (:75), `close-tar` (:79), `seek-end` (:84), `seek-start` (:87), `put-tar` (:92), `put-sha` (:97). `io.MultiWriter(tmp, h)` (:55) drives sha and tempfile in one pass; `tmp.Seek` rewinds for upload. Test `TestPublishErrorIsContextualized` (publish_test.go:15-24) pins the prefix assertion. |
| F5 — `restore.go` cleanup-after-failure best-effort + original error returned | Confirm `cleanupAfterFailure` only logs (does not return error) and the original `restoreOneTable` / `ANALYZE` error is returned to the caller. | PASS | `services/atlas-data/atlas.com/data/baseline/restore.go:75-81` defines `cleanupAfterFailure(ctx, l, db, target)` returning nothing; each cleanup DELETE wraps in `l.WithError(err).Warnf("…best-effort")` and never propagates. Call sites at `:145-149` (table-loop failure) and `:151-156` (ANALYZE failure) both invoke `cleanupAfterFailure(ctx, r.L, r.DB, target)` then `return err` (the original error). The tenant_baselines UPSERT at `:159-166` runs only after both loops complete successfully — `TestRestoreDeferredMarkerStructure` (restore_failure_test.go:14-34) pins that ordering. |
| F2 — commodity/processor.go drops outer tx; `Storage.Add` is per-row | Confirm processor.go has no `database.ExecuteTransaction` wrapping `Register`, and `document.Storage[].Add` (or its delegate) commits per row. | PASS | `services/atlas-data/atlas.com/data/commodity/processor.go:24-39` `Register` iterates `ms` and calls `s.Add(ctx)(m)()` once per row with no enclosing transaction; comment `:17-23` documents the change. `TestRegisterCommodityDoesNotWrapInOuterTx` (processor_test.go:15-23) pins the absence of `database.ExecuteTransaction` in the file. Per-row commit semantics confirmed: `services/atlas-data/atlas.com/data/document/storage.go:95-108` `Storage[I,M].Add` calls `s.dbSto.Add(ctx)(m)()` (which at `db_storage.go:105` wraps a single row in `database.ExecuteTransaction`) and `s.regSto.Add` for the in-memory registry. Each call is one transaction; a mid-loop failure preserves successfully-committed rows so a retry can converge. |
| F14 — wzinput Status JSON:API interface | Confirm Status implements the full MarshalIdentifier/UnmarshalIdentifier set so api2go can decode relationship-bearing payloads. | PASS | `services/atlas-data/atlas.com/data/wzinput/status.go:24-56` implements `GetName()` (returns `"wzInputStatus"`), `GetID()` (returns `"current"`), `SetID(string) error`, `GetReferences()`, `GetReferencedIDs()`, `GetReferencedStructs()`, `SetToOneReferenceID(_, _ string) error`, `SetToManyReferenceIDs(_ string, _ []string) error`, `SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error`. The set matches the canonical `services/atlas-data/atlas.com/data/data/status.go:17-58` `StatusRestModel` 1:1. The handler at `:80-82` calls `server.MarshalResponse[Status](d.Logger())(w)(c.ServerInformation())(queryParams)(...)` (`status.go:80`). |

### Cross-cutting checks that apply to these support packages

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| BACK-LOG | Processor / publisher / restorer accept `logrus.FieldLogger`, not `*logrus.Logger` | PASS | `baseline/publish.go:26` `L logrus.FieldLogger`; `baseline/restore.go:35` `L logrus.FieldLogger`; `commodity/processor.go:13` `NewStorage(l logrus.FieldLogger, …)`. No `*logrus.Logger` in changed files. |
| BACK-ERR | New error paths use `fmt.Errorf("…: %w", err)` wrapping | PASS | publish.go (9 wrapped errors), restore.go (`runRestoreTables`, `restoreOneTable`, `cleanupAfterFailure` log), workers (`item.go:79`, `skill.go:66`, `ui.go:46`, `wztoxml/adapter.go:78,82` all `%w`). |
| BACK-POST | POST/PATCH handlers either use `RegisterInputHandler[T]` or have a documented exception | PASS | `baseline/handler.go:23,24` POSTs both use `rest.RegisterInputHandler[PublishInputModel]` / `[RestoreInputModel]`. `wzinput/resource.go:21-26` is a documented PATCH bypass for binary multipart streaming (F12 plan comment explicitly explains why `RegisterInputHandler[T]` would consume the body as JSON and fail). `data/resource.go:21` is GET-only. |
| BACK-ENV | No new `os.Getenv` reads inside handlers | PASS | `grep -n "os.Getenv" services/atlas-data/atlas.com/data/{baseline,commodity,wzinput}/*.go services/atlas-renders/atlas.com/renders/storage/*.go` returns only `services/atlas-renders/atlas.com/renders/storage/config.go` (pre-existing config loader, not in this diff). |
| BACK-CONST | DOM-21 single-source for atlas-constants | PASS | See F16 row above. Only one ratification path was identified (`accessoryPartClassFor`); the rest of the diff doesn't define new id/classification helpers. `libs/atlas-wz/go.mod` now requires `Chronicle20/atlas/libs/atlas-constants`, confirmed by `go.work.sum` update. |
| BACK-RACE | Race detector clean on every changed package | PASS | `go test -race -count=1 ./...` clean in `libs/atlas-wz`, `libs/atlas-kafka`, `services/atlas-data/atlas.com/data`, `services/atlas-renders/atlas.com/renders`, `services/atlas-channel/atlas.com/channel`, `services/atlas-login/atlas.com/login`. |
| BACK-CONCURRENCY | F4 — wz seek-path concurrency invariants documented | PASS | `libs/atlas-wz/wz/file.go:222-228` annotates `tryParseWithVersion` as Open-only / single-threaded. `libs/atlas-wz/wz/image.go:53-59` documents `Image.parse()` / `parseMu` invariant. `libs/atlas-wz/wz/image.go:117-121` `parsePropertyList` "caller holds wz.parseMu" invariant comment. Production test `TestLockParseIsExclusive` (parse_race_test.go:29-63) and `TestPropertiesFastPathSkipsLock` (`:77-92`) pin the contract. |
| BACK-RESOLVE | `consumergroup.Resolve` simplification — call-site sprintf | PASS | `libs/atlas-kafka/consumergroup/resolver.go:14-22` returns env-value-or-default verbatim; `services/atlas-channel/atlas.com/channel/main.go:151` and `services/atlas-login/atlas.com/login/main.go:66` call `consumergroup.Resolve(fmt.Sprintf(template, config.Id.String()))`. All 50+ other `consumergroup.Resolve(…)` callers (grep across `services/`) pass a single literal string and are source-compatible. |

## Phase 4 — Security Review

Not applicable. None of the changed packages handle auth, tokens, or open-redirect flows. The branch does not touch `services/atlas-login/atlas.com/login` security surfaces (only the `consumergroup.Resolve(...)` wiring in `main.go:66`).

## Summary

### Blocking (must fix)

None.

### Non-Blocking (should fix)

None identified in the diff itself. (The two non-blocking observations from the plan-adherence section above remain: F17 fixture is operator-side; F11 triage JSON awaits operator run.)

## Overall Status

- **Build:** PASS (all changed modules)
- **Tests:** PASS (all changed modules; race detector clean)
- **DOM-21 / single-source for atlas-constants:** PASS
- **F1 / F2 / F5 / F6 / F14 / F16 (specifically requested):** PASS (file:line evidence above)
- **Overall:** **PASS**

The diff is mechanically clean against the backend developer guidelines for the subset of rules that apply to support packages. Full DDD DOM-* / SUB-* checks are not exercised because the changed packages have no domain models — they are pure infrastructure (publish/restore, registry wiring, wz parsing, scope cache, service main wiring).

