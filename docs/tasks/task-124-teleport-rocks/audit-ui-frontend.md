# Frontend Audit — task-124-teleport-rocks (UI list-editor delta)

- **Audit Scope:** atlas-ui files changed in commits `1f084b302..35030b125` (teleport-rock list editor feature)
- **Guidelines Source:** frontend-dev-guidelines skill + `services/atlas-ui/CLAUDE.md` (ground truth — the skill's Next.js references are stale; atlas-ui is Vite + React Router)
- **Date:** 2026-07-18
- **Build:** PASS (reported green by caller; not re-run)
- **Tests:** 1026/1026 passed (reported green by caller; not re-run). `tools/lint.sh --check --ui` exit 0 (reported green by caller; not re-run).
- **Overall:** NEEDS-WORK (build/tests green, but FAIL-level FE-* deviations exist)

## Build & Test Results

Not re-run per instructions — caller confirmed `npm run build`, `npm test` (1026/1026), and `tools/lint.sh --check --ui` all green prior to this audit. This audit is a static/manual review of the diff against FE-* guidelines; a couple of specific doubts were resolved by reading adjacent source (cited below), not by re-running the suite.

## File Inventory

- **Service:** `services/atlas-ui/src/services/api/teleport-rocks.service.ts` — JSON:API adapter (GET/POST/DELETE)
- **Service test:** `services/atlas-ui/src/services/api/__tests__/teleport-rocks.service.test.ts`
- **Hook:** `services/atlas-ui/src/lib/hooks/api/useTeleportRocks.ts` — query key factory + 1 query hook + 2 mutation hooks
- **Hook test:** `services/atlas-ui/src/lib/hooks/api/__tests__/useTeleportRocks.test.tsx`
- **Component:** `services/atlas-ui/src/components/features/characters/AddTeleportRockMapDialog.tsx` — searchable map-picker dialog
- **Component test:** `services/atlas-ui/src/components/features/characters/__tests__/AddTeleportRockMapDialog.test.tsx`
- **Component:** `services/atlas-ui/src/components/features/characters/TeleportRockListCard.tsx` — per-list card + `MapRow` subcomponent
- **Component test:** `services/atlas-ui/src/components/features/characters/__tests__/TeleportRockListCard.test.tsx`
- **Page (wiring only):** `services/atlas-ui/src/pages/CharacterDetailPage.tsx` — adds `useTeleportRockMaps` call + renders the two cards (no dedicated test file exists for this page, before or after this diff)

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ': any\|as any'` over all 9 in-scope files: zero matches |
| FE-02 | No manual class concatenation | PASS | `grep -n 'className={"'` over component files: zero matches; all `className` are plain string literals or `cn()`-free static strings (no violation since no conditional concat is attempted) |
| FE-03 | No direct API client calls in components | PASS | `grep -n '@/lib/api/client'` over `AddTeleportRockMapDialog.tsx`, `TeleportRockListCard.tsx`, `CharacterDetailPage.tsx`: zero matches; both components go through `teleportRocksService` via the hooks layer |
| FE-04 | No inline Zod schemas in components | PASS (N/A) | No `z.object(`/`from "zod"` in either component; feature has no structured form — `AddTeleportRockMapDialog.tsx` is a free-text search input with no validation, so no schema is warranted |
| FE-05 | No spinners for content loading | PASS | `grep -n 'animate-spin'` over all in-scope files: zero matches. Loading state in `AddTeleportRockMapDialog.tsx:82-84` is a plain "Searching…" text row, not a spinner |
| FE-06 | No hardcoded colors | **FAIL** | `TeleportRockListCard.tsx:67` — `className="h-6 w-6 hover:bg-red-100 hover:text-red-600"` on the per-row delete button. Raw Tailwind red instead of semantic `text-destructive`/`hover:bg-destructive/10` tokens (see anti-patterns.md #8: `bg-white text-gray-900` is the canonical bad example; `bg-red-100`/`text-red-600` is the same class of violation). Ignores dark-mode theming — the codebase's own destructive styling elsewhere is done via `variant="destructive"` or `text-destructive` (e.g. `components/ui/button.tsx` `buttonVariants`). |
| FE-07 | No state mutation | PASS | `grep -nE '\.push\(\|\.splice\(\|\.sort\('` over all in-scope non-test files: zero matches (the one `.sort(` hit in `CharacterDetailPage.tsx:129` is pre-existing code outside this diff's changed lines) |
| FE-08 | No default exports for components | PASS | `grep -n 'export default'` over both component files: zero matches; both are named exports (`export function AddTeleportRockMapDialog`, `export function TeleportRockListCard`) |
| FE-09 | Tenant guard in hooks | **PARTIAL FAIL** | `useTeleportRockMaps` (`useTeleportRocks.ts:22-33`) correctly takes an explicit `tenant` param and gates with `enabled: !!tenant?.id && !!characterId` (line 29) — matches Pattern A. But `useAddTeleportRockMap`/`useRemoveTeleportRockMap` (lines 43-59, 61-77) resolve tenant via `useTenant()` internally (lines 49, 67) purely to compute the `onSuccess` cache-write key — a third, mixed pattern not used elsewhere for a resource whose *read* hook takes an explicit param. Compare `useInventory.ts:216,233` (`useDeleteAsset`), where mutation variables carry an explicit `tenant: Tenant` field instead of deriving it from context, and `patterns-react-query.md`'s documented "Optimistic Update Pattern" does the same. Splitting the source of truth (explicit-param read key vs. context-derived write key) is currently harmless only because `CharacterDetailPage.tsx:58` happens to pass the same `activeTenant` object sourced from the same `useTenant()` call — any future caller passing a different (but valid) tenant reference would silently write the mutation result to a cache key the read hook never observes. |
| FE-10 | Tenant ID in query keys | **FAIL** | `useTeleportRocks.ts:18-19` — `teleportRockKeys.detail(tenantId: string | undefined, characterId)` returns `[...all, tenantId, characterId]` with no `?? 'no-tenant'` fallback, unlike the documented convention (anti-patterns.md #3) and 16 of the 50 existing hook files in `lib/hooks/api/` (e.g. `useBans.ts:23`, `useAccounts.ts:35`, `useConversations.ts:47`, `useQuests.ts:22`, `useGuilds.ts:36` all use `tenant?.id || "no-tenant"` / `?? "no-tenant"`). Not exploitable today because the query is `enabled`-gated on `!!tenant?.id`, but it is a real, checkable deviation from the established pattern. |
| FE-11 | Error handling with `createErrorFromUnknown` | **FAIL** | `AddTeleportRockMapDialog.tsx:46-51` and `TeleportRockListCard.tsx` `MapRow` (`52-57`) both catch mutation errors with a hand-rolled `error instanceof Error ? error.message : "An unexpected error occurred..."` instead of the codebase's `createErrorFromUnknown()` utility (`types/api/errors.ts`). Sibling components in the exact same directory use the shared utility: `MonsterBookWidget.tsx:45` — `toast.error(createErrorFromUnknown(collectionQuery.error).message)`; `ApplyPresetDialog.tsx:193` — `toast.error(createErrorFromUnknown(err).message)`. The new code still reaches `toast.error(...)` (so user feedback is not silently dropped) but bypasses the shared error-classification logic (network vs validation vs server errors) that `createErrorFromUnknown` provides. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `teleport-rocks.service.ts:14-18` — `TeleportRockResource { id: string; type: "teleport-rock-maps"; attributes: TeleportRockLists }` matches `{id, attributes}`. Defined locally in the service file rather than `types/models/`, but this matches established precedent — 35 of the service files under `services/api/` define resource interfaces locally (e.g. `monster-book.service.ts:12,18,28,33` — `CollectionResource`, `CardResource`, etc.), not just the 28 files under `types/models/`. Verified against the real backend shape too: `services/atlas-character/.../teleport_rock/rest.go:13-19` (`RestModel{Id, Regular, Vip, RegularCapacity, VipCapacity}`) is api2go-serialized, so GET/POST/DELETE all return the `{data: RestModel}` envelope — confirms the service's `unwrap()` comment (`teleport-rocks.service.ts:20-25`) is factually correct, not a guess. |
| FE-13 | Service extends `BaseService` (when applicable) | PASS | `teleport-rocks.service.ts:41` uses the plain object-literal "Direct API Client Pattern," which is the dominant style in this codebase (33 of 42 `services/api/*.ts` files export a bare object literal rather than a `class X extends BaseService`, confirmed via `grep -l "^export const .*Service = {"` = 33 vs `grep -l "^class .*Service"` = 9). No validation/transformation logic is needed for this resource, so `BaseService` is correctly not used. |
| FE-14 | Query key factory uses `as const` | PASS | `useTeleportRocks.ts:16-20` — both `all` and `detail(...)` end in `as const` |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | PASS (N/A) | No `react-hook-form`/`zodResolver` import anywhere in scope. `AddTeleportRockMapDialog.tsx` is a single free-text search field with no submit/validation semantics (selecting a result immediately fires the mutation) — not a "form" in the guideline's sense, so the pattern doesn't apply. |
| FE-16 | Schema in `lib/schemas/` with inferred type | PASS (N/A) | No Zod schema exists or is needed for this feature (see FE-04/FE-15) |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | **PARTIAL FAIL** | Service, hook, and both new components each have a dedicated test file (`teleport-rocks.service.test.ts`, `useTeleportRocks.test.tsx`, `AddTeleportRockMapDialog.test.tsx`, `TeleportRockListCard.test.tsx`) with real-behavior assertions (cache-key equality checks, mutation call args, capacity-badge text, disabled-at-capacity state) — good coverage of the units in isolation. Two gaps: (1) `CharacterDetailPage.tsx`'s wiring change (`+useTeleportRockMaps` call, `+20` lines rendering the two cards with the `regular`↔`regularCapacity` / `vip`↔`vipCapacity` prop mapping at lines 262-279) has zero test coverage — no `CharacterDetailPage.test.tsx` exists in the repo at all, so a future prop-mapping swap (e.g. `capacity={teleportRocksQuery.data.vipCapacity}` on the `list="regular"` card) would not be caught. (2) `TeleportRockListCard.test.tsx` never clicks the "Add" button, so the conditional-mount path `{open && <AddTeleportRockMapDialog .../>}` (`TeleportRockListCard.tsx:137-145`) and the `existingMapIds={maps}` prop it passes are never exercised by any test — nor is the `mutateAsync` rejection → `toast.error` path in either component test file. |
| FE-18 | Mocks updated when services changed | PASS (N/A) | No `__mocks__/` directories exist anywhere in `services/atlas-ui` (`find src -type d -name __mocks__` → empty); this codebase mocks inline per test file via `vi.mock(...)`, and both component test files correctly mock `@/lib/hooks/api/useTeleportRocks` and `@/services/api/teleport-rocks.service` at the point of use. |

## Non-Checklist Observation (informational, not a numbered FE item)

- `TeleportRockListCard.tsx:96-102` — the item-icon `<img>` omits `loading="lazy"`. `services/atlas-ui/CLAUDE.md` documents this as the required pattern for below-the-fold images ("Plain `<img>` at each call site with explicit `width`/`height` and `loading="lazy"` (below the fold)"), and the sibling component `MonsterBookWidget.tsx:222,264` follows it. This card renders at the very bottom of `CharacterDetailPage`, i.e. below the fold. Not one of the FE-* checklist items, included for completeness.

## Summary

### Blocking (must fix)
- **FE-06** — `TeleportRockListCard.tsx:67`: replace `hover:bg-red-100 hover:text-red-600` with semantic destructive tokens (e.g. `hover:bg-destructive/10 hover:text-destructive`).
- **FE-11** — `AddTeleportRockMapDialog.tsx:46-51` and `TeleportRockListCard.tsx:52-57` (`MapRow`): replace the manual `error instanceof Error ? ... : ...` fallback with `createErrorFromUnknown(error).message`, matching `MonsterBookWidget.tsx:45` / `ApplyPresetDialog.tsx:193`.
- **FE-10** — `useTeleportRocks.ts:18-19`: add the `?? 'no-tenant'` fallback to `teleportRockKeys.detail`'s `tenantId` segment, matching the 16 other hook files that already do this.
- **FE-09** — `useTeleportRocks.ts:43-59,61-77`: have `useAddTeleportRockMap`/`useRemoveTeleportRockMap` accept `tenant` as an explicit mutation-variable field (as `useInventory.ts`'s mutations do) instead of resolving it internally via `useTenant()`, so the read query key and the mutation's cache-write key share one source of truth rather than two.

### Non-Blocking (should fix)
- **FE-17** — Add a `CharacterDetailPage.test.tsx` case (or extend an existing one) asserting the `regular`/`vip` → `maps`/`capacity` prop wiring is not swapped.
- **FE-17** — Add a `TeleportRockListCard.test.tsx` case that clicks "Add" and asserts `AddTeleportRockMapDialog` mounts with the expected `existingMapIds`, plus an error-path (`mutateAsync` rejects → `toast.error` called) test in both component test files.
- Informational: add `loading="lazy"` to the icon `<img>` in `TeleportRockListCard.tsx:96-102` per `services/atlas-ui/CLAUDE.md`'s image convention.

**Overall verdict: NEEDS-WORK.** Build and the full test suite are green (per caller), and the feature is architecturally sound — correct JSON:API envelope handling (verified against the actual Go `api2go` handler), correct Rules-of-Hooks compliance via the `MapRow` subcomponent, correct query-key `as const` usage, and no `any`/mutation/default-export/direct-client-call violations. But four concrete, file:line-cited FE-* deviations (one visual/theming bug, one error-handling utility bypass, and two tenant/query-key convention deviations) should be fixed before merge, plus two non-blocking test-coverage gaps.
