# Frontend Audit — task-171 (atlas-ui authored changes)

- **Audit Scope:** AUTHORED atlas-ui surface only — commit `13b6ab0e0` (prettier/eslint-config-prettier wiring), `git diff 13b6ab0e0 947c45f71 -- services/atlas-ui` (ESLint remediation, 32 files), commit `525dfcda5` (idempotence fix, `AttributesPanel.tsx`). The 4235-file `cde242a84` mass reformat was explicitly excluded (pure Prettier output, no manual edits).
- **Guidelines Source:** frontend-dev-guidelines skill (note: skill's architecture doc assumes Next.js; actual atlas-ui is a Vite + react-router-dom SPA per `services/atlas-ui/CLAUDE.md` — pre-existing mismatch, not introduced by this task, not scored against it here)
- **Date:** 2026-07-17
- **Build:** PASS (`npm run build` — 0 errors, only a pre-existing >500kB chunk-size advisory)
- **Tests:** Targeted re-run of the 3 test files touching split/refactored code: 36 passed (36) in `app-sidebar.test.tsx`, `BaselineTargetPicker.test.tsx`, `useBreadcrumbs.test.ts`. Full-suite 887/887 pass was already verified by Task 3's own review per the dispatch brief; not re-run in full here.
- **Lint:** `npm run lint` — 0 errors, 6 warnings, all pre-existing and outside the audited diff (`ApplyPresetDialog.tsx`, `CreateTenantDialog.tsx`, `QuestsPage.tsx` are not in the changed-file list; the one warning in `AccountsPage.tsx` is on line 22 (`accounts = accountsQuery.data ?? []`), a line untouched by this commit — the diff only added a suppression at line 84).
- **Overall: PASS** — no Critical or Important findings. One Minor/non-blocking observation below.

## File Inventory (authored surface)

**Config/wiring (`13b6ab0e0`):**
- `services/atlas-ui/package.json` — Other (build scripts/deps)
- `services/atlas-ui/.prettierrc`, `.prettierignore` — Other (config)
- `services/atlas-ui/eslint.config.js` — Other (config)

**ESLint remediation (`947c45f71`, 32 files):**
- Components: `app-sidebar.tsx`, `app-sidebar-items.ts` (new), `BaselineTargetPicker.tsx`, `BaselineTargetPicker.utils.ts` (new), `ChangeMapDialog.tsx`, `MapImagePanel.tsx`, `MonsterDropWidget.tsx`, `NpcImage.tsx`, `NpcShopCard.tsx`, `NpcShopCommodityDialog.tsx`, `ConversationCanvas.tsx`, `ConversationInspector.tsx`, `NpcConversationCard.tsx`, `QuestConversationCard.tsx`, `SetupRow.tsx`, `setup-format.ts` (new), `item-name-cell.tsx`, `map-cell.tsx`, `ui/sidebar.tsx`
- Hooks: `hooks/use-mobile.tsx`, `lib/hooks/api/useAccountByName.ts`, `lib/hooks/useBreadcrumbs.ts`
- Lib: `lib/api/errors.ts`, `lib/utils/debounce.ts`
- Pages: `AccountsPage.tsx`, `BaselinesPage.tsx`, `NpcsPage.tsx`, `SetupPage.tsx`
- Services: `services/api/inventory.service.ts`, `services/api/npcs.service.ts`, `services/errorLogger.ts`
- Tests (import updates only): `app-sidebar.test.tsx`, `BaselineTargetPicker.test.tsx`
- Config: `eslint.config.js` (scoped-override additions)

**Idempotence fix (`525dfcda5`):** `AttributesPanel.tsx` — Component

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n "as any\|: any\b"` over the full remediation diff — zero matches. |
| FE-02 | No manual class concatenation | PASS | `grep 'className={"'` over the diff — zero matches. No `cn()` usage was touched. |
| FE-03 | No direct API client calls in components | PASS | `grep 'from "@/lib/api/client"'` over the diff — zero matches. |
| FE-04 | No inline Zod schemas in components | PASS (N/A) | No `z.object`/`z.string` touched by this diff; no schema files in scope. |
| FE-05 | No spinners for content loading | PASS (N/A) | No `animate-spin` additions in the diff; loading-state code (Skeleton usage in `NpcImage.tsx`, `NpcShopCard.tsx`) is unchanged by this commit. |
| FE-06 | No hardcoded colors | PASS | `grep -nE "bg-(white|black|gray-[0-9]+|red-[0-9]+)"` over the diff — zero matches. |
| FE-07 | No state mutation | PASS | All 8 "adjust state during render" conversions (`MapImagePanel.tsx:387-394`, `MonsterDropWidget.tsx:415-420`, `NpcImage.tsx:52-61`, `NpcShopCard.tsx:93-97`, `NpcShopCommodityDialog.tsx:54-58`, `ConversationInspector.tsx:268-273`, `NpcConversationCard.tsx:32-36`, `QuestConversationCard.tsx` equivalent) use `setState` calls only, no `.push`/`.splice`/direct mutation. |
| FE-08 | No default exports for components | PASS | `grep "export default function"` over the diff — zero matches; `app-sidebar-items.ts`, `BaselineTargetPicker.utils.ts`, `setup-format.ts` are all named exports. |
| FE-09 | Tenant guard in hooks | PASS (unaffected) | `useAccountByName.ts` (services/atlas-ui/src/lib/hooks/api/useAccountByName.ts:36) retains its pre-existing `enabled: !!tenant?.id && !!name && !timedOut` guard; the diff only added a suppression comment at line 47, not touching tenant-guard logic. |
| FE-10 | Tenant ID in query keys | PASS (unaffected) | `NpcShopCard.tsx:72-77` `shopKey` (pre-existing, untouched by this diff) already includes `activeTenant?.id ?? "no-tenant"`. |
| FE-11 | Error handling with `createErrorFromUnknown` | PASS | `npcs.service.ts:149` improves error fidelity by adding `{ cause: error }` to the re-thrown `Error` rather than discarding it — a strict improvement over the pre-existing pattern, not a regression. `inventory.service.ts:246` `catch {}` (parameter dropped) is a mechanical no-unused-vars fix; the catch body's fail-open behavior (`return false`) is unchanged. |
| FE-12 (see Architecture) | | | |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-14 | Query key factory uses `as const` | PASS (unaffected) | Not touched by this diff; pre-existing `shopKey`/`accountByNameKeys` factories already use `as const` (`NpcShopCard.tsx:77`, `useAccountByName.ts:19-22`). |
| FE-15 / FE-16 | Forms/Zod, schema+inferred type | N/A | No form or schema files in the authored diff. |
| Scoped override correctness | `eslint.config.js` `react-refresh/only-export-components: "off"` glob additions | PASS | `eslint.config.js:38-45` adds `src/pages/**/*-columns.tsx`, `src/components/**/*ErrorBoundary.tsx`, `src/components/**/*Context.tsx` to the existing off-scope. Verified each glob matches real, existing colocated-component files: 12 `*-columns.tsx` files under `src/pages/`, `src/components/common/ErrorBoundary.tsx` + `src/components/features/npc/NpcErrorBoundary.tsx`, `src/components/features/maps/HoverHighlightContext.tsx`. None of these globs are overbroad (no bare `**/*.tsx`). |
| `@typescript-eslint/no-unused-vars` rule scope | Not globally disabling a check | PASS | `eslint.config.js:24-31` scopes the rule to the same `files: ["src/**/*.{ts,tsx}"]` block already governing all other rules — not a narrowing exemption, an additive rule with sensible `^_` ignore patterns for args/vars/caught-errors. |
| 9 inline eslint-disable suppressions | Justified, line-scoped, non-file-wide | PASS | All 9 are `eslint-disable-next-line` (never file-wide `/* eslint-disable */`), each with an inline `--` justification: <br>1. `ConversationCanvas.tsx:186` (`set-state-in-effect`) — `setLaying(true)` genuinely kicks off an async ELK layout in the same effect; verified the effect's cancellation guard (`cancelled` flag) is intact. <br>2. `item-name-cell.tsx:28` (`set-state-in-effect`) — guarded by `if (!tenant \|\| !itemId \|\| itemNameCache.has(itemId)) return` before the flagged line; genuine fetch kickoff. <br>3. `map-cell.tsx` — same pattern, verified equivalent guard. <br>4. `useAccountByName.ts:47` (`set-state-in-effect`) — resets `timedOut` before re-arming a `setTimeout`; genuine effect. <br>5. `useBreadcrumbs.ts:217` (`set-state-in-effect`) — early-exit branch of the same async label-resolution effect (not a separate render-time-derivable value). <br>6. `AccountsPage.tsx:832` (`set-state-in-effect`) — `fetchBanStatuses` is itself the async fan-out with its own loading state. <br>7-9. `debounce.ts:97,148,189` (`exhaustive-deps, use-memo`) — dependency array is `[callback, delay, ...deps]` where `deps: React.DependencyList` is a caller-supplied variadic parameter; a static array literal is structurally impossible for this generic API. Verified via `npx eslint` probe that the `^_` ignore patterns work correctly for array-destructured vars (see below), so none of these 9 could have been trivially avoided by a naming convention instead. |
| Burn-down tracking | Suppressions tracked, not silent | PASS | `docs/TODO.md` (commit `3c662bea1`) enumerates all 9 suppressions by file and rule, with a stated remediation path (migrate ad-hoc fetch/loading hooks to React Query). Not a silent debt. |
| File-splits: exports intact, importers updated | PASS | `app-sidebar-items.ts` ← `app-sidebar.tsx` + `app-sidebar.test.tsx` updated import; `BaselineTargetPicker.utils.ts` ← `BaselineTargetPicker.tsx` + `BaselineTargetPicker.test.tsx` updated import; `setup-format.ts` ← `SetupRow.tsx`, importers `BaselinesPage.tsx` and `SetupPage.tsx` both updated. `grep` for the old import paths across `src/**` returns zero stale references (verified for all three splits). No duplicate/orphaned exports remain in the split-from files. |
| `useSyncExternalStore` rewrite (`use-mobile.tsx`) | Idiomatic, correct | PASS | `subscribe()` (hooks/use-mobile.tsx:5-9) registers/unregisters a `matchMedia` change listener and returns the cleanup; `getSnapshot()` (line 11-13) reads `window.innerWidth` synchronously, returning a primitive (`Object.is`-stable, no snapshot-instability risk). Confirmed the `jsdom` `matchMedia` stub in `src/test/setup.ts:3-15` implements `addEventListener`/`removeEventListener` as no-ops, so `subscribe` doesn't throw under test. Only consumer is `sidebar.tsx` (`useIsMobile`), exercised transitively by `app-sidebar.test.tsx` (36/36 pass, confirmed via targeted re-run above). |
| `useBreadcrumbs` state elimination | Sound — was a pure mirror | PASS | Removed `[globalError, setGlobalError]` was populated by an effect (`useBreadcrumbs.ts` diff) that did nothing but copy `initialBreadcrumbs.error` verbatim; `initialBreadcrumbs` is itself a `useMemo` (line 150-186) keyed on `pathname`. The hook's returned `error: initialBreadcrumbs.error` (line 459) is therefore provably equivalent to the old effect-synced value on every render, with one fewer render-cycle of staleness. |
| `sidebar.tsx` lazy-init | Correct, preserves one-random-value-per-mount | PASS | `React.useState(() => \`${Math.floor(Math.random() * 40) + 50}%\`)` (`ui/sidebar.tsx:659`) — the lazy initializer runs exactly once per mount, matching the old `useMemo(..., [])`'s effective (if not React-Compiler-safe) behavior. |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS (adequate) | The two files with structural test-import changes (`app-sidebar.test.tsx`, `BaselineTargetPicker.test.tsx`) were updated and pass (18 + 18 = 36 tests, confirmed via targeted re-run). The 8 "adjust state during render" conversions are behavior-preserving refactors of existing effects with existing coverage (per Task 3's own review, 887/887 full-suite pass) — not new untested surface. |
| FE-17 (gap, non-blocking) | `use-mobile.tsx` has no dedicated test file | MINOR — pre-existing, not introduced by this diff | `find src -iname "*use-mobile*"` under any `__tests__` pattern returns nothing, and no test file references `useIsMobile` even indirectly by name. The hook's only consumer is `sidebar.tsx`, exercised transitively via `app-sidebar.test.tsx`, which does not vary viewport width, so the `useSyncExternalStore` rewrite's core behavior (recomputing `isMobile` on a `matchMedia` change event) is not directly exercised by any test, before or after this commit. Not a regression — flagging as a should-fix opportunity given the hook's implementation was substantively rewritten. |
| FE-18 | Mocks updated when services changed | PASS (N/A) | No service *interface* changed — `inventory.service.ts` and `npcs.service.ts` changes are internal-only (dropped unused catch binding; added `cause` to a thrown Error). No mock files reference these methods' signatures in a way requiring updates. |

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- FE-17: Consider adding a direct unit test for `useIsMobile` (`services/atlas-ui/src/hooks/use-mobile.tsx`) exercising a simulated `matchMedia` change event, now that the hook was rewritten from `useEffect`+`useState` to `useSyncExternalStore`. Pre-existing gap, not a regression — the hook has never had direct test coverage, and the sole consumer's test (`app-sidebar.test.tsx`) doesn't vary viewport width.
