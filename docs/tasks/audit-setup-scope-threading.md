# Frontend Audit — fix/setup-status-scope-threading

- **Audit Scope:** 5 changed TS/TSX files between 3d5e406 and 59e4ad0
- **Guidelines Source:** frontend-dev-guidelines skill
- **Date:** 2026-06-12
- **Build:** PASS (`npm run build` — tsc -b + vite build clean; only the pre-existing chunk-size warning)
- **Tests:** 743 passed, 0 failed (full suite, 78 files). Changed files alone: 45 passed.
- **Overall:** PASS

## Build & Test Results

- `npm run build`: BUILD_EXIT=0. Output ends with `built in 1.32s`. The only warning is the pre-existing "chunks larger than 500 kB" notice (ConversationEditorPanel/index), unrelated to this diff.
- `npx vitest run` (full): `Test Files 78 passed (78)`, `Tests 743 passed (743)`.
- `npx vitest run` (changed files only): `Test Files 2 passed (2)`, `Tests 45 passed (45)`.

## File Inventory

- `src/services/api/seed.service.ts` — Service
- `src/lib/hooks/api/useSeed.ts` — Hook
- `src/pages/SetupPage.tsx` — Page
- `src/services/api/__tests__/seed.service.test.ts` — Test (service)
- `src/lib/hooks/api/__tests__/useSeed.test.tsx` — Test (hook)

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | grep for `: any`/`as any`/`as never` over all 3 source files: only hit is the word "as never" inside a comment at SetupPage.tsx:218. No real `any`. |
| FE-02 | No manual class concatenation | PASS | No className edits in this diff; SetupPage uses static class strings only (e.g. SetupPage.tsx:309). |
| FE-03 | No direct API client in components | PASS | SetupPage.tsx imports only hooks from `@/lib/hooks/api/useSeed` (SetupPage.tsx:22-43); no `@/lib/api/client` import. Raw `fetch` lives in the service layer (seed.service.ts:110), which is the established pattern for this module. |
| FE-04 | No inline Zod schemas | PASS | No `z.` usage in any changed file. |
| FE-05 | No spinners for content loading | PASS | `animate-spin` appears only on submit/action buttons (SetupPage.tsx:353,380,408,437,469); status badges render text, not spinners. Not touched by this diff. |
| FE-06 | No hardcoded colors | PASS | No new color classes; existing classes are semantic (`text-muted-foreground`, SetupPage.tsx:312). |
| FE-07 | No state mutation | PASS | No array/object mutation introduced; hooks return immutable query state. |
| FE-08 | No default exports for components | PASS | `export function SetupPage()` (SetupPage.tsx:64); hooks use named exports (useSeed.ts:164,175). |
| FE-09 | Tenant guard in hooks | PASS | `useWzInputStatus`/`useDataStatus` both call `useTenant()` and set `enabled: !!activeTenant` (useSeed.ts:165,169 and 176,181). queryFn uses `activeTenant!` only when enabled. |
| FE-10 | Tenant ID in query keys | PASS | Keys are `[...wzInputStatusKey(activeTenant.id), scope]` = `['wzInputStatus', tenantId, scope]` (useSeed.ts:167,178). Tenant id is segment 2; scope is segment 3. Null-tenant fallback also keyed `['wzInputStatus','none',scope]`. |
| FE-11 | Error handling | PASS (consistent with module) | The status reads throw on `!response.ok` (seed.service.ts:111-113); React Query surfaces the error via `isError`. This matches the module's existing pattern; the page's mutation paths use `toast.error` (SetupPage.tsx:128-137,150-152). No regression. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `fetchJsonApi` reads `body.data.attributes` from the `JsonApiEnvelope<A>` (seed.service.ts:89-95,114-115). Tests assert the envelope shape (seed.service.test.ts:39-43). |
| FE-13 | Service pattern | PASS | `SeedService` is the documented direct-`fetch` adapter for the seeder endpoints (pre-existing). Scope is threaded without changing that pattern. |
| FE-14 | Query key factory `as const` | PASS | `wzInputStatusKey`/`dataStatusKey` return `as const` (useSeed.ts:24-25). Spread into the scoped key preserves tuple typing. |
| FE-15 | Forms use rhf+zodResolver | N/A | No forms in scope. |
| FE-16 | Schema with inferred type | N/A | No schemas in scope. |

## React Query correctness (focus area)

- **Cache key by scope — CORRECT.** Adding `scope` as segment 3 (useSeed.ts:167,178) means toggling tenant↔shared produces a distinct cache entry and triggers a refetch. Verified by test: shared write populates `['wzInputStatus','tenant-1','shared']` and leaves `['wzInputStatus','tenant-1','tenant']` undefined (useSeed.test.tsx:160-161).
- **Invalidation prefix-match — CORRECT.** The mutations (`useUploadWzFiles`, `useRunDataProcessing`) invalidate with the 2-element key `wzInputStatusKey(activeTenant.id)` / `dataStatusKey(activeTenant.id)` (useSeed.ts:144,156). React Query `invalidateQueries` defaults to prefix matching, so a 2-element prefix invalidates every 3-element scoped variant. The inline comment at useSeed.ts:161-163 states this intent and it holds.
- **No stale-scope leak.** `staleTime: 0` + `refetchInterval: 5000` are unchanged (useSeed.ts:170-171,181-182); scoped keys still poll independently.

## Multi-tenancy / header handling (focus area)

- `fetchJsonApi` sends the four tenant headers via `tenantHeaders(tenant)` (seed.service.ts:102) and conditionally adds `X-Atlas-Operator: 1` only for `scope === 'shared'` (seed.service.ts:107-109) — matching the write path (uploadWzFiles seed.service.ts:171-173, runDataProcessing seed.service.ts:199-201).
- Query param `?scope=` is appended on both reads (seed.service.ts:213,217), fixing the original bug where status defaulted to the tenant prefix.
- Tests assert both halves: tenant scope → `/api/data/wz?scope=tenant`, no operator header (seed.service.test.ts:262-270); shared scope → `?scope=shared` + `X-Atlas-Operator: 1` (seed.service.test.ts:272-282, 284-294). These are real assertions on URL and header, not tautologies.

## Test quality (focus area)

- Hook tests exercise real behavior: assert the service is called with `(fakeTenant, 'shared')` / `(fakeTenant, 'tenant')` and assert the resulting cache key segment (useSeed.test.tsx:158-161, 178-188). The default-scope case (no arg) is covered (useSeed.test.tsx:165-188).
- Service tests stub `fetch` and assert URL + header presence/absence — genuine assertions (seed.service.test.ts:248-294).
- Mock surface updated to add `getWzInputStatus`/`getDataStatus` (useSeed.test.tsx:38-39), consistent with FE-18.

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests for changed code | PASS | New describe blocks for both hooks (useSeed.test.tsx:148-190) and both service reads (seed.service.test.ts:233-294). |
| FE-18 | Mocks updated | PASS | Service mock gained the two new methods (useSeed.test.tsx:38-39). |

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- **Minor (consistency, not a guideline violation):** `seed.service.ts` repeats the inline union `'tenant' | 'shared'` (seed.service.ts:100,212,216) instead of importing the exported `Scope` type from `@/components/features/setup/ScopeToggle` (`export type Scope = 'tenant' | 'shared'`, ScopeToggle.tsx:3). The hook layer already imports `Scope` (useSeed.ts:21). Types are structurally identical, so there is no type-safety gap; centralizing the alias would prevent drift if a third scope is ever added. Note the existing `uploadWzFiles`/`runDataProcessing` in the same file already use the inline union, so this matches local precedent.

### Overall
PASS — build clean, full suite green, every applicable FE-* check passes. The fix correctly threads scope through both status reads (query param + operator header) and adds scope as a cache-key segment while keeping invalidation on the 2-element prefix so it matches all scopes.
