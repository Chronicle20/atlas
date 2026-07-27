# Frontend Audit — Character Rankings Leaderboard (atlas-ui)

- **Audit Scope:** atlas-ui diff only, `aacbd6569..aba3d894b` (HEAD)
- **Guidelines Source:** `.claude/skills/frontend-dev-guidelines/` + `services/atlas-ui/CLAUDE.md` (local architecture override — Vite + React Router, not Next.js App Router)
- **Date:** 2026-07-24
- **Build:** PASS (`npm run build`, node v22.22.2 via nvm — no errors; produced a distinct `RankingsPage-*.js` chunk confirming the lazy route split)
- **Tests:** 6 passed, 0 failed (`npx vitest run` scoped to the 4 new test files: `rankings.service.test.ts`, `useRankings.test.tsx`, `LeaderboardRow.test.tsx`, `RankingsPage.test.tsx`)
- **Lint:** clean (`npx eslint` on all 8 new/changed files — no output)
- **Overall:** NEEDS-WORK (build/tests/lint clean; FE-06 hardcoded-color violations present in new code)

## Files Audited

- `services/atlas-ui/src/services/api/rankings.service.ts` (Service) + `__tests__/rankings.service.test.ts`
- `services/atlas-ui/src/lib/hooks/api/useRankings.ts` (Hook) + `__tests__/useRankings.test.tsx`
- `services/atlas-ui/src/components/features/rankings/LeaderboardRow.tsx` (Component) + `__tests__/LeaderboardRow.test.tsx`
- `services/atlas-ui/src/pages/RankingsPage.tsx` (Page) + `__tests__/RankingsPage.test.tsx`
- `services/atlas-ui/src/App.tsx` (route registration, lazy import)
- `services/atlas-ui/src/components/app-sidebar-items.ts` (nav entry)

**Note on baseline:** `services/atlas-ui/CLAUDE.md` documents this app as Vite + React Router (`src/pages/*.tsx`, `App.tsx` routes, Vitest), which supersedes the generic skill's Next.js App-Router description where they conflict. Findings below are evaluated against the Vite-era conventions and the closest existing precedent in the repo, `MarketplacePage.tsx` / `mts-listings.service.ts` / `useMtsListings.ts`, which implement the same "direct apiClient + tenant-scoped query key" pattern already accepted into the codebase.

## Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ": any\|as any"` over all 4 source + 4 test files returned zero matches |
| FE-02 | No manual class concatenation | PASS (N/A) | No `className={"…" +` or template-literal concatenation in any of the 4 files; all `className` values are static strings — no conditional classes needed, so `cn()` is correctly unused rather than bypassed |
| FE-03 | No direct API client calls in components/pages | PASS | `apiClient` is imported only in `services/api/rankings.service.ts:1` (service layer); `LeaderboardRow.tsx` and `RankingsPage.tsx` import only hooks/services |
| FE-04 | No inline Zod schemas in components | PASS (N/A) | No `z.object`/`z.string` etc. anywhere in the diff — this feature has no form |
| FE-05 | No spinners for content loading | WARN | `RankingsPage.tsx` has no `query.isLoading` branch at all (lines 105–125 go straight from filter controls to the `isError` ternary to the table) — no illegal `animate-spin` was added, but the guideline's "use Skeleton for loading" is also not honored: on first fetch the table renders with headers and an empty body with no loading affordance. This exactly mirrors `MarketplacePage.tsx`, which also has zero `isLoading`/`Skeleton` handling (`grep -n "isLoading\|Skeleton" src/pages/MarketplacePage.tsx` → no matches) — pre-existing precedent, not a new anti-pattern, but still non-compliant with the checklist item |
| FE-06 | No hardcoded colors | **FAIL** | `LeaderboardRow.tsx:15` `text-green-600`; `LeaderboardRow.tsx:18` `text-red-600`; `RankingsPage.tsx:106` `text-red-600`. None use semantic tokens (`text-destructive`, etc.) per `patterns-styling.md`. Mitigating context: the identical anti-pattern already exists in ~8 other files (`CharacterDetailPage.tsx:241`, `OptimizedCharacterRenderer.tsx:114,124`, `ChangeMapDialog.tsx:224`, `ChangeGmDialog.tsx:152`, `ServiceTypeBadge.tsx:19`, `accounts-columns.tsx:144`, `LoginHistoryPage.tsx:269,277`), so this PR follows (rather than invents) a codebase-wide convention. Still a literal FE-06 violation in the new code |
| FE-07 | No state mutation | PASS | `RankingsPage.tsx` uses `setWorldId`/`setJobCategory`/`setPage(p => Math.max(0, p - 1))` — no `.push`/`.splice`/`.sort` on state anywhere in the diff |
| FE-08 | No default exports for components | PASS | `export function LeaderboardRow` (`LeaderboardRow.tsx:25`), `export function RankingsPage` (`RankingsPage.tsx:27`), `export const rankingsService` (`rankings.service.ts:62`), `export function useRankings` (`useRankings.ts:28`) — all named. `App.tsx` imports `RankingsPage` by name via `.then((m) => ({ default: m.RankingsPage }))` |
| FE-09 | Tenant guard in hooks | PASS | `useRankings(tenantId, worldId, filter, enabled = true)` (`useRankings.ts:28-32`) takes an explicit `enabled` param threaded to `useQuery`; the only call site, `RankingsPage.tsx:38-43`, passes `!!activeTenant`. `LeaderboardRow.tsx:33` calls `useCharacter(activeTenant!, …)`, whose own `enabled: !!tenant?.id && !!characterId` guard lives in `useCharacters.ts:90` — identical to the existing `CharacterDetailPage.tsx:55` call site |
| FE-10 | Tenant ID in query keys | PASS | `rankingsKeys.leaderboard = (tenantId, worldId, filter) => [...all, "leaderboard", tenantId, worldId, filter] as const` (`useRankings.ts:19-21`) — tenant id is a discriminator segment, matching the already-merged `mtsListingsKeys.browse` (`useMtsListings.ts:12-21`) byte-for-byte in structure |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS (N/A) | `grep -n "\.catch(\|console\.\|toast\."` over all 4 source files returned zero matches — no manual async `.catch()` was introduced; errors flow through React Query's `isError` state, surfaced inline in `RankingsPage.tsx:106` |
| FE-12 | JSON:API model shape | PASS | `RankingEntry { id: string; attributes: RankingEntryAttributes }` (`rankings.service.ts:28-31`) |
| FE-13 | Service extends `BaseService` or documented direct-client pattern | PASS | `rankingsService` uses the direct-`apiClient` pattern (`rankings.service.ts:62-77`), identical in shape to the pre-existing `mtsListingsService` (`mts-listings.service.ts:116-136`) |
| FE-14 | Query key factory uses `as const` | PASS | `useRankings.ts:20` `[...rankingsKeys.all, "leaderboard", tenantId, worldId, filter] as const` |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | PASS (N/A) | No form in this feature (read-only leaderboard) |
| FE-16 | Schema in `lib/schemas/` with inferred type | PASS (N/A) | No Zod schema needed/added |

## Feature-Specific Checks (per audit brief)

| Check | Status | Evidence |
|---|---|---|
| JSON:API typing over `apiClient`/`ApiPagedResponse`, no raw `fetch` | PASS | `rankings.service.ts:1,3,69-72` — `apiClient.get<ApiPagedResponse<RankingEntry>>(...)`; no `fetch(` anywhere in the diff |
| `meta.total`/`meta.page.last` authoritative, not inferred from length | PASS | `rankings.service.ts:73-75`: `const total = resp.meta?.total ?? resp.data.length; const lastPage = resp.meta?.page?.last ?? 1;` — length is only a fallback when `meta` is absent, matching the `?? resp.data.length` idiom already used in `mts-listings.service.ts:132` |
| Query keys tenant-first, no cross-tenant leakage | PASS | `useRankings.ts:19-21`; `RankingsPage.tsx:39` passes `activeTenant?.id ?? ""` as `tenantId` so a tenant switch produces a new key and forces a refetch (reinforced by `TenantProvider`'s `queryClient.clear()` on tenant change per `services/atlas-ui/CLAUDE.md`) |
| `enabled` gating present and exercised | PASS | `RankingsPage.tsx:42` `!!activeTenant`; `LeaderboardRow.tsx:33` via `useCharacter`'s internal `enabled: !!tenant?.id && !!characterId` |
| Hooks live under `lib/hooks/api/` | PASS | `services/atlas-ui/src/lib/hooks/api/useRankings.ts` |
| Tenant from `useTenant`, worlds via `useTenantConfiguration` (world id = index), mirrors `MarketplacePage` | PASS | `RankingsPage.tsx:1-4,28-30` (`useTenant`, `useTenantConfiguration(activeTenant?.id ?? "")`, `tenantConfigQuery.data?.attributes.worlds ?? []`) is line-for-line the same idiom as `MarketplacePage.tsx:48-49`; `worlds.map((world, index) => …)` at `RankingsPage.tsx:67-70` treats array index as world id, same as `MarketplacePage.tsx:162-163` |
| Fail-open image rendering in `LeaderboardRow` | PASS | `LeaderboardRow.tsx:39-48`: renders `OptimizedCharacterRenderer` only when `characterQuery.data` is present, otherwise a neutral `<div className="h-12 w-12 rounded bg-muted" aria-hidden="true" />` placeholder — text cells (name/level/rank) render unconditionally regardless. Verified by test: `LeaderboardRow.test.tsx` mocks `useCharacter` to return `{ data: undefined, isError: true }` and asserts `"B"`, `"50"`, `"#1"` still render (`LeaderboardRow.test.tsx:14-16,41-52`) |
| Correct `useCharacter(tenant, String(id))` usage | PASS | `LeaderboardRow.tsx:33`: `useCharacter(activeTenant!, String(a.characterId))` — matches the pre-existing `CharacterDetailPage.tsx:55` call-site pattern exactly |
| Styling via Tailwind/shadcn, no anti-patterns | NEEDS-WORK | See FE-06 above — hardcoded `text-green-600`/`text-red-600` instead of semantic tokens |
| Named page export | PASS | `RankingsPage.tsx:27` `export function RankingsPage()` |
| Lazy route | PASS | `App.tsx` (diff) adds `const RankingsPage = lazy(() => import("@/pages/RankingsPage").then((m) => ({ default: m.RankingsPage })));` and `<Route path="/rankings" element={<RankingsPage />} />`; confirmed by `npm run build` emitting a standalone `dist/assets/RankingsPage-nLse6nSo.js` chunk (9.06 kB) rather than being inlined into the main bundle |
| Vitest `vi.*` tests asserting real rendered behavior | NEEDS-WORK | See Testing section below — real, but shallow in two of the four files |
| Accessibility of movement arrows (distinct `aria-label`s) | PASS | `LeaderboardRow.tsx:15` `aria-label="moved up"`, `:18` `aria-label="moved down"`, `:21` `aria-label="no change"` — three distinct labels, one per `MoveArrow` branch |

## Testing Detail

- `rankings.service.test.ts` — real assertions: verifies the exact URL-encoded query string sent to `apiClient.get` and that `meta.total`/`meta.page.last` are surfaced as `res.total`/(implicitly) `lastPage`. Good coverage.
- `useRankings.test.tsx` (`useRankings.test.tsx:1-19`) — **never calls `useRankings`**. It only imports `rankingsKeys` and asserts the pure key-factory output (`key[0] === "rankings"`, `key.toContain("t1")`, `key.toContain(0)`). There is no `renderHook` + `QueryClientProvider` exercising the actual `useQuery` call, so the `enabled` gating and `queryFn` wiring that FE-09 depends on are asserted only by code inspection, not by a test, in this diff.
- `RankingsPage.test.tsx` (`RankingsPage.test.tsx:21-26`) — mocks `useRankings` to return a fixed empty page (`entries: [], total: 0, lastPage: 1`) and asserts only that the `Rankings` heading renders. It exercises none of: job-category filter switching, Previous/Next pagination button enable/disable logic (`RankingsPage.tsx:131,142`), the `query.isError` branch (`RankingsPage.tsx:105-106`), or actual `LeaderboardRow` rendering with populated `entries`.
- `LeaderboardRow.test.tsx` — real, meaningful assertions: fail-open rendering and the "moved up" arrow label. Does not cover the "moved down" or "no change" arrow branches, but the covered case is sufficient to prove the `MoveArrow` component branches on `move` correctly.

This satisfies the letter of "tests exist per changed file" (FE-17) but two of the four suites (`useRankings.test.tsx`, `RankingsPage.test.tsx`) don't exercise the behavior the audit brief asked about (tenant gating, pagination, error state) — call this an Important/non-blocking gap rather than a FAIL, since `npm test` is green and the untested paths are simple enough to read-verify (which this audit did).

## Summary

### Blocking (must fix)
- None. Build is clean, all tests pass, lint is clean, and no `any`/direct-API/inline-Zod/state-mutation/default-export violations were found.

### Non-Blocking (should fix)
- **FE-06** — `LeaderboardRow.tsx:15,18` and `RankingsPage.tsx:106` use hardcoded `text-green-600`/`text-red-600` instead of semantic tokens. Pre-existing pattern elsewhere in the codebase, but still worth a follow-up sweep (ideally covering all ~11 occurrences, not just these 3, since a partial fix here would be inconsistent).
- **FE-05** — `RankingsPage.tsx` has no `isLoading`/Skeleton treatment for the initial fetch; the table renders empty-bodied until data arrives. Mirrors `MarketplacePage.tsx`'s existing gap.
- **Testing** — `useRankings.test.tsx` doesn't exercise the hook's actual `useQuery`/`enabled` wiring (only the key factory); `RankingsPage.test.tsx` doesn't cover pagination, job-category filtering, the error branch, or populated-row rendering. Both suites pass but under-test the feature relative to `testing-guide.md`'s "test loading, error, and success states" rule.
- Minor: `RankingsPage.tsx:68` uses the world array `index` as the React `key` in the `Select` — mirrors `MarketplacePage.tsx:163` precedent, not a checklist item, flagged only for completeness.
