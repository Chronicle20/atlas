# Frontend Audit — task-131-random-reward-items (UI possible-rewards add-on)

- **Audit Scope:** `git diff b1c50b67d36c9c7174bdd2977de635b8b074051c..23e9d3c20 -- services/atlas-ui`
- **Guidelines Source:** frontend-dev-guidelines skill (note: skill resources describe a Next.js/Jest stack; the actual atlas-ui runtime is Vite + react-router-dom + Vitest per `services/atlas-ui/CLAUDE.md` — checks below are applied against the real stack, not the skill's Next.js assumptions)
- **Date:** 2026-07-16
- **Build:** PASS (`tsc -b && vite build`, nvm 22)
- **Tests:** 947 passed, 0 failed (118 test files, Vitest)
- **Lint:** PASS (`npx eslint` on all 4 changed files, zero errors)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
> atlas-ui@0.1.0 build
> tsc -b && vite build
...
✓ built in 1.45s
```

```
> atlas-ui@0.1.0 test
> vitest run

 Test Files  118 passed (118)
      Tests  947 passed (947)
```

`npx eslint src/components/features/items/PossibleRewardsCard.tsx src/components/features/items/__tests__/PossibleRewardsCard.test.tsx src/pages/ItemDetailPage.tsx src/types/models/item.ts` → no output, zero errors.

## File Inventory

- `services/atlas-ui/src/components/features/items/PossibleRewardsCard.tsx` — **Component** (feature, new)
- `services/atlas-ui/src/components/features/items/__tests__/PossibleRewardsCard.test.tsx` — **Test** (new, not listed in the original prompt's file set but is part of the diff)
- `services/atlas-ui/src/pages/ItemDetailPage.tsx` — **Page** (wiring only, +4 lines)
- `services/atlas-ui/src/types/models/item.ts` — **Type** (`RewardModel` + `ConsumableAttributes.rewards?`)

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ": any\|as any"` over all 4 files → zero matches |
| FE-02 | No manual class concatenation | PASS | No `className={"..." +` pattern; all classes are static template strings passed to JSX `className=""`, no `cn()` needed since no conditional classes are built (`PossibleRewardsCard.tsx:31,53,55,68,71,75,76,77,78`) |
| FE-03 | No direct API client calls in components | PASS | `grep -n "lib/api/client"` → zero matches in `PossibleRewardsCard.tsx`; it consumes `useItemData` (`PossibleRewardsCard.tsx:4,42`), which itself goes through `itemsService` (`src/lib/hooks/useItemData.ts:9,87`) |
| FE-04 | No inline Zod schemas in components | PASS | `grep -n "z\.object\|z\.string\|from \"zod\""` → zero matches; no schema needed (no form) |
| FE-05 | No spinners for content loading | PASS | `grep -n "animate-spin"` → zero matches; row loading state falls back to `Item #${itemId}` text (`PossibleRewardsCard.tsx:47-48`), same pattern as `DroppedByWidget.tsx:38` |
| FE-06 | No hardcoded colors | PASS | `grep -nE "bg-(white\|black\|gray-[0-9]\|red-[0-9]\|...)"` → zero matches; uses `bg-card`, `bg-accent`, `text-muted-foreground` (`PossibleRewardsCard.tsx:53,71`) — semantic tokens |
| FE-07 | No state mutation | PASS | `.sort()` at `PossibleRewardsCard.tsx:21` runs on the array just produced by `.map()` one line above (`PossibleRewardsCard.tsx:19-21`) — a fresh local array, not a mutation of the `rewards` prop or any React state |
| FE-08 | No default exports for components | PASS | `grep -n "export default"` → zero matches; `PossibleRewardsCard.tsx:15` and `RewardRowWidget` at `PossibleRewardsCard.tsx:41` are both named exports/functions, consistent with `services/atlas-ui/CLAUDE.md` "Named exports on pages" convention |
| FE-09 | Tenant guard in hooks | PASS (delegated, not owned by this diff) | `PossibleRewardsCard.tsx` takes no `tenant` param at all — data comes from parent-fetched props plus `useItemData(reward.itemId)` (`PossibleRewardsCard.tsx:42`), which internally guards via `enabled: options.enabled && itemId > 0 && !!activeTenant` (`src/lib/hooks/useItemData.ts:101`, unchanged file) |
| FE-10 | Tenant ID in query keys | WARN (pre-existing, out of diff scope) | `useItemData`'s query key uses `tenantId \|\| ''` (`src/lib/hooks/useItemData.ts:42`), not the documented `'no-tenant'` fallback (anti-patterns.md #3). This file is unchanged by this diff — flagged for visibility only, not counted against this PR |
| FE-11 | Error handling with `createErrorFromUnknown` | N/A | No new `.catch(` / async op introduced in the diff; `PossibleRewardsCard.tsx` is purely presentational over already-fetched `rewards` prop |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `RewardModel` (`src/types/models/item.ts:88-95`) is an **embedded** value inside `ConsumableAttributes.rewards?` (`item.ts:111`), not a standalone JSON:API resource — same pattern as the pre-existing `spec: Record<string, number>` field (`item.ts:110`). Top-level `ConsumableData` still follows `{id, attributes}` (`item.ts:114-117`) |
| FE-13 | Service extends `BaseService` (when applicable) | N/A | No service changed; `items.service.ts:118-119` `getConsumable` is unchanged and already deserializes the whole `attributes` object generically, per design.md §2 |
| FE-14 | Query key factory uses `as const` | N/A | No new query key factory added by this diff |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No form in this diff |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | `RewardModel` is a plain domain-model interface, not a validation schema — correctly placed in `types/models/item.ts:88-95`, not `lib/schemas/` |
| FE-17 | Tests exist for changed components | PASS | `src/components/features/items/__tests__/PossibleRewardsCard.test.tsx` — 7 test cases covering empty-render, chance computation, sort order, decimal precision, ×count/time-limited/announces shown and omitted |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 (detail) | Tests cover documented acceptance criteria (design.md §3.4) | PASS (6/6 listed cases present) | chance-from-weight not raw (`test.tsx:30-35`), sort descending (`test.tsx:37-41`), `total===0` guard (`test.tsx:43-46`), 3-decimal rare-chance fidelity (`test.tsx:48-52`), ×count/time-limited/announces shown (`test.tsx:54-59`) and omitted (`test.tsx:61-66`), empty-rewards renders nothing (`test.tsx:25-28`) |
| FE-17 (gap) | Link target not asserted | WARN | No test asserts the row `Link to={...}` resolves to `/items/${reward.itemId}` (design.md §3.2 "row linking to `/items/{itemId}`"); component code does implement it (`PossibleRewardsCard.tsx:52`), but it is unverified by the test suite |
| FE-18 | Mocks updated when services changed | N/A | No service module changed; `useItemData` is mocked wholesale in the test (`test.tsx:8-14`), consistent with `services/atlas-ui/CLAUDE.md` conventions |

## Design-Adherence Findings (design-ui-possible-rewards.md)

These are not FE-* guideline violations but are deviations from the task's own approved design doc, found while cross-checking per the audit brief ("Verify against the design"). Flagged because an audit that only checks generic guidelines and ignores the task's explicit acceptance criteria would be a false pass.

| Severity | Finding | Evidence |
|----------|---------|----------|
| Important | **Raw weight indicator is missing.** design.md §3.2 explicitly requires: "Chance + raw weight, e.g. `12.4%` with a muted `· w9900` beside it." The implementation renders only `{pct}%` (`PossibleRewardsCard.tsx:78`) — there is no `w{prob}` (or similar) rendering anywhere in the file (`grep -n "w9900\|raw weight" PossibleRewardsCard.tsx` → zero matches). This is a specified, testable acceptance criterion that was dropped. |
| Important | **Chance decimal precision diverges from design without design.md being updated.** design.md §3.2 specifies "formatted to 1 decimal place (e.g. `12.4%`)". The implementation uses 3 decimals (`PossibleRewardsCard.tsx:46`: `(reward.chance * 100).toFixed(3)`), justified by an inline comment about rare-drop fidelity (`PossibleRewardsCard.tsx:43-45`) and backed by a dedicated test (`test.tsx:48-52`). The reasoning is sound, but the design doc still says 1 decimal — it was not amended to reflect this change, so a future reader/reviewer diffing behavior against design.md will see a mismatch. |
| Info | **Manual verification step (design.md §5) not independently exercised.** "open a reward-box item's detail page (e.g. `2022503`, `2022309`) — the card lists rewards with sane percentages summing to ~100%" is a manual/live check that this static audit cannot perform (no running tenant/browser session available). Not a code defect; flagged so it isn't silently treated as done. |

## Summary

### Blocking (must fix)
- None. Build is clean, all 947 tests pass, lint is clean, and no FE-* anti-pattern/architecture check failed against the actual changed files.

### Non-Blocking (should fix)
- **Design gap — raw weight not rendered** (design.md §3.2). Add the `· w{prob}` muted secondary text next to the percentage in `PossibleRewardsCard.tsx:78`, or get explicit sign-off to drop it and update design.md.
- **Design doc out of sync on decimal precision** (design.md §3.2 says 1 decimal, code uses 3). Update design.md to reflect the 3-decimal decision (already justified in code) so the doc stays authoritative.
- **Missing test for the row navigation link target** (`/items/${reward.itemId}`) — add an assertion (e.g., `getByRole('link').closest('a')` → `href` contains `/items/1`) to lock in the behavior design.md §3.2 calls for.
- **Pre-existing, out-of-scope**: `useItemData`'s query key falls back to `''` instead of the documented `'no-tenant'` sentinel (`src/lib/hooks/useItemData.ts:42`). Not introduced by this diff; noted for a future cleanup pass, not blocking this PR.
