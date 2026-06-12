<!-- ============================================================= -->
<!-- FRONTEND GUIDELINES REVIEW (frontend-guidelines-reviewer)     -->
<!-- ============================================================= -->

# Frontend Audit — task-087-change-map-to-maps

- **Audit Scope:** `main..HEAD` (BASE `464e8c6e`, HEAD `efcb6ef2`), `services/atlas-ui/` TS/React changes
- **Guidelines Source:** frontend-dev-guidelines skill (FE-* checklist)
- **Date:** 2026-06-12
- **Build:** PASS (`npm run build` — tsc + vite, built in 1.81s, no type errors)
- **Tests:** 740 passed, 0 failed (`vitest run` — 80 files)
- **Overall:** PASS

## Build & Test Results

```
npm run build  → ✓ built in 1.81s (tsc -b clean; only a pre-existing chunk-size warning on ConversationEditorPanel)
npm test       → Test Files 80 passed (80) | Tests 740 passed (740)
```

Both objective gates pass. Verified directly, not on faith.

## File Inventory

| File | Classification | Change |
|------|----------------|--------|
| `src/types/models/location.ts` | Type | NEW — `CharacterLocation` JSON:API model + `ChangeMapData` |
| `src/services/api/locations.service.ts` | Service | NEW — `getByCharacterId` GET + `changeMap` PATCH |
| `src/lib/hooks/api/useCharacterLocation.ts` | Hook | NEW — React Query hook, tenant-scoped key |
| `src/services/api/__tests__/locations.service.test.ts` | Test | NEW |
| `src/components/features/characters/CharacterMapCell.tsx` | Component | NEW — per-row location query |
| `src/components/features/characters/__tests__/ChangeMapDialog.test.tsx` | Test | NEW |
| `src/components/features/characters/ChangeMapDialog.tsx` | Component | read/write repointed to location endpoint |
| `src/pages/characters-columns.tsx` | Page (column defs) | Map column → `CharacterMapCell` |
| `src/components/features/characters/AttributesPanel.tsx` | Component | Map row → `CharacterMapCell` |
| `src/types/models/character.ts` | Type | removed `mapId` from attributes + `UpdateCharacterData` |
| `src/components/features/accounts/EmptySlotTile.tsx` | Component | dropped dead `mapId` assignment |
| `src/components/features/characters/ApplyPresetDialog.tsx` | Component | dropped dead `mapId` assignment |
| `src/components/features/characters/__tests__/AttributesPanel.test.tsx` | Test | dropped `mapId` fixture field |
| `src/lib/hooks/api/__tests__/useCharacters.test.tsx` | Test | fixture update |
| `src/lib/hooks/api/useSeed.ts`, `src/services/api/seed.service.ts`, `src/pages/SetupPage.tsx`, `seed.service.test.ts`, `useSeed.test.tsx` | Hook/Service/Page/Test | **OUT OF SCOPE** — unrelated `scope`-param revert riding on this branch (see note below) |

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | grep `: any`/`as any`/`<any>` across all new/changed source files → zero matches. Test fixtures use `as never`, not `any` (`ChangeMapDialog.test.tsx:46`) |
| FE-02 | No manual class concatenation | PASS | New files use plain string literals / `cn()` where conditional. `CharacterMapCell.tsx:14` `to={"/maps/" + mapIdStr}` is a router `to` prop, not `className`. `ChangeMapDialog.tsx:190,193` use a bare ternary on `className` (pre-existing, see FE-06 note) — no `+`/template concat introduced |
| FE-03 | No direct API client in components | PASS | No `@/lib/api/client` import in any component/page. `locations.service.ts:1` imports it — correct (service layer) |
| FE-04 | No inline Zod in components | PASS (N/A) | No `z.*` in `ChangeMapDialog.tsx`/`CharacterMapCell.tsx` (form is hand-rolled `useState`; see FE-15 note) |
| FE-05 | No spinners for content loading | PASS | No `animate-spin` in changed files. `CharacterMapCell` delegates loading to `MapCell` which renders `<Skeleton>`; submit button uses text `"Updating..."` not a spinner |
| FE-06 | No hardcoded colors | PASS (no regression) | `ChangeMapDialog.tsx:190` (`border-red-500 focus-visible:ring-red-500`) and `:193` (`text-red-500`) are hardcoded — but **pre-existing**, unchanged by this diff. No new hardcoded colors introduced by task-087 |
| FE-07 | No state mutation | PASS | `ChangeMapDialog` uses `setMapId`/`setSyncedMapId` immutably; render-time `if (currentMapId != null && currentMapId !== syncedMapId)` (`:34-37`) is React's documented adjust-state-during-render pattern, guarded to avoid loops. No `.push/.splice/.sort` |
| FE-08 | No default exports for components | PASS | `CharacterMapCell.tsx:6` named `export function`; `locations.service.ts`/`useCharacterLocation.ts`/`location.ts` all named exports. Zero `export default` in new files |
| FE-09 | Tenant guard in hooks | PASS | `useCharacterLocation.ts:18-22` takes explicit `tenant` param; `:26` `enabled: !!tenant?.id && !!characterId` |
| FE-10 | Tenant ID in query keys | PASS | `useCharacterLocation.ts:8-9` `detail: (tenantId, characterId) => ["character-location", tenantId, characterId]`; callers pass `tenant?.id` (`useCharacterLocation.ts:23`, `ChangeMapDialog.tsx:110`). On tenant switch `TenantProvider` also `queryClient.clear()`s |
| FE-11 | Error handling | PASS (no regression) | `ChangeMapDialog.tsx:119-153` uses manual `error instanceof Error` string-matching rather than `createErrorFromUnknown` — but this catch block is **pre-existing and unchanged** by the diff (task only swapped the service call inside the `try`). Always surfaces via `toast.error`. `CharacterMapCell` relies on React Query error state + `MapCell`'s own `.catch` fallback to "Unknown" |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `location.ts:8-12` `CharacterLocation { id: string; type; attributes }`; `attributes` holds `worldId/channelId/mapId/instance` (`:1-6`). Matches backend `character-locations` resource |
| FE-13 | Service pattern | PASS | `locationsService` uses the documented direct-client object pattern (consistent with `characters.service.ts`); not all services extend `BaseService`. PATCH wraps a proper JSON:API envelope (`locations.service.ts:13-20`) |
| FE-14 | Query key factory `as const` | PASS | `useCharacterLocation.ts:6-10` — `all` and `detail(...)` both `as const` |
| FE-15 | Forms use react-hook-form + zodResolver | WARN (pre-existing) | `ChangeMapDialog` is a hand-rolled `useState` form with a custom `validateMapId` (`:39-71`), not `useForm({ resolver: zodResolver })`. Pre-existing — the form scaffolding predates task-087 (diff only repointed read/write). Not introduced; non-blocking |
| FE-16 | Schema in `lib/schemas/` w/ inferred type | PASS (N/A) | No Zod schema added; `ChangeMapData`/`CharacterLocationAttributes` are plain JSON:API interfaces in `types/models/location.ts`, the correct home for transport types |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests for changed components | PASS | `locations.service.test.ts` (GET path + PATCH envelope asserted, `:19,:26-30`); `ChangeMapDialog.test.tsx` (write-via-location-endpoint `:74-78` + current-map-from-query `:81-84`). `CharacterMapCell` is a thin query-to-`MapCell` wrapper exercised transitively by the column/panel tests |
| FE-18 | Mocks updated when services changed | PASS | `ChangeMapDialog.test.tsx:12-33` mocks `useCharacterLocation` + `locations.service`; `AttributesPanel.test.tsx` / `useCharacters.test.tsx` fixtures dropped the removed `mapId` field |

## Summary

### Blocking (must fix)
- None. Build green, 740/740 tests green, zero FE-* regressions introduced by task-087.

### Non-Blocking (should fix — all PRE-EXISTING, not introduced by this task)
- **FE-06** `ChangeMapDialog.tsx:190,193` — hardcoded `red-500` classes; should be semantic `border-destructive` / `text-destructive`.
- **FE-11** `ChangeMapDialog.tsx:119-153` — manual `error instanceof Error` string-matching instead of `createErrorFromUnknown()`.
- **FE-15** `ChangeMapDialog.tsx:39-71` — hand-rolled `useState` validation instead of `react-hook-form` + `zodResolver`.

### Scope note
- The `useSeed`/`seed.service`/`SetupPage` changes in this diff range are an **unrelated `scope`-parameter revert** that landed on this branch; they are clean (tenant-guarded hooks, `JsonApiEnvelope` preserved) but outside the task-087 map-to-maps charter. Flagging for awareness — confirm they belong in this PR or were meant for a separate one.

**Verdict: PASS** — the map read/write relocation to the atlas-maps location endpoint is correctly typed (JSON:API), tenant-scoped (explicit param + `enabled` guard + tenant id in key), uses the service→hook→component layering, and is covered by new tests. The three non-blocking items are all pre-existing debt inside `ChangeMapDialog` that this task did not touch.


<!-- ============================================================= -->
<!-- PLAN ADHERENCE REVIEW (plan-adherence-reviewer)               -->
<!-- ============================================================= -->

# Plan Audit — task-087-change-map-to-maps

**Plan Path:** docs/tasks/task-087-change-map-to-maps/plan.md
**Audit Date:** 2026-06-12
**Branch:** task-087-change-map-to-maps
**Base Branch:** main (task fork point `6ea8fd05c`; 32 commits in range)

> **Diff-range note:** `git diff main..HEAD` is misleading — `main` advanced past
> the branch's fork point, so that range inverts unrelated newer-main changes
> (seed.service/SetupPage/useSeed) that task-087 never touched. The authoritative
> audit range is the 32 task-087 commits (`7c8bb5bde^..HEAD`). Files touched by
> those commits are exactly the plan's expected set plus four legitimate Task-6
> mapId-reader cleanups (`AttributesPanel.tsx`, `ApplyPresetDialog.tsx`,
> `EmptySlotTile.tsx`, `useCharacters.test.tsx`, all in commit `d625cdcb8`).

## Executive Summary

All 12 planned tasks are faithfully implemented with real code and tests — none
skipped or stubbed. The three deviation migrations documented in
`execution-notes.md` (atlas-channel, atlas-messages, atlas-pets, each reclassified
PASSIVE→ACTIVE) are confirmed present and correct. atlas-maps owns the warp write
via a single shared `warp.ChangeMap` reached by both the Kafka consumer and the
new `PATCH /characters/{id}/location` handler; map validation returns 400 and a
missing location row returns 404. Every Go module builds, vets, and tests clean
(sole exception: the pre-existing, unrelated `atlas-login socket/init.go:39` vet
warning that also exists on main). atlas-ui builds clean and 740/740 vitest tests
pass. **Verdict: READY_TO_MERGE.**

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-maps shared `warp.ChangeMap` + consumer delegates | DONE | `character/warp/processor.go:65`; consumer delegates `kafka/consumer/character/change_map.go:15,38`. Tests `processor_test.go:79`, `change_map_test.go:28`. Set-error logged at caller (commit `b0e3f5dd8`), functionally equivalent. |
| 2 | atlas-maps PATCH location write + map validation | DONE | `character/location/resource.go:35-37` (PATCH route), `:64` core: 404 `:66`, 400 `:76`, 204 `:92`. Import cycle broken via `WarpProvider` DI seam (`:23-26`, wired `main.go:128`). Tests `resource_test.go:37,59,78,97,113` (5, incl. 500 infra branches). |
| 3 | atlas-ui location service/type/hook | DONE | `services/api/locations.service.ts`, `types/models/location.ts`, `lib/hooks/api/useCharacterLocation.ts`; test `__tests__/locations.service.test.ts`. |
| 4 | atlas-ui ChangeMapDialog repointed | DONE | `ChangeMapDialog.tsx:23` reads hook, `:105` writes `locationsService.changeMap`. Render-time sync guard `syncedMapId` (`:34`) replaces planned useEffect — same effect. Test present. |
| 5 | atlas-ui table map column from per-row location | DONE | `CharacterMapCell.tsx` (hook-per-row, `—` fallback); wired `characters-columns.tsx`, `AttributesPanel.tsx`. |
| 6 | atlas-ui remove mapId from character type | DONE | `types/models/character.ts:9-39` — `mapId` gone; `UpdateCharacterData` = `{gm?}`. Stray readers cleaned (commit `d625cdcb8`). |
| 7 | atlas-parties full-field member from location | DONE | `location/requests.go`; `character/processor.go:270` `location.GetField` + world-only fallback; `model.go:164` `MapId()` now field-backed. |
| 8 | atlas-consumables summoning sack from location | DONE | `location/requests.go`; `consumable/processor.go:427,433,442` use `lf.MapId()`; no `c.MapId()` reads. |
| 9 | atlas-query-aggregator MapCondition from location | DONE | `location/requests.go`; `validation/model.go:399-407` reads `lf.MapId()`, map-0 fallback; nil-safe accessors (`f4d7b2996`). |
| 10 | Passive services strip dead mapId mirror | DONE | login/npc-shops/cashshop/messengers mirror removed from rest.go+model.go (grep clean). atlas-channel reclassified ACTIVE. |
| 11 | atlas-character retire shim (LAST) | DONE | `rest.go:71-116` Transform no longer emits mapId/instance; `Instance` dropped; `MapId` create-input only (`:42`). Dead `input.MapId != 0` branch removed. Commits `dfcc53946`/`fbce37a46` are the last code commits. |
| 12 | Full verification gate | DONE | All 12 Go modules build/vet/test clean (re-run in this audit); atlas-ui build clean + 740/740 vitest pass. |

**Completion Rate:** 12/12 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

> Plan checkboxes render `- [ ]` (unchecked); completion verified against git
> history and working tree, not checkbox state (cosmetic only).

## Deviation Migrations (execution-notes.md)

| Service | Reclassified | Status | Evidence |
|---|---|---|---|
| atlas-channel | PASSIVE→ACTIVE | DONE | `maps/location/resolve.go` `ResolveMapId`; used in `socket/writer/set_field.go:24`, `cash_shop_open.go:19`, `socket/handler/character_chat_whisper.go:66`, `kafka/consumer/character/consumer.go:121`, `kafka/consumer/session/consumer.go:181`. Mirror stripped. |
| atlas-messages | not audited→ACTIVE | DONE | Live inbound `field.Model f` used — `command/map/commands.go:109` (WhereAmI) and `:138` (rates) read `f.MapId()`. Character mirror fully removed. |
| atlas-pets | PASSIVE→ACTIVE | DONE | `location/requests.go`; `pet/processor.go:416` `location.GetField`, `:419` `GetBelow(f.MapId(),…)`. Mirror stripped + mapId dropped from sparse-fieldset request. |

## Invariant Verification

- **atlas-maps owns the warp write (shared method).** CONFIRMED — single
  `warp.ChangeMap`; consumer (`change_map.go:18`) and REST (`resource.go:85`)
  both call it.
- **Map validation 400 / no-row 404.** CONFIRMED — `resource.go:66` (404), `:76`
  (400); unit tests assert no `ChangeMap` call on either reject path.
- **No service reads a character-resource mapId mirror.** CONFIRMED — repo-wide
  sweep clean. Remaining character-package `MapId()` getters are only
  `atlas-maps/character/location/model.go:24` (owner) and
  `atlas-parties/character/model.go:164` (live registry, field-backed). atlas-
  character keeps `RestModel.MapId` create-input only (`rest.go:42`). Residual
  `f.MapId()` in consumables/channel/pets producers are on live `field.Model`
  (Kafka bodies), not the mirror; atlas-login `factory/rest.go:33` is creation
  POST input.
- **atlas-character shim removed LAST.** CONFIRMED — final code commits; GET no
  longer emits mapId/instance; mapId kept as create input.
- **atlas-ui repointing.** CONFIRMED — service/hook/type created, dialog + table
  repointed, mapId removed from character type.

## Build & Test Results

| Service | Build | Vet | Tests | Notes |
|---------|-------|-----|-------|-------|
| atlas-maps | PASS | PASS | PASS | new warp + location PATCH tests |
| atlas-parties | PASS | PASS | PASS | |
| atlas-consumables | PASS | PASS | PASS | |
| atlas-query-aggregator | PASS | PASS | PASS | |
| atlas-channel | PASS | PASS | PASS | |
| atlas-login | PASS | WARN | PASS | `socket/init.go:39` WaitGroup.Add — pre-existing on main, file untouched; changed `character` pkg vets clean |
| atlas-npc-shops | PASS | PASS | PASS | |
| atlas-cashshop | PASS | PASS | PASS | |
| atlas-messengers | PASS | PASS | PASS | |
| atlas-messages | PASS | PASS | PASS | |
| atlas-pets | PASS | PASS | PASS | |
| atlas-character | PASS | PASS | PASS | |
| atlas-ui | PASS | n/a | PASS | build clean (chunk-size advisories only); vitest 740/740. Run via Linux nvm node; login-shell `npm` resolves to a Windows install (`tsc not recognized`) — environment quirk, not a defect. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required for merge. Optional out-of-scope follow-ups:

1. Pre-existing `atlas-login socket/init.go:39` vet warning (on main) — fix in a
   separate task.
2. Plan.md checkboxes remain unchecked (cosmetic).

<!-- ============================================================= -->
<!-- BACKEND GUIDELINES AUDIT (backend-guidelines-reviewer)        -->
<!-- ============================================================= -->

# Backend Audit — task-087-change-map-to-maps

- **Scope:** 12 changed Go modules (diff `main..HEAD`, BASE=464e8c6e, HEAD=efcb6ef2)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-12
- **Build:** PASS (all 12 modules `go build ./...` exit 0)
- **Tests:** PASS (all 12 modules `go test ./... -count=1`, zero failures)
- **Overall:** NEEDS-WORK (build/tests green; one Important EXT-02 gap, two pre-existing non-blocking notes)

## Build & Test Results

All 12 changed modules build and test clean:
atlas-maps, atlas-parties, atlas-consumables, atlas-query-aggregator,
atlas-channel, atlas-pets, atlas-character, atlas-messages, atlas-messengers,
atlas-npc-shops, atlas-login, atlas-cashshop. `go vet` clean on the
highest-logic packages (atlas-maps/character/..., query-aggregator/validation
+ location). The pre-existing `atlas-login socket/init.go:39` WaitGroup vet
warning is on `main` (untouched file) and is NOT attributable to task-087.

## Domain Checklist — atlas-maps `character/location` (domain pkg, has model.go)

Note: task-087 only modified `resource.go` and added `resource_test.go` in this
package; `model.go`/`entity.go`/`administrator.go`/`provider.go`/`processor.go`
pre-existed (task-055).

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go / NewBuilder | PASS | `NewBuilder(characterId)` model.go:48 (lives in model.go, not a separate file — acceptable) |
| DOM-02 | ToEntity() | PASS | `func (m Model) ToEntity(tenantId)` model.go:34 |
| DOM-03 | Make(Entity) | PASS | `func Make(e entity) (Model, error)` entity.go:32 |
| DOM-04/05 | Transform / TransformSlice | PASS | rest.go:52, rest.go:64 |
| DOM-06 | Processor FieldLogger | PASS | `NewProcessor(l logrus.FieldLogger, ...)` processor.go:31 |
| DOM-07 | handlers pass d.Logger() | PASS | resource.go:51-53,99 (no StandardLogger) |
| DOM-08 | PATCH uses RegisterInputHandler[T] | PASS | `rest.RegisterInputHandler[RestModel]` resource.go:36 |
| DOM-09 | Transform errors handled | PASS | resource.go:110-115 checks err |
| DOM-10 | test DB tenant callbacks | N/A→PASS | pkg filters tenant explicitly via `tenant_id = ?` (provider.go:17, administrator.go:33) + `db.WithContext(p.ctx)` (processor.go:70/79/88); does NOT use GORM tenant callbacks, so `newTestDB` omitting `RegisterTenantCallbacks` is correct |
| DOM-11 | providers lazy | PASS | curried provider provider.go:12-21 |
| DOM-12 | no os.Getenv in handlers | PASS | grep clean |
| DOM-14/15 | no direct provider/db writes in handler | PASS | handler → warp.Processor → location.Set/administrator; no db.Create/Save/Delete in resource.go |
| DOM-17 | error→status mapping | PASS | 404 no-row (resource.go:66), 400 bad map (resource.go:77), 500 infra/warp (resource.go:71/80/87), 204 success (resource.go:92) |
| DOM-18 | JSON:API iface | PASS | GetName/GetID/SetID rest.go:22-39 |
| DOM-19 | flat request model | PASS | RestModel flat, no nested Data/Type/Attributes rest.go:13 |
| DOM-20 | table/case tests | PASS | resource_test.go 5 cases (happy/400/500-infra/500-warp/404) |
| DOM-21 | no atlas-constants dup | PASS | uses `field.Model`, `_map.Id`, `world.Id`, `channel.Id` directly |
| DOM-24 | Kafka producer stubbed in emit tests | PASS | warp `ChangeMap` emits via `message.Emit(p.pp)` (warp/processor.go:75); warp test injects capturing `producer.Provider` via `newProcessorWithDeps` (Pattern B), processor_test.go:92; no `producer.ResetInstance` cleanup; change_map_test injects `recordingWarp` mock (no emit), change_map_test.go:42 |

## Sub-task: atlas-maps `character/warp` (new shared processor)

| Item | Status | Evidence |
|------|--------|----------|
| Single authoritative warp shared by REST + Kafka | PASS | both `changeMapFromCommand` (change_map.go:18) and `changeCharacterLocation` (location/resource.go:85) call `warp.Processor.ChangeMap` |
| Import-cycle avoided via DI seam | PASS | `WarpProvider` injected from main.go:128-130; location pkg never imports warp |
| FieldLogger ctor, lazy emit | PASS | warp/processor.go:48 |
| Emit failure non-fatal (parity) | PASS | logged, not returned warp/processor.go:79,83 |

## External HTTP Client Checklist — `location` clients (5 packages)

The 4 newly-added clients (atlas-parties, atlas-consumables, atlas-pets,
atlas-query-aggregator) are byte-identical to one another (verified via `diff`,
all report IDENTICAL) and byte-identical to the pre-existing channel client
(`atlas-channel/.../maps/location/requests.go`). Per task scope, the verbatim
duplication is intentional and NOT flagged.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | relationship iface stubs | PASS | `SetToOneReferenceID`/`SetToManyReferenceIDs` no-ops, requests.go:52-53 (all 5) |
| EXT-02 | httptest integration test per client | **FAIL (Important)** | Only `atlas-channel/.../maps/location/requests_test.go` (pre-existing) and `atlas-character/.../location/requests_test.go` have httptest tests. The 4 NEW client packages — `atlas-parties/.../location/`, `atlas-consumables/.../location/`, `atlas-pets/.../location/`, `atlas-query-aggregator/.../location/` — ship **no test file at all**. Each `location/` dir contains only `requests.go`. |
| EXT-03 | 404 distinguished from other errors | PASS | `errors.Is(err, requests.ErrNotFound)` → `ErrNotFound`; else original error bubbles, requests.go:72-77 (all 5); channel `ResolveMapId` Warn-vs-Error split resolve.go:18-22 |
| EXT-04 | RootUrl(domain), no hardcoded DNS | PASS | `requests.RootUrl("MAPS")` requests.go:56 (all 5) |

## Behavioral-drift review (per-service location consumers)

| Service | Location consumption | Verdict |
|---------|----------------------|---------|
| atlas-parties | foreign-member field: uses resolved location, silent on ErrNotFound, Warn on infra (character/processor.go) | PASS — correct fallback |
| atlas-consumables | summoning sack: `location.GetField` failure ⇒ `ConsumeError` (hard fail) (consumable/processor.go) | PASS — strictest; correct for side-effecting spawn |
| atlas-pets | pet foothold: uses resolved map, skips on ErrNotFound, Warn on infra (pet/processor.go) | PASS |
| atlas-query-aggregator | MapCondition: `location.GetField` failure ⇒ actualValue 0 + Warn (validation/model.go:400-410); nil-safe `Logger()`/`Context()` accessors (validation/context.go:99-119) | PASS |
| atlas-channel | whisper + set_field resolve live via `location.ResolveMapId` (character_chat_whisper.go, set_field.go); `BuildCharacterData(c, bl, mapId)` param threaded (character_data.go) | PASS |
| atlas-messages | map/buff/disease commands read live `f.MapId()` from resolved field, not stale `character.MapId()` (command/map/commands.go) | PASS — no stale echo left |
| atlas-character | GET Transform drops MapId/Instance echo; `transformWithTemporal` sig change; MapId retained as create-input wired to `CreateAndEmit(..., input.MapId)` (rest.go, processor.go, resource.go:161). `atlas-character/location` import retained and still legitimately used by login flows (processor.go:391/425/1144/1194) — NOT dead code. | PASS |

## Security Review

Not an auth/token service. SEC-01..04 N/A. No hardcoded secrets, no `os.Getenv`
in changed handlers (grep clean).

## Summary

### Blocking (must fix)
- None. Build + full test suites pass across all 12 modules; no DOM/SUB/SEC
  check FAILs that block merge.

### Non-Blocking (should fix)
- **EXT-02 (Important):** The 4 new `location` client packages
  (atlas-parties, atlas-consumables, atlas-pets, atlas-query-aggregator) lack
  any test file. Add an httptest-backed integration test per package
  (happy-path JSON:API decode + 404→ErrNotFound + 5xx→non-ErrNotFound),
  mirroring `atlas-channel/.../maps/location/requests_test.go`. Mitigation:
  the client bodies are byte-identical to the channel client, whose unmarshal
  path IS covered, so real-world risk is low — but the guideline calls for
  per-package coverage and copy-paste drift would go uncaught.

### Pre-existing (out of scope — do NOT attribute to task-087)
- `atlas-maps character/location` has no standalone `builder.go` file and its
  `newTestDB` omits `RegisterTenantCallbacks`. Both pre-date task-087 (package
  filters tenant explicitly, so the callback is unnecessary). Not introduced by
  this change; informational only.
- `atlas-login socket/init.go:39` WaitGroup vet warning lives on `main`.
