# Plan Audit — task-037-character-presets

**Plan Path:** docs/tasks/task-037-character-presets/plan.md
**Audit Date:** 2026-04-30
**Branch:** task-037-character-presets
**Base Branch:** main (54d506c16)
**HEAD:** a36405765
**Commits:** 30 (29 implementation + 1 planning)
**Files changed:** 111 (+12330 / -181)

## Executive Summary

The plan's 29 implementation tasks each produced their nominated commit, and the
backend half of the work (libs/atlas-saga, atlas-data, atlas-character,
atlas-saga-orchestrator, atlas-inventory, atlas-configurations,
atlas-character-factory) builds cleanly and passes its full unit-test suite.
The atlas-ui work is the weak point: vitest passes (66 files / 710 tests) but
`tsc -b` fails with **18 type errors** across the new preset pages, the wizard
types/tests, and the factory service test, which means `npm run build` (and
therefore the production Docker image) is broken on this branch. There are also
two integration gaps the plan called for but the implementation skipped: the
README/routes.conf update for the new factory endpoints (Task 21 Step 4) and
gateway routing for the `/api/factory/...` prefix the UI now uses.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | UseAverageStats on `CreateAndEquipAssetPayload` | DONE | `0d60e722e` — `libs/atlas-saga/payloads.go:128`, `unmarshal_test.go:+32` |
| 2 | `Gm` + `Meso` on `CharacterCreatePayload` | DONE | `b900036ae` — `libs/atlas-saga/payloads.go`, +29 lines test |
| 3 | atlas-data skill `MaxLevel` | DONE | `5b902aa3c` — `skill/rest.go`, `reader.go`, `rest_test.go` |
| 4 | atlas-data `ids=` filter on `/data/skills` | DONE | `3a1c318a4` — `skill/resource.go` (+44/-9), `resource_test.go` (+147) |
| 5 | atlas-character `Gm`/`Meso` on `CreateCharacterCommandBody` | DONE | `ed62524a7` — `kafka/message/character/kafka.go`, +22 lines test |
| 6 | `handleCreateCharacter` wires Gm/Meso | DONE (no test) | `a342bb519` — `consumer.go:356-357`. Plan Step 1 explicitly permits skipping the consumer test if no in-package harness exists; only builder calls were added. |
| 7 | `GET /characters/name-validity` | DONE | `18b1bbe18` — `name_validity_resource.go` (+45), `_test.go` (+233), `processor.go` widened, `resource.go` registers route. Handler signature deviates from plan as noted in prompt; tests cover regex/length/duplicate. |
| 8 | Orchestrator threads Gm/Meso | DONE | `34f0d4f9c` — producer + processor + mock + handler + test (`producer_test.go +26`) |
| 9 | Orchestrator threads UseAverageStats and splits create-item APIs | DONE | `041b5d544` — `RequestCreateItem` preserved, `RequestCreateItemWithStats` added; mock + tests updated |
| 10 | `asset.Create` -> `CreateOptions` | DONE | `f80ee969d` — `asset/processor.go`, `compartment/processor.go` |
| 11 | Variance bypass when `UseAverageStats=true` | DONE | `6d465f754` — extracted helper at `asset/processor.go:425 applyEquipStats` (deviation noted in prompt is implemented), 129 lines of new tests |
| 12 | Wire UseAverageStats from consumer | DONE | `5716eafd2` — `compartment/processor.go`, kafka_test.go +36, consumer.go forwards |
| 13 | Preset RestModel both scopes | DONE | `fd8175ef5` — `templates/.../preset/rest.go`, `tenants/.../preset/rest.go` + `rest_test.go` (+42) |
| 14 | `Presets` on `characters.RestModel` | PARTIAL | `6402a3e5b` — backend rest models updated (`templates/characters/rest.go`, `tenants/characters/rest.go`) and tested. **However the parallel atlas-ui type model (`services/atlas-ui/src/types/models/template.ts:26-28`) was never updated to add `presets` to `characters`.** That gap is the root cause of several Task 24 TS failures. |
| 15 | atlas-data client | DONE | `fd389c6ae` — `data/processor.go`, `skill_requests.go`, `item_requests.go`, `mock/processor.go` |
| 16 | Preset validator (12 rules) | DONE | `5aa767d3f` — `validator.go` mirrored at template + tenant scope, 296 lines of table-driven tests |
| 17 | Wire validator into PATCH | DONE | `7e808b84e` — both `templates/processor.go` and `tenants/processor.go`; JSON:API error meta path; test in `tenants/processor_test.go` (+90) |
| 18 | Seed 4th-job presets | DONE | `bc66615db` — 12 jobIds present in `template_gms_83_1.json` (112/122/132/212/222/232/312/322/412/422/512/522). Other GMS/JMS templates received `"presets": []`. Partial equipment fills are explicitly permitted by plan Step 3. |
| 19 | Factory clients | DONE | `f5ad0a09f` — `data/skill_requests.go`, `data/item_requests.go` (uses `data/equipment/{id}` per noted deviation), `configuration/preset_requests.go`, `character/name_validity_requests.go`, mocks for both |
| 20 | `CreateFromPreset` processor | DONE | `3c9a59b0a` — `factory/processor.go +196`, `factory/preset_rest.go`, 216 lines of tests in `processor_preset_test.go`. `byte` types used per noted deviation. |
| 21 | Factory routes | PARTIAL | `083ab6ae2` — routes registered at `factory/resource.go:91-92`, 146 lines of tests. **Step 4 (README + routes.conf update) was skipped:** no diff to `services/atlas-character-factory/README.md` or to `deploy/shared/routes.conf` / `deploy/compose/routes.conf`. The new endpoints are also registered under `/characters/from-preset`/`/characters/name-validity`, not under a `/factory` prefix, and the gateway has no `/api/factory/*` rule, so the UI's `BASE_PATH = "/api/factory"` calls (factory.service.ts:5) will 404 in production. |
| 22 | factory.service.ts | PARTIAL | `bdee880e3` — service implemented (98 lines) and 205 lines of tests (13 cases). **Test build fails:** `factory.service.test.ts:21` passes `type: "tenant"` to a `TenantBasic` literal which has no `type` field (TS2353). Vitest still passes because runtime ignores excess props. |
| 23 | React Query hooks | DONE | `315e06b56` — `useCreateCharacterFromPresetMutation`, `useNameValidity` (300ms debounce), `useAccountByName` (1s poll, 30s watchdog) + 468 lines of tests |
| 24 | Preset catalog editor pages | PARTIAL | `aeb4cf28c` — both pages (599+600 lines), schema (55 lines), 156 lines of test. **TypeScript build fails on these files (10 of the 18 errors):** the form types collide with the un-updated `TemplateAttributes`/tenant `characters` type (root cause = Task 14 gap), and `tsc -b` rejects the field-array `useFieldArray<PresetsFormValues, "presets">` plumbing. Vitest passes. |
| 25 | Routes/breadcrumbs | DONE (sidebar TBD) | `84d4fb305` — `App.tsx +4`, `breadcrumbs/routes.ts +12`. **Sidebar entry not visible in the diff;** plan Step 3 said "Sidebar entries" but no sidebar config was modified. |
| 26 | `ApplyPresetDialog` | DONE | `80d360068` — `ApplyPresetDialog.tsx` (198), test (236), wired into `AccountDetailPage.tsx` |
| 27 | `AdminBootstrapWizard` | PARTIAL | `7ff15dd05` — wizard (644), types (121), tests (343), wired into `AccountsPage.tsx`. **TypeScript build fails (3+ errors):** `AdminBootstrapWizard.types.ts:100` reducer narrowing under `noUncheckedIndexedAccess` is wrong; tests have `Object is possibly 'undefined'` at lines 70/71/94/118/119 and `HTMLElement \| undefined` args at 324/325. |
| 28 | Preset compensation integration test | DONE | `acd738a2b` — `saga/preset_integration_test.go` (+292). Uses `DispatchCharacterCreationRollbacks` per noted deviation. |
| 29 | TODO follow-ups | DONE | `a36405765` — 5 follow-ups recorded in `docs/TODO.md` |

**Completion Rate:** 24 DONE / 5 PARTIAL / 0 SKIPPED out of 29 (~83% fully clean, 100% present).
**Skipped without approval:** 0.
**Partial implementations:** 5 (Tasks 14, 21, 22, 24, 27).

## Skipped / Deferred Items

### Task 14 — frontend type model not updated
The plan modeled Task 14 as a backend-only change ("Add `Presets` field to
characters.RestModel (both scopes)"), but downstream Tasks 22/24 depend on the
TypeScript twin at `services/atlas-ui/src/types/models/template.ts:21-28`,
which still types `characters` as `{ templates: CharacterTemplate[] }` (no
`presets`). Both new presets-form pages compensate with `(... as any).presets`
on the read path but submit a typed object on the write path, which `tsc`
rejects (`templates-character-presets-form.tsx:71`,
`tenants-character-presets-form.tsx:70`, both TS2353). This propagates to
several other TS2322/TS2345 errors on the same files.

**Impact:** atlas-ui production build (`npm run build` = `tsc -b && vite build`)
fails. Until the FE types are extended, the UI cannot be deployed.

### Task 21 Step 4 — README + routes.conf not updated
The plan explicitly listed "Update README + routes.conf". `git diff
54d506c16..HEAD` shows no changes to:
- `services/atlas-character-factory/README.md`
- `deploy/shared/routes.conf`
- `deploy/compose/routes.conf`

`deploy/shared/routes.conf:187` only routes `/api/characters/seed*` to the
factory; `/api/characters/*` falls through to atlas-character. The new factory
handlers register at `/characters/from-preset` and `/characters/name-validity`
(`factory/resource.go:91-92`), so requests routed by the gateway will hit
atlas-character (which only has `/characters/name-validity`, not
`/characters/from-preset`).

The UI exacerbates this by calling under `BASE_PATH = "/api/factory"`
(`factory.service.ts:5`), a prefix the gateway has no rule for at all.

**Impact:** End-to-end traffic from the UI to the new endpoints will 404
through the gateway. PRD acceptance §10 #1 (apply preset to a real account from
UI) cannot pass against the deployed cluster as-is.

### Task 22 / 24 / 27 — TypeScript build is red
`node_modules/.bin/tsc -b` reports 18 errors total:

| File | Line | Class | Cause |
|------|------|-------|-------|
| `factory.service.test.ts` | 21 | TS2353 | `type: "tenant"` not in `TenantBasic` |
| `templates-character-presets-form.tsx` | 43, 71, 109, 121 | TS2322/TS2345/TS2353 | `presets` field not on `TemplateAttributes.characters` (Task 14 FE gap) + resolver/SubmitHandler generics |
| `tenants-character-presets-form.tsx` | 42, 70, 108, 120 | same | same |
| `__tests__/templates-character-presets-form.test.tsx` | 149 | TS2488 | `any[] | undefined` not iterable under `noUncheckedIndexedAccess` |
| `AdminBootstrapWizard.types.ts` | 100 | TS2322 | reducer return type narrows wrongly with strict index-access |
| `__tests__/AdminBootstrapWizard.test.tsx` | 70, 71, 94, 118, 119, 324, 325 | TS2532/TS2345 | strict index-access on test fixtures |

`tsconfig.app.json` has `strict`, `noUncheckedIndexedAccess`,
`exactOptionalPropertyTypes`, `noUnusedLocals`, `noUnusedParameters`, and
`erasableSyntaxOnly` all on (per the atlas-ui CLAUDE.md, all 7 home-hub strict
flags are intentionally on now). The new code does not satisfy them.

### Task 25 — sidebar entry
Plan Step 3 says "Sidebar entries". Diff for commit `84d4fb305` only touches
`App.tsx` + `breadcrumbs/routes.ts`. The sidebar component's nav config did not
receive new entries for the preset pages, so users cannot reach the new pages
without typing the URL.

## Build & Test Results

| Service / Package | Build | Tests | Notes |
|-------------------|-------|-------|-------|
| libs/atlas-saga | PASS | PASS | unmarshal tests cover both new payload fields |
| atlas-data | PASS | PASS | skill MaxLevel + ids filter both covered |
| atlas-character | PASS | PASS | 52 s `character` package, name-validity resource_test included |
| atlas-saga-orchestrator | PASS | PASS | full saga suite ~178 s; preset_integration_test passes |
| atlas-inventory | PASS | PASS | 129 lines variance-bypass test pass; consumer kafka_test pass |
| atlas-configurations | PASS | PASS | validator tests, seeder tests, characters/preset tests all green |
| atlas-character-factory | PASS | PASS | factory package 27 s, preset processor + handler tests pass |
| atlas-ui (vitest) | n/a | PASS | 66 files, 710 tests, 9.46 s |
| atlas-ui (tsc -b) | **FAIL** | n/a | 18 errors across 6 files (see table above); `npm run build` cannot complete |

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE — every plan task produced a commit, and
  the noted deviations from the prompt (handler signature, helper extraction,
  partial seed equipment, equipment-vs-items endpoint, byte-typed skill levels,
  rollback dispatcher in test) are all consistent with the freedoms the prompt
  granted. The genuine gaps are the FE typing oversight from Task 14, the
  routes.conf/README skip in Task 21, the missing sidebar entry from Task 25,
  and the strict-TS regressions in Tasks 22/24/27.
- **Recommendation:** NEEDS_FIXES — the work is functionally close to
  complete, but the UI cannot ship until `tsc -b` is green and the gateway can
  reach the new factory endpoints.

## Action Items

1. **Extend frontend type model** (root cause for most TS errors): add
   `presets?: PresetRestModel[]` to `TemplateAttributes.characters` in
   `services/atlas-ui/src/types/models/template.ts:26` and to the matching
   tenant configuration type, then drop the `as any` casts in
   `templates-character-presets-form.tsx` / `tenants-character-presets-form.tsx`.
2. **Fix the remaining strict-TS errors** in the new preset pages (resolver +
   SubmitHandler generics), the form test
   (`templates-character-presets-form.test.tsx:149` destructuring), and
   `AdminBootstrapWizard.types.ts:100` reducer narrowing plus the four `Object
   is possibly 'undefined'` guards in the wizard tests.
3. **Fix `factory.service.test.ts:21`** — drop the bogus `type: "tenant"`
   property from the `TenantBasic` literal.
4. **Add gateway routing** for the new factory endpoints. Either (a) update
   `deploy/shared/routes.conf` and `deploy/compose/routes.conf` with a rule
   that maps `/api/factory/characters/{from-preset,name-validity}` to
   `atlas-character-factory:8080`, or (b) change the factory's mux prefix to
   `/factory/...` to match the URLs the UI is calling. Either way, also update
   `services/atlas-character-factory/README.md`'s endpoint table per Task 21
   Step 4.
5. **Add the sidebar entry** that Task 25 Step 3 called for so the new preset
   pages are reachable without manual URL entry.
6. **Re-run `npm run build`** in `services/atlas-ui/` and the full
   `go test ./... -count=1` matrix after fixes 1-5; sign off only when
   `tsc -b` is clean.

---

## Backend Audit

- **Audit Scope:** Go changes on branch `task-037-character-presets` (base `54d506c16` -> HEAD `a36405765`)
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/`
- **Date:** 2026-04-30
- **Build:** PASS (every touched module: libs/atlas-saga, atlas-data, atlas-character, atlas-character-factory, atlas-configurations, atlas-inventory, atlas-saga-orchestrator)
- **Tests:** PASS (all packages green; saga-orchestrator/saga ran 180.1s incl. new TestPresetCompensation)
- **Overall:** NEEDS-WORK

### Build & Test Results

```
libs/atlas-saga                    go build ./...   ok
                                   go test  ./...   ok  github.com/Chronicle20/atlas/libs/atlas-saga         0.003s
services/atlas-data                go build ./...   ok
                                   go test  ./...   ok  atlas-data/skill                                     0.082s    (and others)
services/atlas-character           go build ./...   ok
                                   go test  ./...   ok  atlas-character/character                            45.367s
services/atlas-character-factory   go build ./...   ok
                                   go test  ./...   ok  atlas-character-factory/factory                      36.771s
services/atlas-configurations      go build ./...   ok
                                   go test  ./...   ok  atlas-configurations/tenants/characters/preset       0.005s
                                                    ok  atlas-configurations/tenants                          0.019s
                                                    ok  atlas-configurations/templates                        0.011s
services/atlas-inventory           go build ./...   ok
                                   go test  ./...   ok  atlas-inventory/asset                                 0.009s
                                                    ok  atlas-inventory/compartment                            0.066s
services/atlas-saga-orchestrator   go build ./...   ok
                                   go test  ./...   ok  atlas-saga-orchestrator/saga                          180.138s    (TestPresetCompensation passes)
```

### Package Classification (touched only)

| Package | Type | Notes |
|---------|------|-------|
| `libs/atlas-saga` | Shared payload library | Only payload structs added (Gm, Meso, UseAverageStats). No domain checks apply. |
| `services/atlas-data/atlas.com/data/skill` | Read-only data service (no model.go) | `MaxLevel` field + `?ids=` filter. Storage-backed, not a DDD domain. |
| `services/atlas-character/atlas.com/character/character` | Domain | DOM-* checklist applies (preexisting domain; new `CheckNameValidity` + `name_validity_resource.go` added). |
| `services/atlas-character-factory/atlas.com/character-factory/factory` | Sub-domain (no model/entity) | SUB-* checklist applies; emits sagas. |
| `services/atlas-character-factory/atlas.com/character-factory/{character,configuration,data}` | REST clients | `requests.go` pattern checked. |
| `services/atlas-configurations/atlas.com/configurations/tenants` | Domain (JSONB-backed config) | DOM-* applies. New preset Validator wired in. |
| `services/atlas-configurations/atlas.com/configurations/templates` | Domain (JSONB-backed config) | DOM-* applies. New preset Validator wired in. |
| `services/atlas-configurations/atlas.com/configurations/{tenants,templates}/characters/preset` | Embedded RestModel + Validator | Sub-component, no JSON:API top-level resource. |
| `services/atlas-configurations/atlas.com/configurations/data` | REST client | `requests.go` pattern checked. |
| `services/atlas-inventory/atlas.com/inventory/asset` | Domain | DOM-* applies. `Create` signature refactored to `CreateOptions`. |
| `services/atlas-inventory/atlas.com/inventory/compartment` | Domain | DOM-* applies. New `useAverageStats` parameter threaded. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/{character,compartment,saga}` | Saga handler / processor | Threading-only changes; integration test added. |

### Domain Checklist Results

#### atlas-character/character (DOM-*)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | `services/atlas-character/atlas.com/character/character/builder.go` (preexisting). |
| DOM-02 | ToEntity() method | PASS | `services/atlas-character/atlas.com/character/character/entity.go` (preexisting). |
| DOM-03 | Make(Entity) function | PASS | `services/atlas-character/atlas.com/character/character/entity.go` (preexisting). |
| DOM-04 | Transform function | PASS | `services/atlas-character/atlas.com/character/character/rest.go` (preexisting). |
| DOM-05 | TransformSlice function | PASS | `services/atlas-character/atlas.com/character/character/rest.go` (preexisting). |
| DOM-06 | Processor accepts FieldLogger | PASS | `services/atlas-character/atlas.com/character/character/processor.go` `NewProcessor(l logrus.FieldLogger, ...)`. |
| DOM-07 | Handlers pass d.Logger() | PASS | `services/atlas-character/atlas.com/character/character/name_validity_resource.go:32` — `NewProcessor(d.Logger(), d.Context(), d.DB())`. |
| DOM-08 | POST/PATCH use RegisterInputHandler | PASS | New endpoint is GET; resource.go:33 registers `name-validity` via `registerGet(...)`; existing POST/PATCH use `RegisterInputHandler[RestModel]` (resource.go:32, :35 in diff). |
| DOM-09 | Transform errors handled | PASS | New code uses `_ = json.NewEncoder(w).Encode(...)` for the response struct only — there is no `Transform` call to ignore. |
| DOM-10 | Test DB has tenant callbacks | PASS | `services/atlas-character/atlas.com/character/character/name_validity_resource_test.go` reuses existing test harness which calls `database.RegisterTenantCallbacks` (preexisting). |
| DOM-11 | Providers use lazy evaluation | PASS | Existing `provider.go` (preexisting; not modified). |
| DOM-12 | No os.Getenv in handlers | PASS | `name_validity_resource.go` — no `os.Getenv`. |
| DOM-13 | No cross-domain logic in handlers | PASS | `name_validity_resource.go:32` calls own processor `CheckNameValidity`. |
| DOM-14 | Handlers don't call providers directly | PASS | Handler delegates to `processor.CheckNameValidity`. |
| DOM-15 | No direct entity creation in handlers | PASS | No `db.Create`/`db.Save`/`db.Delete` in `name_validity_resource.go`. |
| DOM-16 | administrator.go for writes | PASS | Preexisting; no new write path added. |
| DOM-17 | Domain error -> HTTP status mapping | WARN | `name_validity_resource.go:34` maps every processor error to `500`. The processor returns nil-error for invalid inputs (length/regex/duplicate) via `NameValidityResult.Valid=false`, so 500 fires only for genuine internal errors — acceptable but the mapping is implicit, not explicit. |
| DOM-18 | JSON:API interface on REST models | WARN | `name_validity_resource.go:12-16` defines `NameValidityResponse` as a plain struct that does NOT implement `GetName()`/`GetID()`/`SetID()` and is hand-encoded with `json.NewEncoder` (line 39). Per `ai-guidance.md` §"REST Generation Specifics", all REST models should implement the JSON:API interface; per `anti-patterns.md`, `server.MarshalResponse[T]` should be used instead of manual JSON encoding. The endpoint deliberately returns plain JSON for a simple boolean check, which is the same shape mirrored back from the factory. **This is a deliberate departure from JSON:API conventions called out in the task scope, but it remains a guideline deviation.** |
| DOM-19 | Request models use flat structure | PASS | No new request models added (GET endpoint only). |
| DOM-20 | Table-driven tests | PASS | `name_validity_resource_test.go` uses `tests := []struct{...}` with `t.Run(tt.name, ...)` (preexisting style). |

#### atlas-character-factory/factory (SUB-*)

This package emits sagas; it has no `model.go`, `entity.go`, `administrator.go`, or `provider.go`. SUB-* applies.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Has processor | PASS | `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:43-69` — `Processor` interface + `ProcessorImpl` + `NewProcessor`. |
| SUB-02 | Has administrator for writes | N/A | No DB writes in this package — sagas emit Kafka. |
| SUB-03 | Uses RegisterInputHandler[T] for POST | **FAIL** | `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:91` registers `POST /characters/from-preset` with `rest.RegisterHandler` (no-input variant), then manually decodes the body with `json.NewDecoder(r.Body).Decode(&in)` at line 119. `PresetCreateRestModel` already implements the JSON:API interface (`preset_rest.go:10-12`) and is the natural fit for `rest.RegisterInputHandler[PresetCreateRestModel]`. |
| SUB-04 | No manual JSON parsing | **FAIL** | `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:119` — `json.NewDecoder(r.Body).Decode(&in)`. Per `anti-patterns.md` line 27 (`Manual JSON:API envelope handling`) and the SUB-04 rule, this should not appear in handlers. |

Additional findings on this package not covered by SUB-*:

- **Custom error response helper.** `factory/resource.go:25-38` defines `writeErrorResponse` and uses it from `handleCreateFromPreset` (lines 120, 126, 134) and `handleNameValidity` (lines 150, 155, 163). Per `anti-patterns.md` line 30 — "Custom error response helpers - Just write status codes directly" — this is an explicit anti-pattern. The convention across the rest of the codebase (see e.g. `atlas-character/character/character/name_validity_resource.go:24-35`) is `w.WriteHeader(http.StatusBadRequest); return`.
- **Handler bypasses processor.** `factory/resource.go:159-160` — `handleNameValidity` instantiates `character.NewNameValidityClient(d.Logger())` directly and calls `client.Check(...)`. This is a `resource.go -> requests.go` jump that skips the processor layer (DOM-13/14 / `anti-patterns.md` "Handlers calling provider functions directly"). Even for a simple proxy, the convention is `processor.CheckNameValidity` (or equivalent), not direct client instantiation.
- **Saga emission inside handler request goroutine.** `processor.go:323` — `saga.NewProcessor(p.l, ctx).Create(sg)` in `CreateFromPreset` is acceptable per the cross-service pattern; tenant context propagates because `CreateFromPreset` accepts `ctx` and `saga.NewProcessor` uses it.
- **Tenant propagation.** `processor.go:86, 258` — both `Create` and `CreateFromPreset` call `tenant.MustFromContext(ctx)`. Tenant flows through correctly.

#### atlas-character-factory clients (`character/`, `configuration/`, `data/`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Pattern: `requests.go` uses `requests.GetRequest[T]`/`MakeGetRequest[T]` | WARN | `character/name_validity_requests.go:43-67` — hand-rolls `http.NewRequestWithContext` + `http.DefaultClient.Do` + manual `requests.SpanHeaderDecorator`/`TenantHeaderDecorator` instead of using the standard `requests.GetRequest[T]` helper which `data/skill_requests.go:39` and `data/item_requests.go:27` correctly use. The behaviour is equivalent (tenant + span headers are applied), but the pattern is inconsistent with `file-responsibilities.md` §`requests.go`. |
| RestModel implements JSON:API interface | PASS | `character/name_validity_requests.go:17-21` (NameValidityResult — note: it does NOT implement the interface, but it is consumed via direct `json.Decoder` on line 65, not via the JSON:API stack). `data/skill_requests.go:18-27` and `data/item_requests.go:15-24` implement `GetName/GetID/SetID`. |
| Tenant header propagation | PASS | All three client packages propagate tenant + span headers via the decorators (explicit in `name_validity_requests.go:49-50`, automatic in `skill_requests.go`/`item_requests.go` via `requests.GetRequest[T]`). |

#### atlas-configurations/tenants and templates (DOM-*)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts FieldLogger | PASS | `tenants/processor.go:22` — `NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB)`. Same in `templates/processor.go:22`. |
| DOM-07 | Handlers pass d.Logger() | PASS | `tenants/resource.go:35,53,72,95,121` and `templates/resource.go` all use `d.Logger()`. |
| DOM-08 | POST/PATCH use RegisterInputHandler | PASS | `tenants/resource.go:24` (POST) and `:26` (PATCH) use `rest.RegisterInputHandler[RestModel]`. Same in `templates/resource.go:23,27`. |
| DOM-09 | Transform errors handled | N/A | No Transform calls (RestModel is unmarshaled directly via `json.Unmarshal` on `Entity.Data`). |
| DOM-12 | No os.Getenv in handlers | PASS | Grep returns no matches in `tenants/resource.go` or `templates/resource.go`. |
| DOM-13 | No cross-domain logic in handlers | PASS | All handlers delegate to `NewProcessor(...)`. |
| DOM-14 | Handlers don't call providers directly | PASS | Confirmed by reading `tenants/resource.go` and `templates/resource.go`. |
| DOM-15 | No direct entity creation in handlers | PASS | No `db.Create` in either resource.go. |
| DOM-17 | Domain error -> HTTP status mapping | PASS | `tenants/resource.go:76-86` and `templates/resource.go:124-135` map `*validationFailureError` -> 400 (with JSON:API `errors[]` body), default -> 500. |
| DOM-18 | JSON:API interface on REST models | PASS | Top-level `RestModel` (preexisting) implements JSON:API. The new embedded `preset.RestModel` is an embedded sub-struct (not a top-level resource), so it does not need `GetName`/`GetID`/`SetID` — and the diff at `tenants/characters/rest.go:8-11` keeps it embedded under `RestModel.Characters.Presets`. |
| DOM-20 | Table-driven tests | PASS | `tenants/processor_test.go` and `tenants/characters/preset/validator_test.go` both use the `tests := []struct{...}` + `t.Run` pattern. |
| Error response shape | WARN | `tenants/resource.go:80` — manual `json.NewEncoder(w).Encode(map[string]any{"errors": ve.AsJSONAPIErrors()})`. This is a deliberate JSON:API error envelope (`errors[]`) which `server.MarshalResponse` does not support, so the manual encode is reasonable, but it does spread JSON:API envelope construction across the codebase. |

##### Validator (`tenants/characters/preset/validator.go` and `templates/.../preset/validator.go`)

| Aspect | Status | Evidence |
|---------|--------|----------|
| 12 rules implemented | PASS | `validator.go:53-135` covers R-1..R-12 (name length, description length, jobId, gender, level, equipment template+equippable+slot uniqueness, inventory template+quantity, skill ids+level+batch lookup). |
| Tenant context propagation through `data.Client` | PASS | `validator.go:33,47` accepts `ctx`; passes to `client.GetItemById(ctx, ...)` and `client.GetSkillsByIds(ctx, ...)` which use `requests.GetRequest[T]` (auto tenant headers). |
| UUID assignment before validation (R-1 mutate-then-validate) | PASS | `validator.go:34-38` assigns UUIDs first so error rows always carry a stable id. |
| Mirror between templates and tenants | WARN | `templates/characters/preset/validator.go` and `tenants/characters/preset/validator.go` are byte-identical (143 lines each). This is duplicated logic; one of them could re-import the other. The same is true of `templates/characters/preset/rest.go` vs `tenants/characters/preset/rest.go` (51 lines each, identical), and the `validation_error.go` files in `templates/` and `tenants/`. Per `anti-patterns.md` ("Leaving dead code after refactoring") and the project's CLAUDE.md "Code Patterns" section ("prefer straightforward moves over re-exporting type aliases" — but ALSO "Keep abstractions clean — don't break service boundaries"), this duplication is acceptable as a service-boundary safeguard but should be tracked. |
| `gm` field validation | INFO | `tenants/characters/preset/rest.go:40` exposes a `Gm int` field on the preset, but `validator.go` does not enforce any range/permission rule on it. The PRD calls for 12 rules; gm is intentionally not one of them. Acceptable per scope, but creating a preset with `gm: 1` will pass validation and propagate into character creation. |

##### atlas-configurations data client (`configurations/data/processor.go`)

| Aspect | Status | Evidence |
|---------|--------|----------|
| Mock alongside real client | PASS | `configurations/data/mock/processor.go` mirrors the `Client` interface. |
| Tenant header propagation | PASS | `configurations/data/skill_requests.go:39` uses `requests.GetRequest[[]SkillRestModel]` (auto headers). |
| Equippability derivation | PASS | `processor.go:69-83` — `inventory.TypeFromItemId` short-circuits non-equip lookups, no extra round-trip. |

#### atlas-inventory/asset (DOM-*)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts FieldLogger | PASS | `asset/processor.go` (preexisting). |
| DOM-09 | Transform errors handled | N/A | No new Transform calls. |
| DOM-15 | No direct entity creation in handlers | PASS | `Create` signature refactored to `CreateOptions` struct; no handler-side mutations. |
| DOM-16 | administrator.go for writes | PASS | Preexisting. |
| DOM-20 | Table-driven tests | PASS | `asset/processor_test.go:34-113` uses table-driven `t.Run` (`TestApplyEquipStats_*`). |
| `applyEquipStats` extraction | PASS | `asset/processor.go:419-460` cleanly factors variance vs. average-stat paths; called from `processor.go:312` (the `case` branch for `inventory.TypeValueEquip`). |
| `CreateOptions` reduces param-list size | PASS | Refactor reduces a 10-arg signature to 5 args + struct. Comment notes UseAverageStats is wired through Task 11 (already done in this branch). |

#### atlas-inventory/compartment (DOM-*)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| `CreateAsset*` thread `useAverageStats` | PASS | `compartment/processor.go:968-1024` — three signatures `CreateAssetAndEmit`/`CreateAssetAndLock`/`CreateAsset` all accept `useAverageStats bool`. |
| Existing call sites pass `false` | PASS | `compartment/processor.go:1208,1224` — `AttemptItemPickUp` calls pass `false`; `:947` (ExpireAsset replacement) passes `asset.CreateOptions{Quantity:1, Expiration:time.Time{}}` which leaves `UseAverageStats: false` (struct zero). |
| Wire-format change | INFO | `kafka/message/compartment/kafka.go:99-106` adds `UseAverageStats bool` with `json:"useAverageStats,omitempty"`. The `omitempty` keeps backwards-compatible payloads. The matching `kafka_test.go` snapshot was updated. |

#### atlas-saga-orchestrator (touched packages)

| Aspect | Status | Evidence |
|---------|--------|----------|
| Mock parity | PASS | `character/mock/processor.go:40,247` and `compartment/mock/processor.go:14,32` updated to match new signatures. |
| Saga handler wiring | PASS | `saga/handler.go:1393` passes `payload.Gm, payload.Meso`; `:1421` passes `payload.UseAverageStats`. |
| `RequestCreateItemWithStats` backward compat | PASS | `compartment/processor.go:56-58` — old `RequestCreateItem` delegates to `RequestCreateItemWithStats(..., false)`, preserving every existing caller. |
| Integration test | PASS | `saga/preset_integration_test.go:96-292` defines `TestPresetCompensation` exercising `DispatchCharacterCreationRollbacks` end-to-end. Test uses `t.Run`-free single-scenario layout (acceptable for an integration test); asserts DestroyItem reverse-walk + DeleteCharacter + zero DeleteSkill calls. Passes in 180s suite. |

#### atlas-data/skill

| Aspect | Status | Evidence |
|---------|--------|----------|
| `?ids=` filter implementation | PASS | `skill/resource.go:32-87` — accepts comma-separated and repeated `ids` params, `400` on parse error. |
| `MaxLevel` derivation | PASS | `skill/reader.go:107-115` — clamps `len(es)` to `uint8` (255 ceiling). Tests added in `rest_test.go`. |
| Test coverage | PASS | `skill/resource_test.go` 147 lines, table-driven across name + ids + bad-input cases. |

### Security Review

Not applicable — none of the changed services are auth/token services.

### Summary

#### Blocking (must fix)

- **SUB-03 / DOM-08:** `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:91` registers `POST /characters/from-preset` with `rest.RegisterHandler` (no-input variant) instead of `rest.RegisterInputHandler[PresetCreateRestModel]`. The model already implements JSON:API (`preset_rest.go:10-12`), so the fix is mechanical: change the registration and the handler signature, and delete the manual decode at `resource.go:119`.
- **SUB-04 / Anti-pattern "Manual JSON:API envelope handling":** `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:119` — `json.NewDecoder(r.Body).Decode(&in)`. Replace with `RegisterInputHandler[T]`-driven flow (see fix above).

#### Non-Blocking (should fix)

- **Anti-pattern "Custom error response helpers":** `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:25-38` (`writeErrorResponse`). Delete the helper and use `w.WriteHeader(...); return` like the rest of the codebase (e.g. `atlas-character/character/character/name_validity_resource.go:23-30`). Currently called from lines 120, 126, 134, 150, 155, 163.
- **DOM-13 / "Handlers calling provider functions directly":** `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:159-160` — `handleNameValidity` instantiates `character.NewNameValidityClient(d.Logger())` directly. Move the call onto `factory.Processor` (e.g. add `CheckName(ctx, name, worldId) (NameValidityResult, error)` that wraps the client).
- **Inconsistent client pattern:** `services/atlas-character-factory/atlas.com/character-factory/character/name_validity_requests.go:43-67` hand-rolls `http.NewRequestWithContext` + `http.DefaultClient.Do` instead of the standard `requests.GetRequest[T]` helper. Same package's `data/skill_requests.go` and `data/item_requests.go` already use the standard helper. Refactor for consistency with `file-responsibilities.md` §`requests.go`.
- **DOM-18 deviation called out in scope:** `services/atlas-character/atlas.com/character/character/name_validity_resource.go:12-43` returns plain JSON via `json.NewEncoder(w).Encode(NameValidityResponse{...})` and `NameValidityResponse` does not implement the JSON:API interface. The task scope explicitly notes this is intentional for a boolean-check endpoint, but it is a guideline departure that should be tracked (and now exists in two services with the same shape).
- **Duplicated preset RestModel + Validator:** `services/atlas-configurations/atlas.com/configurations/templates/characters/preset/{rest.go,validator.go}` is byte-identical to `services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/{rest.go,validator.go}` (51 + 143 lines each). Same for `validation_error.go`. Whether to keep this duplication is a judgement call (service-boundary safety vs. DRY), but it should be documented as an intentional split rather than left as drift-prone copies.
- **DOM-17 implicit mapping in name-validity:** `services/atlas-character/atlas.com/character/character/name_validity_resource.go:33-37` returns 500 for any processor error. The processor returns nil-error for invalid/duplicate names (encoded into `NameValidityResult.Valid=false`), so 500 is reserved for genuine internal failures, which is acceptable — but making this contract explicit via a dedicated error type would avoid future regressions.

### audit-backend.json

A machine-readable snapshot of the findings is captured below for downstream tooling.

```json
{
  "service": "task-037-character-presets",
  "scope": "branch (multi-service)",
  "date": "2026-04-30",
  "build": "pass",
  "tests": {"status": "pass", "notes": "all touched packages green"},
  "overallStatus": "needs-work",
  "findings": [
    {"id": "SUB-03", "severity": "blocking", "file": "services/atlas-character-factory/atlas.com/character-factory/factory/resource.go", "line": 91, "detail": "POST /characters/from-preset uses RegisterHandler instead of RegisterInputHandler[PresetCreateRestModel]"},
    {"id": "SUB-04", "severity": "blocking", "file": "services/atlas-character-factory/atlas.com/character-factory/factory/resource.go", "line": 119, "detail": "manual json.NewDecoder(r.Body).Decode in handler"},
    {"id": "anti-custom-error-helper", "severity": "non-blocking", "file": "services/atlas-character-factory/atlas.com/character-factory/factory/resource.go", "line": 25, "detail": "writeErrorResponse helper violates 'Custom error response helpers' anti-pattern"},
    {"id": "DOM-13", "severity": "non-blocking", "file": "services/atlas-character-factory/atlas.com/character-factory/factory/resource.go", "line": 159, "detail": "handleNameValidity instantiates character.NewNameValidityClient directly instead of going through Processor"},
    {"id": "client-pattern-inconsistency", "severity": "non-blocking", "file": "services/atlas-character-factory/atlas.com/character-factory/character/name_validity_requests.go", "line": 43, "detail": "hand-rolls http.NewRequestWithContext instead of using requests.GetRequest[T]"},
    {"id": "DOM-18", "severity": "non-blocking", "file": "services/atlas-character/atlas.com/character/character/name_validity_resource.go", "line": 12, "detail": "NameValidityResponse does not implement JSON:API interface; uses json.NewEncoder instead of server.MarshalResponse (deliberate per task scope)"},
    {"id": "duplication", "severity": "non-blocking", "files": ["services/atlas-configurations/atlas.com/configurations/templates/characters/preset/rest.go", "services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/rest.go", "services/atlas-configurations/atlas.com/configurations/templates/characters/preset/validator.go", "services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/validator.go"], "detail": "byte-identical duplication of preset RestModel + Validator across templates and tenants packages"},
    {"id": "DOM-17", "severity": "non-blocking", "file": "services/atlas-character/atlas.com/character/character/name_validity_resource.go", "line": 33, "detail": "every processor error maps to 500; explicit error-type matching would be safer"}
  ]
}
```
