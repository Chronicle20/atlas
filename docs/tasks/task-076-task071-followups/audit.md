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

---

# Re-audit after rebase — 2026-05-27

**Re-audit Date:** 2026-05-27
**Branch:** `task-076-task071-followups`
**Base Branch:** `main` (`origin/main` @ `f52ba11e9`)
**HEAD:** `4b266eef7`
**Span:** 24 commits ahead / 0 behind main (`git log --oneline f52ba11e9..HEAD` = 24 lines).
**Trigger:** branch was rebased onto current `origin/main` (18 newer main commits since the prior audit's base); one conflict in `deploy/k8s/overlays/main/kustomization.yaml` (F7) was resolved manually.

## Re-audit summary

Twenty of twenty-one plan items remain faithfully implemented post-rebase. The single drift is on the F7 rebase resolution: the operator note said the conflict was resolved by "keeping main's `main-6af3cb0` tag for atlas-reactors and adding atlas-renders at the same tag," but `origin/main` is actually at `main-324205c` (`f52ba11e9 chore(images): bump main overlay to 324205c (#600)`). The branch now pins **all** services (atlas-renders included) to `main-6af3cb0`, regressing every other service's pin by one bump cycle (`995fd2e63` → `f52ba11e9`). Build/test gates on the touched Go modules are green; kustomize, routes_nginxt drift check, and the F8 generator-output match are all clean.

## Per-item post-rebase verification

| # | Item | Plan-mapped file | Post-rebase state | Verdict |
|---|------|------------------|-------------------|---------|
| 1 | F1 — publish.go tempfile + step prefixes | `services/atlas-data/atlas.com/data/baseline/publish.go` | All 9 wrapped errors at `:40-97` with `publish:` prefix; `os.CreateTemp` at `:47`, `io.MultiWriter(tmp, h)` at `:55` (mapped via `grep "publish:"`). Module `go build ./... && go vet ./... && go test -race ./baseline` PASS. | DONE |
| 2 | F7 — pin atlas-renders in main overlay | `deploy/k8s/overlays/main/kustomization.yaml:278-279` | Pin present, alphabetically correct between atlas-reactors (`:276-277`) and atlas-saga-orchestrator (`:280-281`). **However:** tag is `main-6af3cb0` for every entry while `origin/main:deploy/k8s/overlays/main/kustomization.yaml` already advanced to `main-324205c` at commit `f52ba11e9` (the stated BASE_SHA). The rebase resolution applied a stale tag across all 57 pins. The plan's correctness condition ("use the same SHA as the other pins") still holds within the branch — but every pin is now out of date with main. | DONE (with rebase-resolution drift; see "Drift from plan due to rebase" below) |
| 3 | F3 — stop caching negative scope verdicts | `services/atlas-renders/atlas.com/renders/storage/scope.go:30`, `:48`; `smap.go:68`, `:78` | `Caches.Scope.Add` only inside the positive `has` branch (`scope.go:30`); negative path returns `"shared"` uncached (`:48`, `:78`). `go test -race ./storage` PASS. | DONE |
| 4 | F2 — drop outer ExecuteTransaction in commodity register | `services/atlas-data/atlas.com/data/commodity/processor.go:32` | Body loops `s.Add(ctx)(m)()` with no `database.ExecuteTransaction` wrapper; the only `ExecuteTransaction` reference is the explanatory comment at `:21`. Test `TestRegisterCommodityDoesNotWrapInOuterTx` PASS. | DONE |
| 5 | F5 — two-phase finalize baseline restore | `services/atlas-data/atlas.com/data/baseline/restore.go:41`, `:75`, `:145-160` | `runRestoreTables` at `:41`, `cleanupAfterFailure` at `:75` (best-effort log-only), `tenant_baselines` UPSERT at `:160` lives after both gates. `TestRestoreDeferredMarkerStructure` PASS. | DONE |
| 6 | F8 — dedupe routes-config via kustomize configMapGenerator | `tools/gen-routes.sh`, `deploy/k8s/base/atlas-ingress.yaml`, `deploy/k8s/base/kustomization.yaml`, `deploy/k8s/base/routes.conf.template.generated` | Generator runs to identical output (`git diff --stat -- deploy/k8s/base/routes.conf.template.generated` empty after `bash tools/gen-routes.sh`). `kustomize build deploy/k8s/base` PASS. | DONE |
| 7 | F18 — routes drift validation in CI | `deploy/shared/test/routes_nginxt.sh` | Script's final block re-runs the generator and `git diff --quiet`s the output. End-to-end run reports `routes drift check (shared vs k8s-generated): OK`. | DONE |
| 8 | F6 — `Properties()` returns `([]Property, error)` | `libs/atlas-wz/wz/image.go:60` + 16 callers across the monorepo | Signature confirmed at `libs/atlas-wz/wz/image.go:60`. All 16 non-test callers updated: `libs/atlas-wz/{charparts/{extract.go:241, smap.go:40}, icons/extract.go:53,125,176, mapimage/{decoder.go:80,120,145, layers.go:48, minimap.go:18,59}}` + `services/atlas-data/atlas.com/data/data/{workers/{ui.go:44, item.go:77, skill.go:64}, wztoxml/adapter.go:76}`. `libs/atlas-wz/go test -race ./...` PASS across 11 packages. | DONE |
| 9 | F4 — wz seek-path concurrency annotations | `libs/atlas-wz/wz/file.go`, `libs/atlas-wz/wz/image.go`, `services/atlas-data/atlas.com/data/data/workers/runtime.go` | Comments present per plan. Race tests PASS. | DONE |
| 10 | F15 — layout-common helper | `libs/atlas-wz/mapimage/layers.go` | Helper extracted; `mapimage` race tests PASS. | DONE |
| 11 | F16 — `accessoryPartClassFor` → libs/atlas-constants | `libs/atlas-wz/charparts/extract.go`, `libs/atlas-wz/go.mod` | `go.mod` imports `atlas-constants` (verified by `libs/atlas-wz` build pulling the lib transitively). `charparts` race tests PASS. | DONE |
| 12 | F17 — concurrent `Properties()` regression test | `libs/atlas-wz/wz/parse_race_test.go` + (carve-out: absent fixture) | Test in place; `libs/atlas-wz/wz/testdata/` directory does not exist (`ls libs/atlas-wz/wz/testdata/` → "no testdata dir"). Test t.Skips per the operator-side carve-out. **Carve-out shape unchanged post-rebase.** | DONE (carve-out preserved) |
| 13 | F14 — wzinput status uses `server.MarshalResponse[Status]` | `services/atlas-data/atlas.com/data/wzinput/status.go` | Handler module builds + tests clean. | DONE |
| 14 | F13 — delete `processData` orphan | `services/atlas-data/atlas.com/data/data/resource.go` | File is -36 lines net per the diff stat; module builds clean. | DONE |
| 15 | F12 — wzinput PATCH multipart bypass comment | `services/atlas-data/atlas.com/data/wzinput/resource.go` | +6 lines per the diff stat; module builds clean. | DONE |
| 16 | F9 — Recreate cutover runbook | `docs/deploy/runbooks/recreate-strategy-cutover.md` | 68 lines committed, unchanged by rebase (no main churn under `docs/deploy/runbooks/`). | DONE |
| 17 | F10 — stale layer-png cleanup runbook | `docs/deploy/runbooks/clean-stale-layer-pngs.md` | 47 lines committed, unchanged by rebase. | DONE |
| 18 | F11 — triage 359 no-bounds maps | `tools/triage-no-bounds.sh`, `docs/tasks/task-076-task071-followups/no-bounds-triage.json` | Script (88 lines) and JSON placeholder still present. Placeholder reads `"status": "pending-operator-run"`, `reachable: []`, `unreachable: []`, `counts: {reachable: 0, unreachable: 0, total: 0}` — matches the agreed operator-side carve-out shape. **Unchanged by rebase.** | DONE (carve-out preserved) |
| 19 | F19 — atlas-renders in docker-compose.core.yml | `deploy/compose/docker-compose.core.yml:585-606` | Block present, alphabetically between atlas-reactors (`:573`) and atlas-saga-orchestrator (`:607`). | DONE |
| 20 | F20 — Henesys portal dedup | `libs/atlas-wz/mapimage/layers.go:297-329` + `layers_portal_test.go` | `extractPortals` at `:302` uses `seen` map for dedup (`:307`). `libs/atlas-wz/mapimage` race tests PASS. | DONE |
| 21 | Branch-level acceptance gate | Module-level gates + kustomize + routes_nginxt | Re-run from this audit (see "Build & test re-verification" below). All PASS. | DONE |

**Re-audit completion rate:** 21/21 (100%, identical to prior audit).

## Build & test re-verification (executed by this audit)

| Module | `go build ./...` | `go vet ./...` | `go test -race -count=1 ./...` |
|---|---|---|---|
| `libs/atlas-wz` | PASS | PASS | PASS (all 11 packages: `atlas`, `atlas/pngenc`, `canvas`, `charparts`, `crypto`, `icons`, `manifest`, `mapimage`, `maplayout`, `wz`, `wz/property`) |
| `services/atlas-data/atlas.com/data` | PASS | PASS | PASS (re-tested `./baseline ./commodity ./wzinput ./data/workers ./data/wztoxml`) |
| `services/atlas-renders/atlas.com/renders` | PASS | PASS | PASS (`./storage` re-run) |
| `services/atlas-character-factory/atlas.com/character-factory` | PASS | — | — (transitive importer of `libs/atlas-wz`; build only) |

| Deploy check | Result |
|---|---|
| `kustomize build deploy/k8s/base` | PASS |
| `kustomize build deploy/k8s/overlays/main` | PASS — rendered `atlas-renders` deployment image: `ghcr.io/chronicle20/atlas-renders/atlas-renders:main-6af3cb0` |
| `bash deploy/shared/test/routes_nginxt.sh` | PASS — `nginx -t`, MinIO cross-ns, atlas-renders header, F18 drift all OK |
| `bash tools/gen-routes.sh` (regenerate F8 output) | PASS — `git diff --stat -- deploy/k8s/base/routes.conf.template.generated` empty (committed file matches generator output exactly) |

## Drift from plan due to rebase

**One drift, narrowly scoped to F7's conflict resolution.**

`deploy/k8s/overlays/main/kustomization.yaml` now pins **every** service (all 57 entries) at `newTag: main-6af3cb0`. Reference: `grep "newTag" deploy/k8s/overlays/main/kustomization.yaml | sort -u` returns the single value `main-6af3cb0`. `origin/main`'s same file at base `f52ba11e9` pins all entries at `main-324205c` (commit `f52ba11e9 chore(images): bump main overlay to 324205c (#600)`). `main-6af3cb0` is the **prior** bump (`995fd2e63 chore(images): bump main overlay to 6af3cb0 (#595)`), superseded on main by `f52ba11e9`.

Concretely:
- The merged-on-main `atlas-reactors` pin was advanced to `main-324205c` at `f52ba11e9`. The rebase kept the branch-side pre-bump tag `main-6af3cb0` instead.
- The same regression applied to every other pin in the file (all 57 services), because the conflict resolution preferentially kept the branch side of the entire `images:` block rather than just the atlas-renders insertion.
- The atlas-renders insertion itself (lines 278-279) is correct in shape and position; only the tag value is wrong.

Plan adherence consequence: F7 is structurally done (the pin exists alphabetically positioned). Operationally, this PR will, on merge, **revert main's most recent image bump** for every pinned service. Whether that's acceptable depends on whether renovate's next sweep will catch it back up — but the assertion in the operator note ("kept main's `main-6af3cb0` tag") is factually incorrect (main is at `main-324205c`, not `main-6af3cb0`).

**Other rebase-touched files (`deploy/k8s/base/atlas-ingress.yaml`, `deploy/shared/routes.conf`, `deploy/shared/test/routes_nginxt.sh`):** spot-checked. No drift — F8 generator output is identical to the committed `routes.conf.template.generated`, F18 drift check passes against current source, F7 atlas-renders entry is in the canonical position.

## Carve-outs (verified preserved post-rebase)

| Carve-out | File | Post-rebase state |
|---|---|---|
| F11 — operator-pending triage JSON | `docs/tasks/task-076-task071-followups/no-bounds-triage.json` | `status: "pending-operator-run"`, all counts zero, note refers operator to script header — matches the agreed shape exactly. |
| F17 — absent concurrent fixture | `libs/atlas-wz/wz/testdata/` | Directory absent. Test `TestPropertiesConcurrentParse` (`libs/atlas-wz/wz/parse_race_test.go`) t.Skips with the documented message. CI remains green. |

## Spot-check of existing audit.md claims (2026-05-22 → 2026-05-27)

The earlier audit's commit-SHA references are stale because the rebase rewrote commit SHAs (e.g., audit cites "commit 952e22193" for F1; post-rebase the F1 commit is `15e64aa63`). Every cited **file-path:line** reference in the prior audit still resolves correctly though — re-checked:

- F1 publish.go: prior audit cites `:46-100` → re-verified at `publish.go:40-97` (4-line shift; the `nil-db` / `nil-mc` guards now live at `:40-44`, which the prior audit's "every error wrapped" claim still covers).
- F7 kustomization line citations (276-281): structurally identical post-rebase. Tag value drifted (see above).
- F3 scope.go cache add at `:30`, `:48`; smap.go `:62-78`: confirmed at `scope.go:30`/`:48` and `smap.go:68`/`:78` — minor 4-line shifts within smap.go but the gate condition is intact.
- F5 restore.go `runRestoreTables` / `cleanupAfterFailure` / UPSERT positioning: confirmed at `:41` / `:75` / `:160`.
- F8/F18 file layout: confirmed; generator-output match holds.
- F6 Properties() callers (16 non-test sites): re-enumerated via `grep -rn "\.Properties()" --include="*.go" libs/atlas-wz services/atlas-data | grep -v _test.go` — 16 matches, all assigning into `(props, err)` or `(root, err)`.

Every functional claim from the earlier audit holds. Commit SHAs are stale; file evidence is current.

## Overall re-audit assessment

- **Plan adherence:** FULL (21/21)
- **Build/test:** PASS on every re-run module + deploy check
- **Carve-outs:** Both preserved
- **Rebase drift:** One narrow drift, F7 tag value — does not affect plan correctness but does affect merge-day operational behaviour

**Recommendation:** **NEEDS_REVIEW** — not because the plan was unfaithfully implemented, but because the F7 rebase resolution applied a stale image tag (`main-6af3cb0`) across all 57 pinned services when main has already advanced to `main-324205c`. Merging this PR as-is will revert the most recent image bump on main.

## Action items (re-audit)

1. **F7 image tag re-resolution.** Reset the atlas-renders pin (and every other service pin in `deploy/k8s/overlays/main/kustomization.yaml`) to match `origin/main`'s current `main-324205c`, then re-validate `kustomize build deploy/k8s/overlays/main`. The cleanest path from the worktree root (`<repo-root>/.worktrees/task-076-task071-followups`):
   ```bash
   git checkout origin/main -- deploy/k8s/overlays/main/kustomization.yaml
   # Re-insert ONLY the atlas-renders pin (2 lines) at the alphabetical position
   #   between atlas-reactors (currently :276-277) and atlas-saga-orchestrator (:278-279 on main).
   #   Use the same `main-324205c` tag as its alphabetical neighbours.
   git commit -m "fix(deploy): F7 align atlas-renders pin with current main bump"
   ```
   Confirm with `grep "newTag" deploy/k8s/overlays/main/kustomization.yaml | sort -u` — expect a single value matching main's current image tag.

2. **Operator's note** that the conflict was resolved by "keeping main's `main-6af3cb0` tag" should be revisited — main is at `main-324205c` as of the stated BASE_SHA. The 18-commit rebase span includes `f52ba11e9 chore(images): bump main overlay to 324205c (#600)`, which is the bump that was lost.

3. Once (1) lands, this branch is READY_TO_MERGE — every other plan item survived the rebase intact.

---

## Backend re-audit after rebase (2026-05-27)

Re-audit triggered by 24-commit rebase onto `origin/main` (BASE_SHA `f52ba11e9`, HEAD_SHA `4b266eef7`). Scope: all 29 Go files in the diff plus three changed modules (`libs/atlas-wz`, `services/atlas-data`, `services/atlas-renders`).

### Phase 1 — Build / Vet / Race Tests

| Module | `go build ./...` | `go vet ./...` | `go test -race -count=1 ./...` |
|---|---|---|---|
| `libs/atlas-wz` | PASS (no output) | PASS | PASS (11 packages, all `ok`; `parse_race_test.go:29,77,105` race-clean) |
| `services/atlas-data/atlas.com/data` | PASS | PASS | PASS (all packages `ok`; baseline, commodity, data/workers, data/wztoxml, wzinput all pass) |
| `services/atlas-renders/atlas.com/renders` | PASS | PASS | PASS (atlas-renders, character, mapr, storage all `ok`) |

`libs/atlas-kafka` was not touched (`git diff --name-only origin/main..HEAD -- libs/atlas-kafka/` is empty). The previous audit covered it; nothing to re-verify.

Only `libs/atlas-wz/go.mod` was modified — it gains a direct require on `github.com/Chronicle20/atlas/libs/atlas-constants v0.0.0-20260522184656-55cd21714da6` (`libs/atlas-wz/go.mod:5-9`), which is the F16 dependency.

### Phase 2 — Domain Discovery (re-audit)

No `model.go` exists in any changed package, so the full DOM checklist is not applicable. Re-confirmed:

- **Support packages (no `resource.go`, no `model.go`):** `libs/atlas-wz/{charparts,icons,mapimage,wz}`, `services/atlas-data/atlas.com/data/data/workers`, `services/atlas-data/atlas.com/data/data/wztoxml`, `services/atlas-renders/atlas.com/renders/storage`. SUB-* and DOM-* checks not applicable; SEC and project-wide guidelines (DOM-21, error handling, concurrency) apply.
- **Sub-domain (has `resource.go` only):** `services/atlas-data/atlas.com/data/commodity` (`resource.go:18-26`), `services/atlas-data/atlas.com/data/data` (`resource.go:14-22`), `services/atlas-data/atlas.com/data/wzinput` (`resource.go:17-28`).
- **`baseline`** uses `handler.go` rather than `resource.go` — runs SUB anti-pattern checks anyway.

### Phase 3 — Per-package re-audit

#### F6 — `Image.Properties()` signature `[]property.Property` → `([]property.Property, error)`

| Check | Status | Evidence |
|---|---|---|
| New signature declared | PASS | `libs/atlas-wz/wz/image.go:60` |
| Every caller threads the error | PASS | All 17 call sites in `libs/atlas-wz/{charparts,icons,mapimage}/*`, `services/atlas-data/atlas.com/data/data/{workers,wztoxml}/*` use `props, err := img.Properties()` and check err. Confirmed by `grep -rnE '\.Properties\(\)' libs/atlas-wz/ services/atlas-data/ services/atlas-renders/` — no caller drops err. |
| External callers in other services unaffected | PASS | `services/atlas-party-quests/.../processor.go:593,1380,1423` call `.Properties()` on a different non-`wz.Image` type; `go build ./...` in that service is clean. |
| Race detector clean against the change | PASS | `parse_race_test.go:29,77,105` (LockParse exclusive, Properties fast-path skips lock, concurrent parse against fixture) — all green under `go test -race`. |

#### F16 — DOM-21 atlas-constants single-source

| Check | Status | Evidence |
|---|---|---|
| `accessoryPartClassFor` imports atlas-constants | PASS | `libs/atlas-wz/charparts/extract.go:22` imports `github.com/Chronicle20/atlas/libs/atlas-constants/item`. |
| Uses `item.Classification(...)` not a local type | PASS | `libs/atlas-wz/charparts/extract.go:100` — `switch item.Classification(id / 10000)`. |
| Uses shared constants not magic numbers | PASS | `libs/atlas-wz/charparts/extract.go:101-106` — `item.ClassificationFaceAccessory`, `item.ClassificationEyeAccessory`, `item.ClassificationEarring`. |
| Constants exist in atlas-constants | PASS | `libs/atlas-constants/item/constants.go:7,14-16` defines `type Classification uint32` and the three values used. |
| No new redeclarations elsewhere in the diff | PASS | `git diff origin/main..HEAD -- '*.go' \| grep '^+' \| grep -iE '/ 10000\|^\+type .* uint\|^\+const'` returns only the F16 usage above — no new local types or numeric ranges introduced. |

#### F4 — Concurrency invariants (`parseMu`)

| Check | Status | Evidence |
|---|---|---|
| `parseMu` documented as the single seek-serialiser | PASS | `libs/atlas-wz/wz/file.go:25-46` and `libs/atlas-wz/wz/file.go:225-227`. |
| `Image.Properties()` documents the lock + double-check pattern | PASS | `libs/atlas-wz/wz/image.go:43-59` (godoc), `image.go:60-80` (impl). The pre-lock read of `i.parsed` is a benign double-checked-locking fast path; the post-lock re-check at `image.go:70-72` makes it correct. `-race` on `parse_race_test.go` produces no flags. |
| Tests pin the contract | PASS | `parse_race_test.go:29` (LockParse exclusive), `parse_race_test.go:77` (fast path skips lock), `parse_race_test.go:105` (concurrent parse, fixture-gated). |
| Worker fetch path documented | PASS | `services/atlas-data/atlas.com/data/data/workers/runtime.go:114-122` documents that each `fetchArchive` call gets a fresh `*wz.File`, so `Open()` is single-threaded by construction. |

#### F1 — `baseline/publish.go` temp-file buffering + step-error wrapping

| Check | Status | Evidence |
|---|---|---|
| Buffers to a temp file before upload | PASS | `services/atlas-data/atlas.com/data/baseline/publish.go:47-55` — `os.CreateTemp` + `io.MultiWriter(tmp, h)`. |
| Temp file cleanup deferred (LIFO order correct) | PASS | `publish.go:51-52` — `defer os.Remove(tmp.Name())` then `defer tmp.Close()`; the LIFO order closes first, then removes. |
| Every step error wrapped with `publish: <step>:` | PASS | `publish.go:49,67,70,75,79,84,87,92,97` — nine explicit `fmt.Errorf("publish: <step>: %w", err)` sites; nil-dep guards at `:40,:43`. |
| Test pins the contract | PASS | `publish_test.go:15-24` asserts the `publish:` prefix on nil-dep paths. |
| `runCopyOut` SQL string composition safe | PASS | `publish.go:124-133` — `table` comes from `DumpTables` (package constant); `canonical.TenantUUID` is also a constant. `target` uses `?` placeholders elsewhere. No user input reaches the COPY statement. |

#### F5 — `baseline/restore.go` two-phase finalize

| Check | Status | Evidence |
|---|---|---|
| sha256 verified BEFORE any DB mutation | PASS | `restore.go:111-117` hashes the full body; `:114-117` returns `ErrShaMismatch` before any DELETE/COPY. |
| Schema version checked before mutation | PASS | `restore.go:135-137` returns `ErrSchemaMismatch` before runRestoreTables. |
| Table loop pulled into `runRestoreTables` | PASS | `restore.go:41-58`. |
| `cleanupAfterFailure` invoked on both table-loop and ANALYZE failures | PASS | `restore.go:146-148` (table loop), `restore.go:152-156` (ANALYZE). |
| Marker UPSERT deferred until both succeed | PASS | `restore.go:159-166` — runs after the ANALYZE loop. Verified structurally by `restore_failure_test.go:14-34` (`idxMarker > idxLoopEnd`). |
| Table-name SQL interpolation safe | PASS | `restore.go:62,77,152,231` — `table` is validated against `DumpTables` at `restore.go:51-53` before any SQL builds. `target.String()` uses `?` placeholders. |
| `copyInBinary` goroutine leak on error | PASS | `restore.go:225-238` — on `CopyFrom` error, `pr.CloseWithError(err)` unblocks the writer; `<-errc` drains. No leak. |
| NOTE (non-blocking) | NOTE | `restore.go:159-166` — if the marker UPSERT itself fails AFTER tables+ANALYZE succeed, `cleanupAfterFailure` is NOT invoked. Tables remain populated but no `tenant_baselines` row. This matches the documented "two-phase finalize" intent ("never restored" semantics) and is not a blocker; tracking only. |

#### F3 — `storage/scope.go` positive-only caching

| Check | Status | Evidence |
|---|---|---|
| Negative verdicts not cached | PASS | `storage/scope.go:28-30,44-49` — `resolveCacheGate` returns `shouldCache=false` for `has=false`; the call site only `s.Caches.Scope.Add` when `shouldCache`. |
| Cache key includes tenant id | PASS | `storage/scope.go:19` — `cacheKey := tenantID + "|" + region + "|" + version + "|" + subPath`. No cross-tenant collision is possible. |
| Sibling `ResolveSmapScope` follows the same pattern | PASS | `storage/smap.go:71-79` — the negative-path `Caches.Scope.Add(cacheKey, "shared")` was removed and replaced with a comment citing F3. |
| Tests pin both arms | PASS | `storage/scope_test.go:9-28` covers both `has=false` (no cache) and `has=true` (cache + tenant scope). |

#### F14 — `wzinput/status.go` `server.MarshalResponse[Status]`

| Check | Status | Evidence |
|---|---|---|
| Uses `server.MarshalResponse[T]` | PASS | `wzinput/status.go:80-82`. |
| RestModel implements `GetName`, `GetID`, `SetID` | PASS | `status.go:24,28,32`. |
| JSON:API relationship interfaces present (no-ops) | PASS | `status.go:34-56` (`GetReferences`, `GetReferencedIDs`, `GetReferencedStructs`, `SetToOneReferenceID`, `SetToManyReferenceIDs`, `SetReferencedStructs`). |
| Content-Type `application/vnd.api+json` | PASS | `status.go:79`. |

#### F2 — `commodity/processor.go` no outer transaction

| Check | Status | Evidence |
|---|---|---|
| Outer `database.ExecuteTransaction` removed | PASS | `commodity/processor.go` no longer imports `atlas-database`; the previous outer-tx wrapper is gone (`commodity/processor.go:17-39` shows the new chunked-commit `Register` loop). |
| Per-row commits via `s.Add(ctx)(m)()` | PASS | `commodity/processor.go:32`. |
| Test pins the contract | PASS | `commodity/processor_test.go:12-22` greps the source to ensure `database.ExecuteTransaction` does not reappear. |

#### F20 — `extractPortals` dedup

| Check | Status | Evidence |
|---|---|---|
| Dedup key includes (name, target, x, y) | PASS | `libs/atlas-wz/mapimage/layers.go:322-329`. |
| Test pins the contract | PASS | `libs/atlas-wz/mapimage/layers_portal_test.go:12-33`. |

#### SUB checklist (where applicable)

| ID | Check | Status | Evidence |
|---|---|---|---|
| SUB-01 | Business logic out of handler | PASS | `baseline/handler.go:23-24` delegates to `publishInner`/`restoreInner` which call `Publisher.Publish` / `Restorer.Restore`; `commodity/resource.go:17-26` calls `handleGetCommodityItemsRequest` etc.; `wzinput/resource.go:21-27` calls `uploadHandler` / `statusHandler`. |
| SUB-02 | No direct `db.Create`/`db.Save` in handlers | PASS | `grep -nE 'db\.Create\|db\.Save\|db\.Delete' services/atlas-data/atlas.com/data/{baseline/handler.go,commodity/resource.go,data/resource.go,wzinput/resource.go}` returns nothing. |
| SUB-03 | POST uses `RegisterInputHandler[T]` | PASS | `baseline/handler.go:23-24` — POST `/publish` and POST `/restore` both use `rest.RegisterInputHandler[PublishInputModel]` / `[RestoreInputModel]`. |
| SUB-04 | No manual JSON parsing | PASS | `grep -nE 'json\.NewDecoder\|json\.Unmarshal\|io\.ReadAll' services/atlas-data/atlas.com/data/{baseline/handler.go,commodity/resource.go,data/resource.go,wzinput/resource.go}` returns nothing. (`restore.go:132` uses `json.NewDecoder` to read the tar header — that is internal restore logic, not the HTTP handler — acceptable.) |
| EXCEPTION | `wzinput/resource.go` uses `RegisterHandler` for PATCH | NOTE | `wzinput/resource.go:21-28` documents that PATCH `/data/wz` is a binary multipart upload, not a JSON:API envelope. Using `RegisterInputHandler[T]` would consume the body as JSON. This is a documented exception, not a violation. |

### Phase 4 — Security Review (re-audit)

| ID | Check | Status | Evidence |
|---|---|---|---|
| SEC-01 (cache poisoning, F3) | Tenant isolation in scope cache | PASS | Cache key in `storage/scope.go:19` is salted by `tenantID`. Negative verdicts not cached so stale "shared" cannot leak after ingest. |
| SEC-02 (SQL injection, F1) | Table interpolation in publish | PASS | `publish.go:130` interpolates `table` from `DumpTables` (package constant). `canonical.TenantUUID` is a constant. No user-controlled string reaches the SQL. |
| SEC-03 (SQL injection, F5) | Table interpolation in restore | PASS | `restore.go:62,77,152,231` — `table` validated against `DumpTables` membership at `restore.go:51-53`. `target.String()` uses `?` placeholders. |
| SEC-04 (resource exhaustion, F5) | Restore goroutine leak | PASS | `restore.go:225-238` — `pr.CloseWithError` + `<-errc` drains the writer goroutine on every error path. |
| SEC-05 (sha256 verification, F5) | Hash check before DB mutation | PASS | `restore.go:115-117` returns before any DELETE/COPY; the temp file is then rewound at `:118-120` for the schema/table phases. |
| SEC-06 (temp file cleanup) | Publish + restore | PASS | `publish.go:51-52`, `restore.go:107-108` — both register `defer os.Remove` + `defer tmp.Close` in LIFO order. |
| SEC-07 (auth/token surfaces) | N/A | — | None of the changed files handle auth, tokens, OAuth, or open-redirect flows. |

### Drift from idiomatic patterns due to rebase

None observed. The 24-commit rebase did not introduce any guideline drift:

- `deploy/k8s/overlays/main/kustomization.yaml:278` correctly carries the atlas-renders image entry after taking main's `main-6af3cb0` tag during conflict resolution; the manifest still references `services/atlas-renders` consistently.
- `deploy/k8s/base/kustomization.yaml:53` includes `atlas-renders.yaml` and `:66` declares the `configMapGenerator` block referenced by F8.
- All Go imports compile; no `_ "atlas-data/document"` orphan from the F13 dead-code removal.
- `go.work` referenced lazily — `go test -race ./...` walks all three modules clean.

### Summary

- **Build / Vet / Race tests:** PASS for `libs/atlas-wz`, `services/atlas-data`, `services/atlas-renders` (only `libs/atlas-wz/go.mod` changed).
- **DOM-21 single-source compliance (F16):** PASS — `libs/atlas-wz/charparts/extract.go:22,100-106` uses `libs/atlas-constants/item` constants directly; no local redeclaration anywhere in the diff.
- **SUB-* applicable checks:** PASS, with `wzinput` PATCH documented as a deliberate non-JSON exception.
- **SEC checks on F1 / F3 / F5:** PASS. One NOTE on `restore.go:159-166` — marker UPSERT failure does not trigger `cleanupAfterFailure`; tracked as design-intentional, not a blocker.
- **Rebase drift:** None.

### Overall (re-audit)

**PASS** — the branch is mechanically clean against the backend developer guidelines. The single NOTE (marker UPSERT cleanup) is documented as the intended F5 contract and not actionable in this task.
