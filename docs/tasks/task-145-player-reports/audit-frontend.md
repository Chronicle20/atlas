# Frontend Audit — task-145-player-reports (whole workstream)

- **Audit Scope:** `services/atlas-ui/` changes in range `9ecf9c9759a9db059fb40f497e7c6a04e43f06fd..6a9e91e561dbd7138274e636ec9fa545f304cd2f` (7 commits: data layer, list page, detail page, plus three review-fix commits)
- **Guidelines Source:** `.claude/skills/frontend-dev-guidelines/` (SKILL.md + all `resources/*.md`), cross-checked against the actual `services/atlas-ui/CLAUDE.md` (Vite/React Router, not the skill doc's stale Next.js template) since the real service doc overrides the generic skill doc wherever they disagree
- **Date:** 2026-08-05
- **Build:** PASS
- **Tests:** 1400 passed, 0 failed (195 test files)
- **Overall:** NEEDS-WORK (one blocking gap; everything else PASS with cited evidence)

## Build & Test Results

```
$ nvm use 22 && npm run build
✓ built in 1.28s
(only the pre-existing ConversationEditorPanel >500kB chunk warning — unrelated to this workstream)

$ nvm use 22 && npm test -- --run
 Test Files  195 passed (195)
      Tests  1400 passed (1400)
   Duration  23.57s

$ npx tsc --noEmit -p tsconfig.app.json
(clean, no output)

$ npx eslint <all 11 in-scope files>
ESLint: No issues found
```

## File Inventory

- `src/types/models/report.ts` — **Type**
- `src/services/api/reports.service.ts` — **Service**
- `src/lib/hooks/api/useReports.ts` — **Hook**
- `src/services/api/__tests__/reports.service.test.ts` — **Test**
- `src/pages/ReportsPage.tsx` — **Page**
- `src/pages/reports-columns.tsx` — **Component** (colocated column config, kebab-case per `services/atlas-ui/CLAUDE.md`)
- `src/components/features/reports/ReportStatusBadge.tsx` — **Component**
- `src/pages/__tests__/ReportsPage.test.tsx` — **Test**
- `src/pages/ReportDetailPage.tsx` — **Page**
- `src/components/features/reports/UpdateReportStatusDialog.tsx` — **Component**
- `src/components/features/reports/__tests__/UpdateReportStatusDialog.test.tsx` — **Test**
- `src/App.tsx` (route registration, +13 lines) — **Other** (routing)
- `src/components/app-sidebar-items.ts` (+1 line) — **Other** (sidebar)
- `src/lib/breadcrumbs/routes.ts` (+12 lines) — **Other** (breadcrumbs)

## Verifying the two already-fixed findings

**Row navigation (B1, commit `56c979f59`).** `src/pages/reports-columns.tsx:69-83` renders a `DropdownMenu` with a `DropdownMenuItem onClick={() => onView?.(report)}` — no `<div onClick>`/`cursor-pointer` anywhere. Confirmed with a scoped grep across all 11 in-scope files for `cursor-pointer` and `<div[^>]*onClick` — zero matches. Structurally identical to `src/pages/bans-columns.tsx:126-129`.

**Dialog reopen resync (commit `457d7b78b`).** `src/components/features/reports/UpdateReportStatusDialog.tsx:42-46` — the `wasOpen` render-time-adjustment pattern is byte-for-byte structurally identical to `src/components/features/reward-pools/PoolFormDialog.tsx:52-56`. The regression test at `__tests__/UpdateReportStatusDialog.test.tsx:73-94` genuinely exercises the fix: it selects "Reviewed", clicks Cancel, closes (`rerenderWithOpen(false)`) and reopens (`rerenderWithOpen(true)`) without unmounting the component (the dialog is rendered unconditionally by `ReportDetailPage` and only `open` toggles — confirmed at `ReportDetailPage.tsx:226-230`), then asserts the combobox shows "Open" and Save is disabled again. Without the `wasOpen` reset, `status` state has no other write path back to `report.attributes.status` after a cancel, so the assertion would fail — the test is a real regression guard, not a tautology.

**Error/not-found split (commit `457d7b78b`).** `ReportDetailPage.tsx:66-74` toasts via `createErrorFromUnknown(reportQuery.error, "Failed to load report")` in a dedicated `useEffect`, and `:80-95` / `:97-112` render genuinely distinct branches (error branch offers no retry copy about "not found"; not-found branch is reached only once `reportQuery.error` is falsy and `report` is still null). Confirmed correct.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ': any\|as any'` across all 11 files returns zero matches |
| FE-02 | No manual class concatenation | PASS | `grep -n 'className={"'` returns zero matches; all conditional classes go through literal Tailwind strings or none at all — no `cn()` calls were needed since no branch-conditional classNames exist in this workstream |
| FE-03 | No direct API client calls in components | PASS | `@/lib/api/client` is imported only by `reports.service.ts:1` and its own test `__tests__/reports.service.test.ts:12` — both service-layer files, never a page/component |
| FE-04 | No inline Zod schemas in components | PASS | `grep -nE 'z\.(object\|string\|number)\('` returns zero matches — this workstream has no forms and no Zod schemas at all |
| FE-05 | No spinners for content loading | PASS | `ReportsPage.tsx:24-39` (`ReportsPageSkeleton`) and `ReportDetailPage.tsx:23-46` (`ReportDetailSkeleton`) are Skeleton-based; the only `animate-spin` in scope is `UpdateReportStatusDialog.tsx:109`, a `Loader2` gated behind `updateStatus.isPending` inside the Save button — exactly the sanctioned submit-button exception |
| FE-06 | No hardcoded colors | PASS | `grep -nE 'bg-(white\|black\|gray-[0-9]+\|red-[0-9]+\|blue-[0-9]+\|green-[0-9]+)'` returns zero matches; all status/muted styling goes through `text-muted-foreground`, `text-foreground`, and shadcn `Badge`/`Button` variant props (`ReportStatusBadge.tsx:4-9` maps status to `"default"\|"secondary"\|"outline"` variants, never raw colors) |
| FE-07 | No state mutation | PASS (matches established convention) | `reports.service.ts:12` `reports.sort(...)` mutates in place, but the array is the fresh one just returned by `api.getList` inside the same function scope, never external/React state — identical to `bans.service.ts`'s `sortBans`. Not a new violation; same call already made in the prior per-commit audit for this exact line |
| FE-08 | No default exports for components | PASS | `grep -n 'export default'` returns zero matches across all 11 files |
| FE-09 | Tenant guard in hooks | PASS | `useReports.ts:37` `enabled: !!tenant?.id`; `useReports.ts:50` `enabled: !!tenant?.id && !!id` — both hooks take explicit `tenant: Tenant \| null` (`:31`, `:44`), Pattern A per `patterns-multitenancy.md`. Both pages obtain `activeTenant` via `useTenant()` (`ReportsPage.tsx:42`, `ReportDetailPage.tsx:49`) and pass it straight through (`ReportsPage.tsx:54`, `ReportDetailPage.tsx:54`) — no bypass |
| FE-10 | Tenant ID in query keys | PASS | `useReports.ts:23-24` `list()` and `:26-27` `detail()` both include `tenant?.id ?? "no-tenant"` |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS | `ReportDetailPage.tsx:66-74` (query error → toast) and `UpdateReportStatusDialog.tsx:57-63` (mutation `onError` → toast) both use `createErrorFromUnknown` + `toast.error`. `reports.service.ts` and `useReports.ts` themselves have no `.catch(` — consistent with `bans.service.ts`/`useBans.ts`, which let React Query's own error channel surface failures to the calling component instead of swallowing them at the service/hook layer |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `report.ts:55-59` — `Report { id: string; type: "reports"; attributes: ReportAttributes }`. The extra `type` field beyond the skill doc's minimal `{id, attributes}` template matches an existing codebase pattern (`reward-pool.ts:12-15`, `quest.ts`, `service.ts`, `location.ts` all carry a `type` field too) — not a new divergence |
| FE-13 | Service extends `BaseService` (when applicable) | PASS (matches codebase convention) | `reports.service.ts:19` is a plain object-literal export (`export const reportsService = {...}`), matching `bans.service.ts`'s actual style, not the skill doc's stale class-based template |
| FE-14 | Query key factory uses `as const` | PASS | `useReports.ts:20-28` — every branch of `reportKeys` ends in `as const` |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No forms in this workstream — `UpdateReportStatusDialog` is a single-`Select` dialog with local `useState`, matching `ExpireBanDialog.tsx`'s pattern (a plain confirmation-style dialog), not `PoolFormDialog`'s multi-field form. Correct choice: a one-field status picker doesn't need RHF/Zod overhead |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No Zod schemas needed (see FE-15) |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | **NEEDS-WORK** (one gap; rest PASS) | See "Testing coverage analysis" below |
| FE-18 | Mocks updated when services changed | N/A | No `__mocks__/` directory entries reference `reportsService`; each test file mocks `@/services/api/reports.service` or `@/lib/api/client` inline via `vi.mock`, matching the established per-test-file mocking convention (`bans.service.test.ts`, `PoolFormDialog.test.tsx`) rather than a shared `__mocks__/` fixture |

### Testing coverage analysis (Focus Area 4)

Applying "match the actual analog," not "always test everything":

- **`reports.service.ts`** → `reports.service.test.ts` exists (`services/atlas-ui/src/services/api/__tests__/reports.service.test.ts`, added in commit `f7c3cb97f` specifically to close the FE-17 gap the first per-task audit flagged). Covers `sortReports` ordering, the `?status=` branch vs. no-filter branch, and the JSON:API PATCH envelope — matches the `bans.service.test.ts` precedent. **PASS.**
- **`ReportsPage.tsx`** → `ReportsPage.test.tsx` exists, added in commit `56c979f59` explicitly to mirror `BansPage.test.tsx`. Covers default-fetch, rendering, and status-filter re-fetch. **PASS.**
- **`UpdateReportStatusDialog.tsx`** → `UpdateReportStatusDialog.test.tsx` exists, added in commit `457d7b78b`. `PoolFormDialog.tsx` (a dialog holding editable local state) has `__tests__/PoolFormDialog.test.tsx`; `ExpireBanDialog.tsx`/`CreateBanDialog.tsx`/`DeleteBanDialog.tsx` — verified via `find src/components/features/bans -type f` — have **no** test files. `UpdateReportStatusDialog` holds editable local `status` state (like `PoolFormDialog`, unlike the bans dialogs which are pure confirmation actions), so it falls on the "tested" side of that line. **PASS**, and the test is a genuine regression guard (see verification above), not filler.
- **`ReportStatusBadge.tsx`, `reports-columns.tsx`** → no test files, and no precedent requires one: `find src/components/features/bans -iname "*.test.tsx"` returns zero hits for `BanStatusBadge.tsx`, `BanTypeBadge.tsx`, or `bans-columns.tsx`. **N/A, not a gap.**
- **`useReports.ts`** → no dedicated hook test file, and no precedent requires one: `find src/lib/hooks/api/__tests__` has no `useBans.test.tsx`. Hook coverage arrives indirectly through the page/dialog tests, consistent with the rest of the codebase. **N/A, not a gap.**
- **`ReportDetailPage.tsx`** → **no test file exists**, and this is the one place the "match the actual analog" standard cuts against the workstream, not for it:
  - `BanDetailPage.tsx` (the direct structural analog per the plan: "structural copy of `BanDetailPage.tsx`", `plan.md:4958`) also has no test — so on its own, a missing `ReportDetailPage.test.tsx` would be defensible by that precedent alone.
  - But `RewardPoolDetailPage.tsx` — a comparably complex detail page (multi-branch loading/error/not-found, nested tables, service mocks) — **does** have `pages/__tests__/RewardPoolDetailPage.test.tsx`. Detail-page test coverage in this codebase is genuinely mixed, not a clean "never."
  - More importantly: commit `457d7b78b`'s own message explicitly distinguishes the two fixes it shipped in the same commit — the dialog fix got "*Added a regression test that fails without the fix*," while the `ReportDetailPage` error/not-found split got no equivalent regression test. That's an internal inconsistency in the same review-fix commit, not a pre-existing codebase convention being followed. The bug that was fixed (conflating a genuine fetch error with "report not found") is exactly the kind of branch-logic bug a `RewardPoolDetailPage.test.tsx`-style test would catch on regression, and nothing in `ReportDetailPage.tsx` currently locks in: (a) that an error renders the error branch and toasts, rather than falling through to "not found"; (b) that a null `chatLog`/`serverTranscript` render the "No chat log submitted."/"No server transcript captured." empty states rather than crashing or rendering blank (Focus Area 1); (c) that a populated `serverTranscript` renders the four-column table correctly.
  - **Verdict: blocking gap.** Add `services/atlas-ui/src/pages/__tests__/ReportDetailPage.test.tsx` covering at minimum: (1) the error-vs-not-found branch split with a toast assertion on the error path, (2) the null-transcript/null-chatLog empty-state rendering, and (3) a populated-transcript render. This is scoped, bounded work — not a new task.

## Focus Area 1 — Nullable runtime state (chatLog / serverTranscript)

- **No `!` non-null assertions.** `grep -nE '[a-zA-Z0-9_$)\]]![.;,) ]'` across all 11 files returns zero matches. `ReportDetailPage.tsx` handles both nullable fields with plain conditionals: `attributes.chatLog ? (...) : (...)` (`:172-180`) and `attributes.serverTranscript && attributes.serverTranscript.length > 0 ? (...) : (...)` (`:189-221`) — no assertion needed because the guard itself narrows the type.
- **Empty states are clear and non-alarming**, not blank and not styled as errors: `"No chat log submitted."` (`:177-179`) and `"No server transcript captured."` (`:218-220`), both rendered as `<p className="text-sm text-muted-foreground">` — informational muted text, not `text-destructive` or any error-adjacent styling. Matches the codebase's established empty-state convention (`patterns-components.md`'s "Empty State Pattern," `text-muted-foreground`).
- **Populated transcript is legible.** The table (`:191-216`) has four columns — Time (`new Date(line.timestamp).toLocaleString()`, includes the date per the `457d7b78b` fix, not just time-of-day), Sender (`line.senderName`), Type (`line.chatType`), Message (`line.text`, `whitespace-pre-wrap` so long lines wrap instead of overflowing). The `Table` primitive (`src/components/ui/table.tsx:9`) wraps in `<div className="relative w-full overflow-auto">`, so a genuinely wide table scrolls its own container rather than the page. No horizontal-scroll finding.
- Runtime-nullability is a genuine backend contract (not speculative): `services/atlas-ban/atlas.com/ban/report/resource.go` types both fields as pointer/nullable JSON:API attributes, matching `report.ts:48-49`'s `string | null` / `TranscriptLine[] | null`.

## Focus Area 2 — Cross-task consistency

Checked for drift across the three implementation passes (data layer / list / detail):

- **Naming.** `reportKeys`, `useReports`, `useReport`, `useUpdateReportStatus`, `useInvalidateReports` all mirror `banKeys`/`useBans`/`useBan`/... exactly in shape and casing.
- **Skeleton components.** Both `ReportsPageSkeleton` (`ReportsPage.tsx:24-39`) and `ReportDetailSkeleton` (`ReportDetailPage.tsx:23-46`) follow the same `Skeleton`-array-with-`Array.from({length: N})` idiom as `BansPageSkeleton`/`BanDetailSkeleton` — no divergence between the list-task pass and the detail-task pass.
- **No duplicated helpers.** `ReportStatusBadge` is the only status-badge component for reports (one definition, reused in both `reports-columns.tsx:54` and `ReportDetailPage.tsx:132`) — not reimplemented per page.
- **Loading/error handling divergence (intentional, documented).** `ReportDetailPage` deliberately does *not* auto-bounce on error the way `BanDetailPage.tsx:78-87` does (`navigate("/bans")` inside the error effect) — `ReportDetailPage.tsx:60-65`'s comment explains why (GM navigated via URL, so distinguishing "backend is down" from "doesn't exist" matters more than auto-redirecting). This is a deliberate, explained divergence from the reference implementation, not accidental drift.
- **One component that arguably should have been reused rather than rewritten:** none found. `Card`/`Table`/`Skeleton`/`Badge`/`Select`/`Dialog` are all the shared shadcn primitives; no report-specific reimplementation of an existing shared component exists.
- **Minor: triple `<Toaster richColors />` mount inside one component.** `ReportDetailPage.tsx:92`, `:109`, `:232` each render a `<Toaster richColors />` — one per early-return branch (error, not-found, success), because the branches are three separate JSX returns rather than one shared shell. `BansPage.tsx:199`/`BanDetailPage.tsx` and 13 other pages already mount a page-level `<Toaster richColors />` on top of `App.tsx:253`'s global `<Toaster />`, so a page-level `Toaster` at all is pre-existing convention, not new. But mounting it three times in three branches of the *same* component (where `BanDetailPage.tsx`'s not-found branch, by contrast, mounts none) is a small avoidable inconsistency — not a guideline violation, non-blocking.

## Focus Area 3 — Tenant handling

Already covered under FE-09/FE-10 above. Summary: both `useReports`/`useReport` take explicit `tenant: Tenant | null` (Pattern A), both pages source `activeTenant` from `useTenant()` and pass it through untouched, and both `enabled` guards (`!!tenant?.id` and `!!tenant?.id && !!id`) prevent firing without a tenant — no gaps.

## Summary

### Blocking (must fix)
- **FE-17** — `ReportDetailPage.tsx` has no test file. `RewardPoolDetailPage.test.tsx` establishes that comparably-complex detail pages in this codebase do get tested, and the same review-fix commit (`457d7b78b`) that added a genuine regression test for the sibling dialog fix left the `ReportDetailPage` error/not-found split — an equally real, previously-shipped bug — with zero regression coverage. Add `src/pages/__tests__/ReportDetailPage.test.tsx` covering: the error-toast-vs-not-found branch split, null-chatLog/null-serverTranscript empty-state rendering, and a populated-transcript render.

### Non-Blocking (should fix)
- `ReportDetailPage.tsx:92,109,232` mounts `<Toaster richColors />` redundantly in all three of its early-return branches instead of once at a shared point (or relying on `App.tsx`'s global `<Toaster />`, as most other detail pages already lean on more heavily). Cosmetic; does not affect correctness since Sonner's toast queue is a singleton regardless of how many `<Toaster>` viewports are mounted.
