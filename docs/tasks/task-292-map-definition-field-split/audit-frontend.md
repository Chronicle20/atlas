# Frontend Audit — task-292-map-definition-field-split

- **Audit Scope:** `git diff 9613e7259..9f6679907 -- services/atlas-ui` (41 files, +3627/-7). New service modules, hooks, fields locator + detail pages, Live Fields section, Map Objects tab, `docs/service-layer.md`.
- **Guidelines Source:** frontend-dev-guidelines skill (`.claude/skills/frontend-dev-guidelines/`), reconciled against the atlas-ui project `CLAUDE.md` which supersedes the skill's stale Next.js/Jest references — this app is Vite + React Router + Vitest, confirmed by `services/atlas-ui/CLAUDE.md`.
- **Date:** 2026-09-03
- **Build:** PASS (not re-run per instructions — reported green by controller: `tools/verify.sh` flagless, atlas-ui tests + build green)
- **Tests:** 2439/2439 passed across 291 files (reported by controller, not re-run)
- **Overall:** NEEDS-WORK

## Build & Test Results

Per task instructions, the full `tools/verify.sh` was already confirmed green on HEAD `9f6679907` (atlas-ui tests + build, 2439/2439 tests across 291 files, lint 0 errors) and was **not** re-run. This audit focuses on defects a green build/test run cannot see: design-choice correctness, tenancy discipline, cache-key isolation, pagination safety, and gaps in what the passing suite actually covers.

## File Inventory

- **Page:** `src/pages/FieldsPage.tsx` — fields locator (world/channel/map filters)
- **Page:** `src/pages/FieldDetailPage.tsx` — field detail (header/summary/tabs)
- **Page (touched):** `src/pages/MapDetailPage.tsx` — wires `LiveFieldsSection` + `MapObjectsTable`
- **Component:** `src/components/features/fields/FieldHeader.tsx`
- **Component:** `src/components/features/fields/FieldSummaryPanels.tsx`
- **Component:** `src/components/features/fields/FieldTabs.tsx`
- **Component:** `src/components/features/fields/FieldsFilterBar.tsx`
- **Component:** `src/components/features/fields/FieldsResultTable.tsx`
- **Component:** `src/components/features/fields/FieldCharactersTab.tsx`
- **Component:** `src/components/features/fields/FieldMonstersTab.tsx`
- **Component:** `src/components/features/fields/FieldObjectsTab.tsx`
- **Component:** `src/components/features/maps/LiveFieldsSection.tsx`
- **Component:** `src/components/features/maps/MapObjectsTable.tsx`
- **Component:** `src/components/features/maps/SurfaceKindBadge.tsx`
- **Component (touched):** `src/components/features/maps/MapDetailTabs.tsx`, `MapEntitySummary.tsx`, `MapHeader.tsx`
- **Hook:** `src/lib/hooks/api/useFields.ts`
- **Hook:** `src/lib/hooks/api/useFieldRuntime.ts`
- **Hook:** `src/lib/hooks/api/useWorlds.ts`
- **Hook (touched):** `src/lib/hooks/api/useMapEntities.ts` (added `useMapObjects`), `src/lib/hooks/api/useMaps.ts` (added `useMapNames`)
- **Service:** `src/services/api/fields.service.ts`
- **Service:** `src/services/api/worlds.service.ts`
- **Service:** `src/services/api/live-monsters.service.ts`
- **Service (touched):** `src/services/api/map-entities.service.ts` (added `getObjects`)
- **Other:** `src/App.tsx` (routes), `src/components/app-sidebar-items.ts`, `src/lib/breadcrumbs/routes.ts`, `services/atlas-ui/docs/service-layer.md`
- **Tests:** 11 new test files (`FieldCharactersTab`, `FieldMonstersTab`, `FieldObjectsTab`, `LiveFieldsSection`, `MapDetailTabs`, `MapObjectsTable`, `SurfaceKindBadge`, `useFields`, `FieldDetailPage`, `FieldsPage`, `fields-routes` breadcrumbs)

Note: this project is **not** Next.js/Jest — `src/pages/*.tsx` are React Router page components, not `app/*/page.tsx`; tests run on Vitest. The FE-* checklist below is applied against the actual stack per `services/atlas-ui/CLAUDE.md`.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -rn ': any\|as any'` across all new/touched files in scope returns zero matches. |
| FE-02 | No manual class concatenation | PASS | `grep -rn 'className={"'` across scope returns zero matches; all conditional classes go through `cn()`, e.g. `src/pages/FieldsPage.tsx:130`, `src/pages/FieldDetailPage.tsx:170`. |
| FE-03 | No direct API client calls in components | PASS | `grep -rn 'from "@/lib/api/client"'` across all new pages/components returns zero matches; all data access goes through `fieldsService`/`worldsService`/`liveMonstersService`/`mapEntitiesService` via hooks. |
| FE-04 | No inline Zod schemas in components | PASS | `grep -rn 'z\.object\|z\.string('` across scope returns zero matches. There are no forms in this change (filters are plain controlled inputs, not RHF/Zod-validated) — see FE-15/FE-16 below. |
| FE-05 | No spinners for content loading | PASS | Only two `animate-spin` matches, both on the Refresh action button's icon, not a content-loading gate: `src/pages/FieldsPage.tsx:130`, `src/pages/FieldDetailPage.tsx:170`. Content loading uses `PageLoader`/text/`Skeleton` (`FieldDetailPage.tsx:95`, `FieldSummaryPanels.tsx:52-55`). |
| FE-06 | No hardcoded colors | PASS | `grep -rnE 'bg-(white\|black\|gray-...)|text-gray-...'` across scope returns zero matches; semantic classes (`text-muted-foreground`, `text-destructive`) used throughout. |
| FE-07 | No state mutation | PASS | `FieldsFilterBar.tsx:44` sorts a fresh spread copy (`[...worlds].sort(...)`), not the prop array. `FieldSummaryPanels.tsx:31` pushes into a function-local `order` array built inside `useMemo`, never a mutation of props/state. |
| FE-08 | No default exports for components | PASS | `grep -rn 'export default function'` across scope returns zero matches; all new components/pages are named exports (e.g. `export function FieldsPage()` at `src/pages/FieldsPage.tsx:22`). |
| FE-09 | Tenant guard in hooks | PASS | Every new/touched query hook guards with `enabled: !!activeTenant` (or a compound including it): `useFields.ts:41,53`, `useFieldRuntime.ts:41,66`, `useWorlds.ts:22,35`, `useMapEntities.ts:27,38,51,64,77`, `useMaps.ts:129` (`useMapNames`). No hook in scope omits the guard. |
| FE-10 | Tenant ID in query keys | **N/A by design, verified correct** — see note below |
| FE-11 | Error handling with `createErrorFromUnknown` | PARTIAL — see Non-Blocking | All data fetching in scope goes through React Query hooks, surfaced via `.isError`/`.error` and rendered inline (`FieldsPage.tsx:165-167`, `FieldMonstersTab.tsx:34-44`, `FieldObjectsTab.tsx:43-53`, `ErrorDisplay` in `FieldDetailPage.tsx:98-107,109-118`) rather than manual `.catch()` blocks — `grep -n '\.catch('` across scope returns zero matches, so the anti-pattern (unhandled/ad-hoc `.catch`) does not apply. No toast usage in this change; all surfacing is inline error text, consistent with existing map-definition pages this branch extends (e.g. `MapDetailTabs`'s pre-existing error rendering pattern). Not a violation of the letter of FE-11, but see Non-Blocking note on `createErrorFromUnknown`. |

**FE-10 detail (query-key tenant isolation — deliberate deviation, verified sound):** `fieldKeys`, `fieldRuntimeKeys`, and `worldKeys` deliberately omit tenant from their key tuples. This is not the anti-pattern the checklist is written against (accidental omission causing cross-tenant cache bleed) — it is a documented, verified-correct substitute:
- `src/context/tenant-context.tsx:68` — the tenant-change effect calls `queryClient.clear()` (per `docs/service-layer.md:160-162` and the effect's own doc comment referenced there), which wipes **every** cache entry, including all `fields`/`fieldRuntime`/`worlds` keys, on every tenant switch. A key without a tenant segment cannot leak stale data across tenants because the whole cache is discarded first.
- Every hook that uses these keys also carries `enabled: !!activeTenant` (FE-09 above), so no query fires before a tenant exists, closing the other half of the risk (a query firing with `tenant: null` and getting cached under a `'no-tenant'` bucket that later collides).
- `mapEntityKeys` (`useMapEntities.ts:12-18`) and `mapKeys` (`useMaps.ts:28-38`, pre-existing) follow the **same** no-tenant-in-key convention for definition data, so this branch's fields/worlds hooks are consistent with the existing map hooks they sit beside, not a new deviation.
- Verified the mechanism is real, not just a comment: `src/context/tenant-context.tsx:68` was read directly and does call `queryClient.clear()` inside the tenant-change effect (per the `docs/service-layer.md:157-162` description, which quotes the exact line number and was corroborated against the hooks' own guard comments in `useFields.ts:9-12` and `useFieldRuntime.ts:12-16`).

Verdict: **PASS** — the convention is correctly and completely applied across every new hook in scope; no hook in this diff includes tenant in a query key nor omits the `enabled` guard.

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `FieldData` (`fields.service.ts:3-13`), `WorldData`/`ChannelData` (`worlds.service.ts:3-37`), `LiveMonsterData` (`live-monsters.service.ts:15-42`), `MapObjectData` (`map-entities.service.ts:47-62`) all follow `{ id: string, type: string, attributes: {...} }`. |
| FE-13 | Service pattern | PASS | `services/atlas-ui/docs/service-layer.md:5` documents there is **no** `BaseService` in this codebase (removed in task-004) — every service is a plain class/object calling `api.*` directly. `FieldsService`, `WorldsService`, `LiveMonstersService` (all new) and `MapEntitiesService` (touched) follow this documented direct-client pattern consistently, e.g. `fields.service.ts:31-57`. This is the project's actual, current pattern, not the skill's stale `BaseService`-preferred description — the project doc takes precedence per its own explanation of the task-004 migration. |
| FE-14 | Query key factory uses `as const` | PASS | `fieldKeys` (`useFields.ts:13-16`), `fieldRuntimeKeys` (`useFieldRuntime.ts:17-22`), `worldKeys` (`useWorlds.ts:11-15`), `mapEntityKeys` (`useMapEntities.ts:12-18`) all use `as const` on every key-returning arrow/property. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | **N/A** | No form exists in this diff. `FieldsFilterBar.tsx` is a set of plain controlled `Select`/`Input` components with no submission or validation — it's a live filter, not a form, matching the documented "Filter/search page" pattern in `docs/service-layer.md:186` ("read `?q=…` from `useSearchParams`... make the URL the single source of truth. No `autoSearched` ref, no initial-load `useEffect`"), which `FieldsPage.tsx:81-92` follows for the `?map=` param. |
| FE-16 | Schema in `lib/schemas/` with inferred type | **N/A** | No Zod schema is introduced by this diff (consistent with FE-04/FE-15 — no form/validation surface exists in the changed files). |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | **FAIL (partial)** | Most components have direct or page-integration coverage (`FieldCharactersTab.test.tsx`, `FieldMonstersTab.test.tsx`, `FieldObjectsTab.test.tsx`, `LiveFieldsSection.test.tsx`, `MapObjectsTable.test.tsx`, `SurfaceKindBadge.test.tsx`, `MapDetailTabs.test.tsx`, plus `FieldsPage.test.tsx`/`FieldDetailPage.test.tsx` exercising `FieldHeader`, `FieldSummaryPanels`, `FieldTabs`, `FieldsFilterBar`, `FieldsResultTable` indirectly). **However**, none of the four new/touched service modules (`fields.service.ts`, `worlds.service.ts`, `live-monsters.service.ts`, `map-entities.service.ts`'s new `getObjects`) has a dedicated unit test, and no test anywhere in the diff asserts on the actual `page[size]`/`filter[...]` query-string construction. Verified by grep: `grep -rn "page\[size\]\|PAGE_SIZE\|250" src/lib/hooks/api/__tests__/useFields.test.tsx src/pages/__tests__/FieldsPage.test.tsx src/pages/__tests__/FieldDetailPage.test.tsx src/components/features/maps/__tests__/LiveFieldsSection.test.tsx` returns zero matches. `useFields.test.tsx:41-45` mocks `fieldsService` at the module boundary (`fieldsService.getFields = vi.fn()`), so the pagination logic *inside* `fields.service.ts` — the exact fix for the stated prior defect ("at least one earlier defect on this branch was a silently-truncating unpaginated request") — is never exercised by any test in this diff. This is the one area the task asked to be scrutinized and it has no regression test. |
| FE-18 | Mocks updated when services changed | PASS | No shared `__mocks__/` directory changes were needed — each new test file mocks its own service module inline (e.g. `useFields.test.tsx:41-45` mocks `@/services/api/fields.service`), consistent with the project's per-test mocking convention (no centralized `jest.mock`/`vi.mock` shim files in this codebase for these services). |

## Pagination — specific finding

`fields.service.ts:21-24` and `live-monsters.service.ts:44-48` both explicitly set `page[size]=250` with an inline comment explaining the backend's default-250 truncation trap — this is the fix for the stated prior defect and it is present and correctly applied to both new list-fetching service methods (`getFields`, `getFieldCharacters`, `getMonsters`).

`worlds.service.ts:40-46` (`getWorlds`, `getChannels`) does **not** set an explicit `page[size]`, unlike its sibling services in the same PR. In practice this is very unlikely to truncate (a tenant realistically has single-digit worlds and channels-per-world, both far under 250), so this is **not** a blocking defect, but it is an inconsistency worth flagging: the branch established an explicit-page-size convention specifically to avoid silent truncation, and `worlds.service.ts` doesn't follow it. `map-entities.service.ts`'s new `getObjects` (line 105-107) also has no explicit page size, but it exactly mirrors the four pre-existing sibling methods on the same class (`getPortals`, `getNpcs`, `getReactors`, `getMonsters`, none of which set `page[size]` either) — that is consistency with pre-existing, out-of-scope code, not a new gap introduced by this branch.

## Cache-profile duplication — judgment call

The `RUNTIME_STALE_TIME`/`RUNTIME_GC_TIME` constants are declared independently in `useFields.ts:18-19` and `useFieldRuntime.ts:24-25` (both `5 * 1000` / `60 * 1000`), and the doc explicitly flags this at `docs/service-layer.md:118-121`: "declared independently... they're duplicated, not shared from a common constant, so keep them in sync by hand if either changes." This is a real, acknowledged maintenance risk (a future edit to one and not the other silently desyncs the runtime cache profile), but it is disclosed, small in scope (two file pairs, four numeric literals), and documented at the exact location a future editor would look. Judgment: **acceptable, not a finding** — the guidelines don't mandate a shared-constants module for stale-time values, and the explicit doc comment is the mitigation the guidelines' spirit (favor legibility, avoid silent drift) actually asks for. Would be worth a one-line shared-constant extraction as a low-cost follow-up, but doesn't rise to a checklist violation.

## Loading-state distinction (`FieldMonstersTab`/`FieldObjectsTab` vs `MapObjectsTable`)

Confirmed the reported pattern directly:
- `MapObjectsTable.tsx:21-23` distinguishes `objects === undefined` (still loading, shows "Loading objects...") from `objects.length === 0` (resolved empty, shows "No named objects on this map.") — three-way branch: error / loading / empty / populated.
- `FieldMonstersTab.tsx:46` collapses loading and empty into one branch: `if (!monsters || monsters.length === 0)` — `monsters === undefined` (query still in flight, since `FieldDetailPage.tsx` does not gate rendering of `FieldMonstersTab` on `monstersQuery.isLoading`) renders the same "No monsters are currently in this field." text as a genuinely resolved empty result.
- `FieldObjectsTab.tsx:55-71` has the same collapse: `defined` being `undefined` (loading) produces `untrackedObjects.length === 0`, indistinguishable from a resolved empty definition list, so the false "No map objects are declared or tracked for this field." on first paint reported by the task is confirmed.

Per the anti-patterns doc (`anti-patterns.md` #7, "Spinner for Content Loading") and the components doc's "Empty State Pattern" (`patterns-components.md:281-293`, `{data.length === 0 && !loading && (...)}`), the guidelines' own reference implementation for an empty state explicitly gates on `!loading` — i.e., the documented pattern is exactly the three-way branch `MapObjectsTable.tsx` uses, and exactly what `FieldMonstersTab`/`FieldObjectsTab` skip. So this is not merely a UX nicety outside the guidelines' scope — the guidelines' own canonical empty-state example is the pattern being violated. I read this as a genuine (if narrow) FE-05-adjacent guideline gap, not just a design nit: a false "No monsters/objects" message on first paint is a content-loading misrepresentation, the same category of harm FE-05 legislates against (loading state must not present as a final/empty state). Given it's already tracked on the requester's list, I'm not adding it as a new blocking item, but I concur it is a real, guideline-grounded defect rather than optional polish.

## Not evaluable from the diff

- FE-11 (`createErrorFromUnknown`) — whether the *existing* map-definition pages this branch extends (`MapDetailTabs.tsx`, `MapEntitySummary.tsx`) use `createErrorFromUnknown` anywhere in their pre-existing (out-of-scope) code paths was not checked; only the newly-added lines were read. If the established pattern for this page already uses inline React Query error rendering without `createErrorFromUnknown` for read-only queries, the new fields/field-runtime code is simply consistent with it. Would need to read the full pre-existing `MapDetailTabs.tsx`/`MapEntitySummary.tsx` error paths to confirm this is the established convention rather than a gap.
- Toast usage — whether any write/mutation path exists anywhere adjacent to this feature (none was found in the diff — all new hooks are read-only `useQuery`, no `useMutation`) was confirmed absent, so toast-on-error (FE-11's other half) doesn't apply; not independently verified beyond grep for `useMutation` returning zero in the diffed files.
- Backend pagination cap behavior (`paginate.MaxPageSize` = 250) — taken on faith from the in-code comments (`fields.service.ts:21-23`, `live-monsters.service.ts:44-47`); the Go backend source for `paginate.MaxPageSize` was not read to confirm the cited default/cap values are accurate.

## Summary

### Blocking (must fix)
- FE-17 — no test in the diff exercises `fields.service.ts`'s or `live-monsters.service.ts`'s pagination construction (`page[size]=250`, filter params) at the service layer; the fix for the branch's own previously-identified silent-truncation defect ships without a regression test. `src/services/api/fields.service.ts:21-24,43-56`, `src/services/api/live-monsters.service.ts:44-61`, confirmed untested via `grep -rn "page\[size\]\|PAGE_SIZE\|250"` across all new hook/page/component test files (zero matches).

### Non-Blocking (should fix)
- Pagination inconsistency: `worlds.service.ts:40-46` (`getWorlds`, `getChannels`) doesn't set an explicit `page[size]` unlike its sibling `fields.service.ts`/`live-monsters.service.ts` in the same PR, despite the PR's own stated rationale for doing so elsewhere. Low practical risk (world/channel counts are small) but inconsistent with the convention this branch itself established.
- `FieldMonstersTab.tsx:46` and `FieldObjectsTab.tsx:61` render a false "no data" empty state while the query is still loading (already on requester's tracked list) — confirmed against the guidelines' own canonical empty-state pattern (`patterns-components.md:281-293`), which gates on `!loading`; `MapObjectsTable.tsx:21` shows the correct three-way branch already present in this same PR.
- Cache-profile stale/gc-time constants duplicated between `useFields.ts:18-19` and `useFieldRuntime.ts:24-25` — disclosed and documented (`docs/service-layer.md:118-121`), judged acceptable, but a one-line shared-constant extraction would remove the manual-sync risk entirely.
