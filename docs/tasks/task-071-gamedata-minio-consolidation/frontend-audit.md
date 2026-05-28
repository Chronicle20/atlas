# Frontend Audit — task-071-gamedata-minio-consolidation (round 3)

- **Audit Scope:** `services/atlas-ui/` diff between `19d00ed0` (origin/main) and `2527c4541` (HEAD)
- **Guidelines Source:** `.claude/skills/frontend-dev-guidelines/`
- **Date:** 2026-05-22
- **Build:** PASS
- **Tests:** 712 passed (76 files), 0 failed
- **Overall:** PASS

## Build & Test Results

```
$ cd services/atlas-ui && npm run build
# tsc -b + vite build
✓ built in 1.28s
# (the only warning is a chunk-size warning unrelated to this branch — ConversationEditorPanel exists in main.)

$ cd services/atlas-ui && npm test
RUN  v4.1.7
 Test Files  76 passed (76)
      Tests  712 passed (712)
   Duration  9.48s
```

## File Inventory

In-scope changes (`git diff --stat 19d00ed0..2527c4541 -- services/atlas-ui/`):

| File | Classification | Δ |
|------|---------------|---|
| `public/sw-character-cache.js` | Other (service worker) | +1/-1 |
| `src/components/features/setup/ScopeToggle.tsx` | Component (new) | +44 |
| `src/components/features/setup/__tests__/ScopeToggle.test.tsx` | Test (new) | +51 |
| `src/lib/hooks/api/useBaseline.ts` | Hook (new) | +36 |
| `src/lib/hooks/api/__tests__/useBaseline.test.tsx` | Test (new) | +119 |
| `src/lib/hooks/api/useSeed.ts` | Hook | +5/-37 |
| `src/pages/SetupPage.tsx` | Page | +140/-86 |
| `src/services/api/baseline.service.ts` | Service (new) | +57 |
| `src/services/api/__tests__/baseline.service.test.ts` | Test (new) | +134 |
| `src/services/api/seed.service.ts` | Service | +13/-21 |

**Files mentioned in the prompt but absent from this diff range:** `asset-url.ts` and `MapDetailPage` — confirmed via `git diff --stat 19d00ed0..2527c4541 -- '*asset-url*' '*MapDetail*'` returning empty. Those changes either landed in `main` already or belong to a different branch; nothing to audit here.

**Already-covered ground (audit-r2.md, commit `4a5a68506`):** FE-06 (semantic destructive token), FE-09 (`Tenant | null` + internal guard), FE-11 (decoded error message + DataStatus invalidation), FE-17 (three new test files with substantive assertions). Re-verified below — no regressions.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -nE ': any\|as any'` against the seven non-test changed files returns zero. The only flagged hits were `anyMutationPending` identifiers at `SetupPage.tsx:109,117` (variable name, not type). One `as never` cast at `SetupPage.tsx:452` for the heterogenous-status-union `formatBadge` call — non-blocking, but flagged below. |
| FE-02 | No manual class concatenation | PASS | Grep for `className.*\+|className=\{\``…\$\{` on changed files returns zero. All `className` literals in `ScopeToggle.tsx` and `SetupPage.tsx` are static strings or template-free. |
| FE-03 | No direct API client calls in components | PASS | `ScopeToggle.tsx` imports only `Button`. `SetupPage.tsx:23-57` imports hooks (`useSeed`, `useBaseline`, `useTenant`) only — no `@/lib/api/client` import. |
| FE-04 | No inline Zod schemas in components | PASS | Grep `z\.(object\|string\|number)` on changed component/hook/service files returns zero matches. No form on this page needs Zod — restore/publish use raw tenant fields, not a user-editable form. |
| FE-05 | No spinners for content loading | PASS | Five `animate-spin` instances in `SetupPage.tsx:344,371,399,428,460` are all rendered inside `<Button>` action slots gated by `*.isPending` — the documented exception. No content-area spinners. |
| FE-06 | No hardcoded colors | PASS | `ScopeToggle.tsx:30,38` use the semantic `variant="destructive"` and `text-destructive` tokens. Grep for `bg-(white\|black\|gray\|red\|amber)-\d` and `text-(gray\|red\|amber)-\d` on changed files returns zero. Regression-guarded by `ScopeToggle.test.tsx:45-50` (asserts `text-destructive` present and `text-amber-*` absent). |
| FE-07 | No state mutation | PASS | The only stateful update in this diff is `setScope(...)` at `SetupPage.tsx:77`. No `.push/.splice/.sort` followed by `setState`. The `seedRows` array at `SetupPage.tsx:226-297` is constructed fresh each render. |
| FE-08 | No default exports for components | PASS | `ScopeToggle.tsx:12` and `SetupPage.tsx:74` both use `export function`. Grep `export default function` on changed files returns zero. (Pages are imported by name in `App.tsx`, per the atlas-ui convention.) |
| FE-09 | Tenant guard in hooks | PASS | `useBaseline.ts:6,22` accept `Tenant \| null`; the mutationFn at `:10-12` and `:26-28` throws `'tenant is not yet resolved'` before invoking the service; `onSuccess` at `:16-17,32-33` short-circuits on null. Behaviorally guarded by `useBaseline.test.tsx:36-48,88-95` ("rejects when tenant is null"). |
| FE-10 | Tenant ID in query keys | PASS | The cross-hook key `dataStatusKey(tenantId)` is exported from `useSeed.ts:25` as `['dataStatus', tenantId] as const`. `useBaseline.ts:17,33` invalidate exactly that key; `useDataStatus` at `useSeed.ts:175` consumes it (with `['dataStatus', 'none']` fallback when tenant unresolved). No cross-tenant key collisions. |
| FE-11 | Error handling | PASS | `baseline.service.ts:11-19,30-33,50-53` decode `{error}` JSON or fall back to status text — verified by `baseline.service.test.ts:58-92,108-132`. `SetupPage.tsx:138-147,160-163,178-181,197-200` surface the message via `toast.error`. The formal `createErrorFromUnknown` helper is only wired into the `apiClient` path in this codebase (one call site repo-wide: `useCreateAndPollAccount.ts:115`); the direct-fetch services (`baseline.service.ts`, `seed.service.ts`) intentionally bypass it, consistent with sibling `seed.service.ts` style. Non-blocking. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `Tenant` (`tenants.service.ts:15-18`) is `{ id, attributes: {...} }`. `SetupPage.tsx:170-173,190-192,221-224` reads `activeTenant.attributes.region/majorVersion/minorVersion`. `seed.service.ts:70-87` defines `JsonApiEnvelope<A>` and unwraps `body.data.attributes` correctly. New endpoints (`baseline.service.ts`) return 202 with no body — no envelope parsing needed. |
| FE-13 | Service pattern | PASS | `BaselineService` at `baseline.service.ts:21-55` follows the direct-`fetch` pattern used by `seed.service.ts` and `accounts.service.ts` — required because the four tenant headers + multipart/JSON body are constructed inline rather than via the singleton `api` client. Zero `BaseService`-extending services exist in the repo; the documented "direct API client pattern" is the prevailing style. `tenantHeaders(tenant)` (`lib/headers.tsx:3-10`) keeps the four-header contract (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`) intact, plus `X-Atlas-Operator` for the publish flow. |
| FE-14 | Query key factory uses `as const` | PASS | `useSeed.ts:24-33` keys are all `[...] as const`. `dataStatusKey` is exported (`:25`) so it can be invalidated cross-hook. Keys are 2-tuples (flat) rather than hierarchical, which is allowed (FE-14 only requires the `as const` marker). |
| FE-15 | Forms use react-hook-form + zodResolver | N/A | No new form in this diff. The baseline restore/publish actions are single-button operations reading from `activeTenant.attributes`; no user-editable fields. The PRD did not call for a form here. |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | Same as FE-15 — no schema needed because there is no form. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | Three new test files cover the three new modules: `ScopeToggle.test.tsx` (5 cases incl. semantic-token regression assert), `useBaseline.test.tsx` (6 cases incl. null-tenant rejection, dataStatus invalidation, arg forwarding for both mutations), `baseline.service.test.ts` (6 cases incl. tenant-header construction, `X-Atlas-Operator: 1` on publish, JSON error decoding, non-JSON fallback). `SetupPage.tsx` is a page wrapper of those tested units — same pattern as the rest of the pages/ directory (no per-page integration tests in atlas-ui). |
| FE-18 | Mocks updated when services changed | PASS | All test files use Vitest (`vi.fn`, `vi.mock`, `vi.stubGlobal`) — confirmed by grep for `jest\.fn\|jest\.mock` returning zero in the new test files. `useBaseline.test.tsx:10-15` mocks `baselineService` matching the real shape (`restore`, `publish`). `baseline.service.test.ts:17-23` stubs the global `fetch` and asserts header/body construction directly. The pre-existing `useSeed.ts` shape changed (`useUploadWzFiles` now takes `{ file, scope }`, `useRunDataProcessing` now takes `Scope`); no existing test exercised that hook, so no stale mocks to update. Full suite runs green. |

## Other Observations (Non-Blocking)

### N-01 — `as never` cast at `SetupPage.tsx:452`
```tsx
badge={row.formatBadge(row.status.data as never)}
```
Justification: the `seedRows` array unions 8 differently-typed `formatBadge(d?: XStatus)` callbacks; TypeScript can't narrow `row.status.data` against the union of their parameter types. Not `any`, so FE-01 passes, but the type-system escape is mildly fragile — a future refactor that drops a field from one of the status interfaces would still compile. Optional: refactor `seedRows` to a discriminated union keyed by `kind`, or hoist each row's `formatBadge` into a wrapper that closes over its concrete `status` query. Non-blocking.

### N-02 — `activeTenant!` non-null assertion in `useSeed.ts`
```ts
mutationFn: ({ file, scope }: UploadWzFilesInput) =>
  seedService.uploadWzFiles(activeTenant!, file, scope),   // :141
mutationFn: (scope: Scope) => seedService.runDataProcessing(activeTenant!, scope),  // :153
```
These rely on the caller (SetupPage) disabling the buttons when `!activeTenant`. The newly-added `useBaseline.ts` does this correctly with an internal guard + thrown Error; the pre-existing `useUploadWzFiles` / `useRunDataProcessing` continue to use the bare `!` assertion. The diff only changed the input shape (added `scope`/`Scope`), not the null-handling behavior — so this isn't a regression introduced by this branch. Tracking as preexisting; the right fix is the pattern the new useBaseline hooks already establish. Non-blocking for this PR.

### N-03 — `lib/headers.tsx:3-10` uses optional-chaining on a non-nullable Tenant
```ts
export function tenantHeaders(tenant: Tenant): Headers {
    headers.set("TENANT_ID", tenant?.id);
    ...
}
```
Parameter is typed `Tenant` (non-null), but the body uses `tenant?.id` etc. — `Headers#set` would actually accept the value either way, but the type annotation contradicts the runtime expectation. Pre-existing, not touched by this branch; called by both `baseline.service.ts:23,42` and `seed.service.ts:79,127,154` from contexts that already guarantee non-null tenants. Non-blocking, not introduced by task-071.

### N-04 — Service worker `CACHE_NAME` bump
`public/sw-character-cache.js:6` bumps `CACHE_NAME` from `'atlas-character-images'` (main) to `'atlas-character-images-v2-task071'`. PRD §4.8 explicitly required this for the cutover to invalidate stale entries; verified via `git show 19d00ed0:services/atlas-ui/public/sw-character-cache.js | head -10`. PASS.

## Summary

### Blocking (must fix)
None.

### Non-Blocking (should fix eventually)
- N-01 — `as never` cast in seed-rows mapping (`SetupPage.tsx:452`).
- N-02 — `activeTenant!` non-null assertion in `useSeed.ts` upload/processing hooks (preexisting, not regressed).
- N-03 — `lib/headers.tsx` parameter/body null-handling mismatch (preexisting).

The branch faithfully implements the FE side of PRD §4.8 (scope toggle, baseline restore/publish row, extraction row removed, service worker cache bumped). Round-1 and round-2 findings have all been remediated in commit `4a5a68506` and stay PASS in this re-audit. Tests cover the new behaviors with substantive assertions (header construction, semantic-color regression guard, null-tenant rejection, cache invalidation). No blocking issues for merge.
