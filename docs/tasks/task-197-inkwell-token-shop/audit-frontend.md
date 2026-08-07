# Frontend Audit — task-197-inkwell-token-shop

- **Audit Scope:** services/atlas-ui/src changes on task-197-inkwell-token-shop (merge-base `31c7a664f`, HEAD `52a878b13`) — item-search extraction (`useItemSearch`, `ItemSearchResults`, `ItemPicker`), `ItemSearchCombobox` rewrite, `poolSearchConfig.ts` move, `EquipmentSection.tsx` import fix, `NpcShopCommodityDialog.tsx` item-picker wiring, plus new tests.
- **Guidelines Source:** frontend-dev-guidelines skill (FE-* checklist)
- **Date:** 2026-08-06
- **Build:** PASS (verified by prior agent — see `.superpowers/sdd/plan/task-9-gates-report.md` Step 5; not re-run, no doubt raised by code reading)
- **Tests:** 1400 passed, 0 failed (194/194 files) — per same gates report
- **Overall:** NEEDS-WORK (build/tests clean; two guideline deviations found, both mitigated/precedented, no blocking correctness bug)

## Build & Test Results

Not re-run — reused the prior agent's verified output (`npm run build` exit 0, `npm run test` 194 files / 1400 tests passed, `tools/lint.sh --check` clean apart from 6 pre-existing warnings in untouched files). Reading the diff raised no doubt that would justify a targeted re-run.

## File Inventory

- **Component (new):** `src/components/features/items/item-search/useItemSearch.ts` — headless search hook (React-Query-backed; classified as Hook-in-a-component-dir, see FE-14 finding)
- **Component (new):** `src/components/features/items/item-search/ItemSearchResults.tsx` — presentational `<ul role="listbox">`
- **Component (new):** `src/components/features/items/item-search/ItemPicker.tsx` — value-bearing field
- **Component (new, test):** `src/components/features/items/item-search/__tests__/ItemPicker.test.tsx`
- **Component (rewritten):** `src/components/features/characters/templates/ItemSearchCombobox.tsx` — thin shell over the two extracted pieces
- **Component (import-path-only edit):** `src/components/features/characters/presets/EquipmentSection.tsx`
- **Component (edit):** `src/components/features/npc/NpcShopCommodityDialog.tsx` — two fields converted to item pickers
- **Component (new, test):** `src/components/features/npc/__tests__/NpcShopCommodityDialog.test.tsx`
- **Other (rename, contents unchanged):** `src/lib/items/poolSearchConfig.ts` (from `.../characters/templates/poolSearchConfig.ts`)
- **Regression harness (verified UNCHANGED):** `src/components/features/characters/templates/__tests__/ItemSearchCombobox.test.tsx` — `git diff --stat 31c7a664f..52a878b13` and `git log` over this path both empty. This is the load-bearing proof that the extraction preserved DOM/behaviour.
- **Verified NOT edited (as required):** `src/components/features/npc/NpcShopCard.tsx` — absent from `git diff --stat 31c7a664f..52a878b13 -- services/atlas-ui/src`, confirming the dialog's only consumer needed no change.

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ": any\|as any"` over all 9 in-scope files: zero matches. |
| FE-02 | No manual class concatenation | PASS | `grep -n 'className={"'`: zero matches. All conditional classes use ternaries assigned directly to `className` (e.g. `ItemSearchResults.tsx:51-55`) or plain template-free strings — no `+`-concatenation found. |
| FE-03 | No direct API client calls in components | PASS | `grep -n 'from "@/lib/api/client"'`: zero matches in any component/page in scope. `useItemSearch.ts:3` imports `itemsService`, not the raw client. |
| FE-04 | No inline Zod schemas in components | PASS | `grep -n 'z\.object(\|z\.string(\|z\.number('`: zero matches. No Zod usage at all in this feature (see FE-15/FE-16 — N/A, no forms use schemas here). |
| FE-05 | No spinners for content loading | PASS | `grep -n 'animate-spin'`: zero matches. `ItemSearchResults.tsx:104-106` uses a text row ("Searching…") for the loading state, not a spinner — acceptable for a compact popover list. |
| FE-06 | No hardcoded colors | PASS | `grep -nE 'bg-(white\|black\|gray-[0-9]\|red-[0-9]\|green-[0-9]\|blue-[0-9]\|slate-[0-9]\|zinc-[0-9])'`: zero matches. `ItemSearchResults.tsx:108` uses `text-warning-foreground`, a real semantic token defined for both themes at `src/index.css:34,143,215`. |
| FE-07 | No state mutation | PASS | `grep -nE '\.push\(\|\.splice\(\|\.sort\('`: zero matches. All updates in `useItemSearch.ts` (`setSettled`), `NpcShopCommodityDialog.tsx` (`setForm`) use spread (`{...prev, [key]: next}`, `(s) => ({...s, page: s.page + 1})`). |
| FE-08 | No default exports for components | PASS | `grep -n 'export default function'`: zero matches. All components use named exports (`ItemPicker.tsx:32`, `ItemSearchResults.tsx:19`, `useItemSearch.ts:37`). |
| FE-09 | Tenant guard in hooks | PASS | `useItemSearch.ts:42,76` — `const { activeTenant } = useTenant();` ... `enabled: open && !!activeTenant && settled.term.trim().length > 0` (Pattern B). `useItemStrings.ts:11,18` (dependency, unchanged) — `const { activeTenant } = useTenant();` ... `enabled: !!itemId && !!activeTenant`. Binding constraint on the item-0 guard verified: `ItemPicker.tsx:46` and `NpcShopCommodityDialog.tsx:59` both call `useItemName(value > 0 ? String(value) : "")`, so `useItemStrings.ts:18`'s `!!itemId` never sees a truthy `"0"`. |
| FE-10 | Tenant ID in query keys | **FAIL** | `useItemSearch.ts:74` — `queryKey: ["item-search", poolKey, settled.term, settled.page]` omits `activeTenant?.id`. Item search results are tenant-scoped (different tenants can have different item catalogs per region/version), so per the documented pattern (`patterns-react-query.md` "Always include tenant ID in query keys") this key should read `[..., activeTenant?.id ?? "no-tenant", ...]`. **Mitigation found:** `context/tenant-context.tsx:64` calls `queryClient.clear()` on every active-tenant change, which purges this cache on tenant switch, closing most of the practical cross-tenant-staleness window. **Precedent found:** the same omission exists in the pre-existing `SkillSearchCombobox.tsx:47` (`queryKey: ["skill-search", settled.term, settled.page]`), so this is a consistent (if non-compliant) local convention for search-combobox hooks, not a one-off oversight. Still a documented-guideline violation on a new file — recommend adding `activeTenant?.id ?? "no-tenant"` to the key before merge, or documenting the `queryClient.clear()` mitigation inline so the omission reads as deliberate rather than missed. |
| FE-11 | Error handling with `createErrorFromUnknown` | N/A / PASS | No manual `.then().catch()` async flows were introduced. `useItemSearch.ts` errors are TanStack Query's native `isError`, surfaced to the user at `ItemSearchResults.tsx:107-111` ("Search failed — enter an id manually"). `NpcShopCommodityDialog.tsx:85-92`'s `handleSubmit` has no `catch` — it delegates to the caller-supplied `onSubmit`, and the only consumer (`NpcShopCard.tsx`, unchanged/out of scope) already wraps its own calls in try/catch with toast feedback. No new unhandled-rejection path introduced. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS (N/A for new code) | `CommodityAttributes` (`types/models/npc.ts:43-53`, unchanged this branch) already followed `{id, attributes}` via `Commodity` (`npc.ts:37-41`); this branch didn't touch model types. `ItemSearchResult` (`types/models/item.ts:65-71`, unchanged) is a flattened search-result DTO, not a domain model — produced by `items.service.ts:95-101`'s transform of the real JSON:API response, which is the documented service-layer transform pattern. |
| FE-13 | Service extends `BaseService` | N/A | No service files changed on this branch. |
| FE-14 | Query key factory uses `as const` | **FAIL** | `useItemSearch.ts:74` inlines a bare array literal directly into `useQuery` — no exported `itemSearchKeys` factory, no `as const`. Per `patterns-react-query.md`, hook files should export a hierarchical key factory (`export const xKeys = { ... } as const`); this hook also lives in `components/features/items/item-search/` rather than `lib/hooks/api/` (the documented location for React Query hooks per the architecture-overview file-responsibility table). Same precedent caveat as FE-10: `SkillSearchCombobox.tsx:47` follows the identical inline-key, no-factory, component-colocated pattern, so this is a consistent local convention for lightweight combobox-search hooks rather than a novel regression. Non-blocking given the precedent, but worth flagging for a follow-up standardization pass since two instances now exist. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | FAIL, pre-existing (not introduced by this branch) | `NpcShopCommodityDialog.tsx:74` uses raw `useState<CommodityAttributes>` instead of `useForm` + `zodResolver`. Confirmed via `git diff 31c7a664f..52a878b13 -- .../NpcShopCommodityDialog.tsx`: the pre-branch version already used the same raw-`useState` `form`/`setForm` pattern with plain `<Input type="number">` fields — this branch only swaps two fields' control type (`Input` → `ItemPicker`) without touching the form-management approach. Not new debt from this PR; flagged as pre-existing and non-blocking for this audit. |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No Zod schema involved in this feature. |

## Accessibility Deep-Dive (requested scrutiny)

The `<Label htmlFor>`-avoids-the-picker-trigger approach in `NpcShopCommodityDialog.tsx:118-143` was checked against all four points requested:

1. **`aria-labelledby` target exists** — `NpcShopCommodityDialog.tsx:104-105,128` — `labelId = \`${controlId}-label\`` is passed as both the wrapping `role="group"`'s `aria-labelledby` (line 125) and the `<Label id={labelId}>` (line 128). The id is generated from the same `controlId` in the same render pass, so it is always present — confirmed PASS.
2. **Group's computed accessible name is the field label** — `role="group" aria-labelledby={labelId}` (line 124-125) resolves its accessible name to the referenced `<Label>` element's text content ("Template ID" / "Token Template ID") per the `aria-labelledby` accessible-name computation algorithm. `components/ui/label.tsx:16-20` confirms `Label` renders Radix's `<label>`-tag primitive with no `htmlFor`, so it carries only text content, not implicit control association — correct for this use as a pure name source.
3. **Trigger button keeps its own text as its accessible name** — `ItemPicker.tsx:66-75` renders `<Button {...(id ? {id} : {})}>{label}</Button>` — no `aria-label`/`aria-labelledby` is applied to the button itself, and the wrapping `role="group"` does not propagate a name down to descendant interactive elements (per ARIA, `aria-labelledby` on a container does not override a nested control's own accessible name). Confirmed by both test suites asserting on the button's own computed name: `NpcShopCommodityDialog.test.tsx:80` (`getByRole("button", { name: "Select an item…" })`), `:124` (`{ name: "Perfect Pitch · 4310000" }`), and `ItemPicker.test.tsx:60,69,78,88` (analogous assertions). If the `<Label>` had instead used `htmlFor={controlId}` pointing at the button, these exact assertions would have failed (HTML-AAM would substitute the label text for the button's own content) — the passing test suite is direct evidence the override is correctly avoided.
4. **Numeric input rows kept ordinary `htmlFor` association** — `NpcShopCommodityDialog.tsx:146,151` — for `kind === "number"` rows (and the read-only edit-mode Template ID row, `:146,168`), the code takes the `else` branch and renders plain `<Label htmlFor={controlId}>` + `<Input id={controlId}>` (or `<ResolvedItemName id={controlId}>`), unchanged from the pre-picker version. `NpcShopCommodityDialog.test.tsx:89` (`screen.getByLabelText("Token Price")`) exercises this path directly.

**Verdict: PASS.** The `role="group"`/`aria-labelledby` pattern is the standards-correct choice given the hard constraint (button must keep its own text as name, but the field still needs a discoverable label) — it matches the WAI-ARIA APG "labelled group of controls" pattern. One non-blocking implementation note: the group wrapper uses `className="contents"` (`NpcShopCommodityDialog.tsx:126`) purely for CSS Grid layout (so the 4-column grid track assignment isn't broken by the extra wrapping `div`). `display: contents` removed elements from the accessibility tree in some older browser engines (a since-fixed bug, not current behavior in evergreen Chrome/Firefox/Safari) — worth a one-line code comment noting the dependency on modern `display: contents` a11y-tree behavior, but not a guideline violation today.

One test-coverage gap, non-blocking: neither test file asserts the group's accessible name directly (e.g. `screen.getByRole("group", { name: "Template ID" })`) — coverage currently proves the button's name is *not* overridden by omission (the assertions above would fail if it were), but doesn't positively assert AT users can discover "Template ID" via the group. Worth adding one such assertion in a follow-up.

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | `ItemPicker.test.tsx` (new, 8 cases) covers `ItemPicker.tsx` directly. `NpcShopCommodityDialog.test.tsx` (new, 4 cases) covers the dialog's picker wiring, payload shape/order, and edit-mode read-only rendering. `useItemSearch.ts` and `ItemSearchResults.tsx` have no dedicated unit test file, but are fully exercised transitively through both new suites and — most importantly — the **unchanged** `ItemSearchCombobox.test.tsx` regression harness, which is the explicit proof the extraction preserved behavior (verified unchanged via `git diff --stat`/`git log`, see File Inventory). |
| FE-18 | Mocks updated when services changed | N/A | No service files changed on this branch (`items.service.ts`, `item-strings.service.ts` untouched per `git log 31c7a664f..52a878b13`). Both new test files mock `itemsService.searchItems` and `useItemName` inline (`ItemPicker.test.tsx:6-14`, `NpcShopCommodityDialog.test.tsx:6-14`) matching the existing exported signatures. |

## Binding-Constraint Verification (from the audit brief)

- **Regression harness unchanged:** CONFIRMED — `git diff --stat 31c7a664f..52a878b13 -- .../ItemSearchCombobox.test.tsx` and `git log` over the same path both empty.
- **`exactOptionalPropertyTypes` idiom honored:** CONFIRMED — every optional-prop pass-through in scope uses conditional spread, never `prop={undefined}`: `ItemPicker.tsx:72` (`{...(id ? { id } : {})}`), `ItemPicker.tsx:91-111` (`{...(allowClear ? { leadingRow: (...) } : {})}`), `NpcShopCommodityDialog.tsx:138-140` (`{...(key === "tokenTemplateId" ? { allowClear: true, placeholder: "None" } : {})}`), `useItemSearch.ts:68-70` (filter object conditional spreads). `tsconfig.app.json` confirms `"exactOptionalPropertyTypes": true` is genuinely on (line 21) — this project's own `services/atlas-ui/CLAUDE.md` claims it's "off pending a follow-up," which per project memory (`bug_atlas_ui_claude_md_strict_flags_stale.md`) is known-stale; the tsconfig read here is authoritative.
- **`useItemName` item-0 gate:** CONFIRMED — see FE-09 above; both call sites guard with `value > 0 ? String(value) : ""` before the hook's own `!!itemId` check.
- **Commodity payload shape/order unchanged:** CONFIRMED — `NpcShopCommodityDialog.tsx:16-24` (`EMPTY`) and `:43-54` (`FIELDS`) preserve `templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimit` exactly, matching `CommodityAttributes` (`types/models/npc.ts:43-53`, unchanged). `setForm((prev) => ({ ...prev, [key]: next }))` spread-updates never reorder keys. Tests `NpcShopCommodityDialog.test.tsx:96-104,130,140` assert the exact object shape submitted.
- **`NpcShopCard.tsx` needs no edit:** CONFIRMED — absent from the branch's changed-file list (`git diff --stat 31c7a664f..52a878b13 -- services/atlas-ui/src`).

## Summary

### Blocking (must fix)
None. No FE-* violation found rises to a build break, a real cross-tenant data leak, or a broken accessible-name contract. Both FAIL items below are guideline deviations with either an active mitigation or an established precedent, and this audit doesn't treat "matches an existing, also-noncompliant local convention" as license to wave them through silently — they're reported as FAIL so a human decides whether to fix now or track.

### Non-Blocking (should fix)
- **FE-10** — `useItemSearch.ts:74`'s query key omits `activeTenant?.id`. Add it (`["item-search", poolKey, activeTenant?.id ?? "no-tenant", settled.term, settled.page]`) or add a code comment citing the `queryClient.clear()` mitigation (`context/tenant-context.tsx:64`) so the omission reads as a deliberate, documented tradeoff rather than a miss. Low risk today (mitigated), but the pattern is now used in two places (`SkillSearchCombobox.tsx`, `useItemSearch.ts`) and will keep propagating if not corrected once.
- **FE-14** — `useItemSearch.ts:74` has no exported key factory / `as const`, and the hook lives outside `lib/hooks/api/`. Consistent with the `SkillSearchCombobox.tsx` precedent; consider a follow-up standardization task for search-combobox hooks specifically (they're a distinct, lighter-weight category from full-resource hooks) rather than fixing ad hoc.
- **FE-15** — `NpcShopCommodityDialog.tsx` doesn't use `react-hook-form`/`zodResolver`; pre-existing, not introduced by this branch — no action required for this PR, but don't use this dialog as a template for new forms.
- **Accessibility** — add a `screen.getByRole("group", { name: "Template ID" })`-style assertion to positively verify the group's accessible name (currently only proven indirectly by absence of override), and consider a one-line comment on the `className="contents"` wrapper noting its dependency on modern `display: contents` accessibility-tree behavior.
