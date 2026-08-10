# Frontend Audit — task-207-cash-shop-surprise

- **Audit Scope:** atlas-ui TS/React changes, branch point `1e0a321b8` → HEAD `92fddbb61` (`git diff --stat 1e0a321b8..HEAD -- services/atlas-ui`)
- **Guidelines Source:** frontend-dev-guidelines skill
- **Date:** 2026-08-10
- **Build:** PASS (per requester — not re-executed this session; instructed to skip re-running `npm run build`)
- **Tests:** 243 files / 1994 tests passed (per requester — not re-executed this session; instructed to skip re-running `npx vitest run`)
- **Overall:** NEEDS-WORK

## Build & Test Results

Not independently re-run in this session (explicitly out of scope per the audit request — the requester already verified `npx vitest run` and `npm run build` pass, and `tools/lint.sh --check` passes, with a documented pre-existing ESLint baseline failure unrelated to this branch). Findings below are drawn from static reading of the changed files.

## File Inventory

- `services/atlas-ui/src/components/features/reward-pools/KindBadge.tsx` — Component
- `services/atlas-ui/src/components/features/reward-pools/PoolFormDialog.tsx` — Component
- `services/atlas-ui/src/components/features/reward-pools/PoolItemDialog.tsx` — Component
- `services/atlas-ui/src/components/features/reward-pools/PoolItemsTable.tsx` — Component
- `services/atlas-ui/src/components/features/reward-pools/PoolNameCell.tsx` — Component
- `services/atlas-ui/src/components/features/reward-pools/__tests__/KindBadge.test.tsx` — Test
- `services/atlas-ui/src/components/features/reward-pools/__tests__/PoolFormDialog.test.tsx` — Test
- `services/atlas-ui/src/components/features/reward-pools/__tests__/PoolItemDialog.test.tsx` — Test
- `services/atlas-ui/src/lib/hooks/api/__tests__/useRewardPools.test.tsx` — Test (existing hook file `useRewardPools.ts` itself is unchanged in this branch)
- `services/atlas-ui/src/lib/schemas/__tests__/reward-pools.schema.test.ts` — Test
- `services/atlas-ui/src/lib/schemas/reward-pools.schema.ts` — Schema
- `services/atlas-ui/src/lib/utils/reward-pool-chance.ts` — Other (pure utility, holds `POOL_ITEM_TABLE_LAYOUT` kind-layout Record)
- `services/atlas-ui/src/pages/RewardPoolDetailPage.tsx` — Page
- `services/atlas-ui/src/pages/RewardPoolsPage.tsx` — Page
- `services/atlas-ui/src/services/api/__tests__/reward-pools.service.test.ts` — Test
- `services/atlas-ui/src/types/models/reward-pool-item.ts` — Type
- `services/atlas-ui/src/types/models/reward-pool.ts` — Type

`services/atlas-ui/src/services/api/reward-pools.service.ts` and `services/atlas-ui/src/lib/hooks/api/useRewardPools.ts` were **not** touched by this branch (only their tests were) — read for context but not counted as changed files.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | No `: any` / `as any` matches in any in-scope file. |
| FE-02 | No manual class concatenation | PASS | All conditional classes use `cn()`, e.g. `services/atlas-ui/src/pages/RewardPoolsPage.tsx:134` (`cn("h-4 w-4", isRefreshing && "animate-spin")`). No `className={"..." + ...}` matches. |
| FE-03 | No direct API client calls in components | PASS | No `from "@/lib/api/client"` in any component/page in scope; all reads/writes go through `rewardPoolsService` / the `useRewardPools` hooks. |
| FE-04 | No inline Zod schemas in components | PASS | No `z.object(` / `z.string(` etc. in any component or page in scope. All three pool schemas and both item schemas live in `services/atlas-ui/src/lib/schemas/reward-pools.schema.ts:11-66`. |
| FE-05 | No spinners for content loading | PASS | Only `animate-spin` match is on the refresh icon button, `services/atlas-ui/src/pages/RewardPoolsPage.tsx:134`, gated by `isRefreshing` — allowed per the submit/action-button exception. |
| FE-06 | No hardcoded colors | PASS | No `bg-white/black/gray-N/red-N/...` matches. The new `KindBadge.tsx:15,20` amber/violet utility classes use opacity-scale tokens (`bg-amber-500/15`, `bg-violet-500/15`) matching the pre-existing amber-badge convention noted in the file's own comment, not raw grayscale/white/black. |
| FE-07 | No state mutation | PASS | `RewardPoolsPage.tsx:73` calls `acc[p.attributes.kind].push(p)` inside a `.reduce()` whose accumulator is a **fresh** object literal seeded each render (`{ gachapon: [], incubator: [], "cash-surprise": [] }`, `RewardPoolsPage.tsx:76`) — this mutates a locally-owned array being built, not existing React state, so it is not a FE-07 violation. |
| FE-08 | No default exports for components | PASS | No `export default function` in any in-scope file. |
| FE-09 | Tenant guard in hooks | N/A (out of scope) | `useRewardPools.ts` (guards present, e.g. `enabled: !!activeTenant`) was not modified by this branch — only its test was touched to add `commodityId: 0` to a mutation fixture (`useRewardPools.test.tsx:57-62`). No new hook was added. |
| FE-10 | Tenant ID in query keys | FAIL (pre-existing, not newly introduced) | `services/atlas-ui/src/lib/hooks/api/useRewardPools.ts:23-31` — `rewardPoolKeys.list()`, `.detail(id)`, `.items(poolId)`, `.globalItems()` contain no `tenant?.id`. This file is unchanged by the branch, so it is not a new regression from this feature, but the cash-surprise code path added in this branch (`RewardPoolsPage.tsx`, `RewardPoolDetailPage.tsx`) rides on these same unscoped keys. Flagged for visibility, not counted against this branch's own conformance. |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS | Every new `try { await ...mutateAsync(...) } catch (e) { toast.error(createErrorFromUnknown(e).message); }` block follows the pattern, e.g. `PoolFormDialog.tsx:150-152,173-175,196-198`, `PoolItemDialog.tsx:156-158`. |

## Architecture / Exhaustiveness Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `RewardPoolData` / `RewardPoolItemData` both `{ id: string, type: string, attributes: {...} }` — `services/atlas-ui/src/types/models/reward-pool.ts:12-16`, `services/atlas-ui/src/types/models/reward-pool-item.ts:11-15`. `commodityId: number` is non-optional on `RewardPoolItemAttributes` as required by design — `reward-pool-item.ts:8`. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | PASS | `PoolFormDialog.tsx:93-111` (`gachaponForm`, `incubatorForm`, `cashSurpriseForm` each via `useForm({ resolver: zodResolver(...) })`); `PoolItemDialog.tsx:89-94`. |
| FE-16 | Schema in `lib/schemas/` with inferred type | PASS | `cashSurprisePoolSchema` / `CashSurprisePoolFormData` (`reward-pools.schema.ts:50-54`), `cashSurpriseItemSchema` / `CashSurpriseItemFormData` (`reward-pools.schema.ts:60-66`). |
| Exhaustiveness — `KindBadge` | Record over `RewardPoolKind`, no fallback | PASS | `KindBadge.tsx:13-25` — `Record<RewardPoolKind, React.ReactElement>` with all three keys and no `?? fallback`; a 4th kind is a compile error. |
| Exhaustiveness — `PoolNameCell` icon source | Record over `RewardPoolKind` | PASS | `PoolNameCell.tsx:27-34` — `ICON_SOURCE: Record<RewardPoolData["attributes"]["kind"], "item" \| "npc">`, no fallback. |
| Exhaustiveness — `PoolItemsTable` layout | Record over `RewardPoolKind` | PASS | `reward-pool-chance.ts:17-22` — `POOL_ITEM_TABLE_LAYOUT: Record<RewardPoolKind, "tiered" \| "flat">`, consumed at `PoolItemsTable.tsx:97`. |
| Exhaustiveness — `RewardPoolsPage` kind grouping | Record accumulator, no `.filter()` per kind | PASS | `RewardPoolsPage.tsx:69-79` — `reduce<Record<RewardPoolKind, RewardPoolData[]>>` seeded with all three kind keys; a 4th kind fails to compile (`acc[p.attributes.kind]` would be `undefined`-typed only if the Record's key set didn't cover it — TS forces the initial value to declare every key). |
| Exhaustiveness — `PoolFormDialog` active-form switch | `switch` with no `default`, definite-assignment | PASS | `PoolFormDialog.tsx:206-372` — `let activeForm: React.ReactElement;` declared without an initializer, `switch (kind)` has exactly the three `RewardPoolKind` cases and no `default`; using `activeForm` after the switch relies on TypeScript's exhaustive narrowing, so an unhandled 4th kind produces a "used before being assigned" compile error. |
| Exhaustiveness — `RewardPoolDetailPage` icon/header logic | **Boolean ternary reintroduced**, not exhaustive | **FAIL** | `RewardPoolDetailPage.tsx:75` (`const isIncubator = pool?.attributes.kind === "incubator";`) drives `eggIconUrl` (`:114-124`), `machineIconUrl` (`:126-136`, gated on `!isIncubator`), `headerName` (`:137-139`), and the header `<img>` branches (`:145,148`). This is exactly the `isIncubator`-boolean anti-pattern the task/PR description calls out as the bug class that silently mis-rendered a 3rd kind. Currently harmless for `cash-surprise` only because `npcIds` is always empty for that kind (so `machineIconUrl` degrades to `null` rather than showing a wrong icon), but it is not a compile-time guarantee — a future 5th kind (or a cash-surprise pool that somehow carries an `npcIds` entry) falls into the `!isIncubator` branch and renders the NPC-icon logic silently, same failure mode the sibling components (`KindBadge`, `PoolNameCell`, `POOL_ITEM_TABLE_LAYOUT`, the `RewardPoolsPage` reduce, the `PoolFormDialog` switch) were all rewritten in this same PR to eliminate. This file is in the changed-file set (16 lines touched) and reachable from the "cash-surprise" kind, so it is in scope. **Fix:** replace `isIncubator`/`!isIncubator` with a `Record<RewardPoolKind, ...>` (icon source + header-name formatter), mirroring `PoolNameCell.tsx`'s `ICON_SOURCE` idiom, which already solves the identical problem one component away. |
| Exhaustiveness — `PoolItemDialog` weighted/commodity gating | **Boolean chain, not exhaustive** | **FAIL** | `PoolItemDialog.tsx:56-62`: `const weighted = kind === "incubator" \|\| kind === "cash-surprise"; const needsCommodity = kind === "cash-surprise"; const schema = needsCommodity ? cashSurpriseItemSchema : weighted ? weightItemSchema : tierItemSchema;` — this is the exact `isIncubator`-style boolean/ternary chain the task flags as the historical bug pattern, reintroduced here for a *different* axis (item-form shape) than the one already fixed via `POOL_ITEM_TABLE_LAYOUT`. It governs schema selection (`:58-62`), submit-payload shaping (`:125-144`), and field rendering (`:194-230`). `kind` here is `"gachapon" \| "incubator" \| "cash-surprise" \| "global"` (`:43`), a 4-member union distinct from `RewardPoolKind`, so `POOL_ITEM_TABLE_LAYOUT` can't be reused directly — but nothing stops a local `Record<PoolItemDialogKind, "tiered" \| "flat" \| "global">`-shaped lookup with the same guarantee the rest of the PR uses. As written, a new pool kind added to `RewardPoolKind` (and threaded into this component's `kind` prop) silently falls through to `weighted = false`, i.e. the tier/gachapon form and `tierItemSchema` — no compile error, no runtime error, just a wrong form shown to the operator. **Fix:** replace the boolean chain with an exhaustive switch/Record keyed on the full `kind` union (including `"global"`) with no fallback arm. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PARTIAL — FAIL for the two pages | `KindBadge.tsx` ✅ covered (`KindBadge.test.tsx`). `PoolFormDialog.tsx` ✅ covered, including a dedicated cash-surprise create-path test (`PoolFormDialog.test.tsx:65-89`). `PoolItemDialog.tsx` ✅ covered, including dedicated cash-surprise commodity-required and commodity-submit tests (`PoolItemDialog.test.tsx:89-125`). `PoolItemsTable.tsx` and `PoolNameCell.tsx` have **no** test files in this diff and none pre-existing (`find` over `__tests__/` shows only `KindBadge.test.tsx`, `PoolFormDialog.test.tsx`, `PoolItemDialog.test.tsx`) despite `PoolItemsTable.tsx` gaining new cash-surprise-specific rendering (`showCommodity` column, `PoolItemsTable.tsx:98,111,124-126`). **`RewardPoolDetailPage.tsx`** (16 lines changed, adds `isFlatKind`/cash-surprise handling) and **`RewardPoolsPage.tsx`** (38 lines changed, adds the cash-surprise tab and `poolsByKind` grouping) both have pre-existing test files (`RewardPoolDetailPage.test.tsx`, `RewardPoolsPage.test.tsx`) that were **not** updated in this branch — `grep -n "cash-surprise\|Cash Surprise" src/pages/__tests__/RewardPoolDetailPage.test.tsx src/pages/__tests__/RewardPoolsPage.test.tsx` returns zero matches. Neither the new "Cash Surprise" tab/count on `RewardPoolsPage`, nor the flat-layout / no-tier-card / no-NPC-card branch on `RewardPoolDetailPage` for a cash-surprise pool, is exercised by any test. |
| FE-18 | Mocks updated when services changed | PASS (N/A) | `reward-pools.service.ts` itself is unchanged; its test fixtures were updated to include `commodityId: 0` on existing gachapon/incubator item payloads (`reward-pools.service.test.ts` diff), consistent with `RewardPoolItemAttributes.commodityId` becoming non-optional. `useRewardPools.test.tsx` fixture updated the same way. |
| Commodity-non-zero rule pinned by a test | Yes | PASS | `reward-pools.schema.test.ts:91-108` — `cashSurpriseItemSchema` rejects `commodityId: 0` and `commodityId: -1`; would fail immediately if the `.positive()` constraint were removed or loosened to `.min(0)` / `.nonnegative()`. Also exercised end-to-end through the form in `PoolItemDialog.test.tsx:89-106` ("requires a commodity id for cash-surprise entries"). |

## Summary

### Blocking (must fix)

- **FE-Exhaustiveness (RewardPoolDetailPage)** — `services/atlas-ui/src/pages/RewardPoolDetailPage.tsx:75,114-136,137-139,145,148` reintroduces the `isIncubator` boolean/ternary anti-pattern this PR was explicitly built to eliminate elsewhere. Not a currently-visible bug (cash-surprise pools happen to have empty `npcIds`), but it is not compile-time-guaranteed and is inconsistent with every sibling kind-dispatch site in the same PR (`KindBadge`, `PoolNameCell`, `POOL_ITEM_TABLE_LAYOUT`, `RewardPoolsPage`'s reduce, `PoolFormDialog`'s switch).
- **FE-Exhaustiveness (PoolItemDialog)** — `services/atlas-ui/src/components/features/reward-pools/PoolItemDialog.tsx:56-62` (`weighted`, `needsCommodity`, ternary `schema` selection) is a boolean/ternary chain over the kind union, not an exhaustive construct. A future kind silently gets the wrong schema/form (falls to `tierItemSchema`) with no compile error.
- **FE-17 (Testing)** — `RewardPoolDetailPage.tsx` and `RewardPoolsPage.tsx` were both modified for cash-surprise support but their existing test files were not updated to cover the new "Cash Surprise" tab/count, `isFlatKind` branch, or the header rendering for a cash-surprise pool. `PoolItemsTable.tsx`'s new `showCommodity` column has no test coverage at all (no `PoolItemsTable.test.tsx` exists).

### Non-Blocking (should fix)

- **FE-10** — `useRewardPools.ts` query keys (`rewardPoolKeys.list/detail/items/globalItems`) still omit `tenant?.id`, a pre-existing gap this branch does not introduce but does build new cash-surprise data flows on top of.
