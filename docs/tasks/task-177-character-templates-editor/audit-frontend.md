# Frontend Guidelines Audit (FE-*) — task-177-character-templates-editor

**Scope:** new/changed TS/TSX in the shared character-templates visual editor, base `092069bce` → head `391a4f979`.
**Note on guideline source:** the packaged `frontend-dev-guidelines` skill docs describe a stale Next.js/Jest-era atlas-ui. The actual atlas-ui (per `services/atlas-ui/CLAUDE.md`) is a Vite + React Router SPA with Vitest (`vi.*`). Findings below are checked against the **real, current** codebase conventions (verified by grepping sibling/precedent files), not the stale skill examples, and deviations from the skill's literal snippets (e.g. query keys without `tenant.id`) are only flagged where they also deviate from actual codebase precedent.

**Gates:** `npm run test` (1094 pass), `npm run build` (tsc -b + vite, 0 errors), `npm run lint` (no new errors) — all reported green per task instructions; not re-run in full. One targeted grep/read pass was done per finding below instead.

## Files audited

- `src/components/features/characters/templates/{jobNames,editorState,previewLoadout,poolSearchConfig}.ts`
- `src/components/features/characters/templates/{ItemSearchCombobox,MapPicker,TemplateSelector,TemplateActionsMenu,IdentitySection,AppearanceThumb,AppearancePoolSection,AppearanceBrowserDialog,ItemRow,EquipmentPoolSection,StartingKitSection,PreviewCard,SaveBar,CharacterTemplatesEditor}.tsx` + `__tests__/*`
- `src/services/api/cosmetics.service.ts`, `src/lib/hooks/api/useCosmetics.ts`, `src/lib/hooks/api/useItemNames.ts`
- `src/components/ui/popover.tsx`
- `src/services/api/characterRender.service.ts` (diff only — `isFemaleCosmeticId` extraction)
- `src/pages/TenantsCharacterTemplatesPage.tsx`, `src/pages/TemplatesCharacterTemplatesPage.tsx` (+ their new tests)

## Mechanical anti-pattern sweep (all in-scope files)

Ran targeted greps for `: any`/`as any`, `className={"` concatenation, `@/lib/api/client` imports, inline `z.object(`, `animate-spin`, hardcoded Tailwind palette classes (`bg-white`, `text-gray-*`, etc.), `.push(`/`.splice(`/`.sort(` mutation, `export default`, `console.log`.

| Check | Result |
|---|---|
| `any` type | **PASS** — zero matches |
| Manual class concatenation | **PASS** — zero matches; all conditional classes use `cn()` (e.g. `AppearanceThumb.tsx:43-47`) |
| Direct `@/lib/api/client` import in components/pages | **PASS** — zero matches; `cosmetics.service.ts:1` goes through `services/api/pagination.ts`'s `fetchAll`, which is the approved service-layer→client path |
| Inline Zod schema in a component | **PASS** — zero matches (no schema at all; see D1 note below) |
| Spinner (`animate-spin`) for content loading | **PASS** — zero matches; `PreviewCard.tsx:98` uses `<Skeleton>` for the image-loading state, not a spinner |
| Hardcoded raw-palette Tailwind colors | **PASS** — zero matches for `bg-white/black/gray-N/...`; `MapPicker.tsx:128` and `AppearancePoolSection.tsx:59` use `text-warning-foreground`, a real semantic token defined in both themes (`src/index.css:33-34,129-131,201-203`) with existing precedent in `src/components/ui/alert.tsx` — not a violation |
| State mutation (`.push`/`.splice`/`.sort` on referenced state) | **PASS** — the three `.sort()` hits (`cosmetics.service.ts:17`, `characterRender.service.ts:85,119`) all sort a fresh spread/derived array (`[...items].sort(...)`, `Object.values(...).sort(...)`), not the original reference. `editorState.ts` reducer uses spread/`filter`/`map` throughout (e.g. `editorState.ts:137-139,182-184`) |
| Default export for a component | **PASS** — zero matches; all named exports |
| `console.log` | **PASS** — zero matches |

## Multi-tenancy / React Query

- `useCosmetics.ts:15-24,26-35` — Pattern B (`useTenant()` + `enabled: !!activeTenant`), matches the codebase's real Pattern-B precedent (`lib/hooks/api/useMaps.ts:27-35`). **PASS**.
- `useItemNames.ts:15-27` — reuses `itemStringKeys.byId`, gated `enabled: !!activeTenant`. **PASS**.
- `cosmeticsKeys` (`useCosmetics.ts:5-9`) does **not** embed `tenant.id`, which looks like a literal FE-10 violation against the skill doc's generic example — but cross-checked against the actual codebase precedent (`useMaps.ts:27-30`, `mapKeys`) which also omits `tenant.id` from Pattern-B keys. This is consistent because `TenantProvider`'s tenant-switch effect calls `queryClient.clear()` (`services/atlas-ui/CLAUDE.md` "Tenant contract" section), wiping the whole cache on every switch — the tenant-id-in-key requirement is redundant under this app's actual invalidation strategy. **PASS** (codebase-consistent, not a real cross-tenant leak).
- `cosmetics.service.ts` and other read services referenced here (`item-strings.service.ts`, `items.service.ts`) don't call `api.setTenant()` per-call; this matches the current codebase's centralized-in-`TenantProvider` tenant injection (explicitly documented as intentional in `services/atlas-ui/CLAUDE.md` — per-call `setTenant` calls elsewhere are "legacy... a follow-up PR will remove them"). **PASS**, not a regression.
- `ItemSearchCombobox.tsx:67-73` and `MapPicker.tsx:37-38` gate their queries on `!!activeTenant` (transitively, via the underlying hooks) and `open`. **PASS**.

## Forms / validation (FE-15/FE-16)

No `react-hook-form` / Zod schema anywhere in this module — by design. `docs/tasks/task-177-character-templates-editor/design.md:78-99` ("D1 — Form state: plain reducer, not react-hook-form") documents this as a deliberate, PRD-permitted decision (the PRD explicitly allows "react-hook-form... or equivalent controlled state"), justified by the near-total absence of free-typed validated inputs (every mutation is a programmatic array op). **N/A, not a violation.**

## Styling

- `[image-rendering:pixelated]` present on every sprite/icon `<img>`: `PreviewCard.tsx:48,118`, `AppearanceThumb.tsx:62`, `ItemSearchCombobox.tsx:149`, `ItemRow.tsx:37`, `StartingKitSection.tsx:31`. **PASS**.
- Light/dark: no manual theme branching found (correct — `next-themes`-equivalent CSS-variable theming handles it); all colors are semantic tokens. **PASS**.
- `PreviewCard.tsx:118` — `drop-shadow-[0_6px_4px_rgba(0,0,0,0.25)]`, a hardcoded arbitrary-value RGBA. See Triage item 2 below. **Minor, not a hard FE-06 violation** — it's a decorative elevation shadow (same category as Tailwind's own built-in `shadow-*` utilities, which are also hardcoded black-rgba under the hood in both themes), not a background/foreground/border color that would break dark-mode legibility.

## JSON:API / service-layer / type checks

- `CharacterTemplate` (`types/models/template.ts:4-17`, unmodified by this branch) is a nested value object inside `TemplateAttributes.characters.templates`, not a standalone top-level resource — correctly has no own `id`/`attributes` wrapper, consistent with JSON:API nested-attribute conventions. **PASS** (pre-existing, out of scope for this task).
- `cosmetics.service.ts` — plain-object export (`export const cosmeticsService = {...}`), matching existing precedent (`jobs.service.ts`, `tenants.service.ts`, `item-strings.service.ts`, `items.service.ts` all use the same pattern). **PASS**.
- `characterRender.service.ts:53-59` — clean extraction of `isFemaleCosmeticId`, `resolveGender` now delegates to it (`characterRender.service.ts:65-69`). No behavior change, no new violation.

## Accessibility — Triage of pre-identified item 1 (TemplateSelector tabs)

**Verdict: real gap, should fix before PR, but not merge-blocking on its own.**

`TemplateSelector.tsx:25-46` uses `role="tablist"` / `role="tab"` / `aria-selected` with **no** roving-tabindex, no arrow-key handler, and (compounding it) **no associated `tabpanel`** — no `aria-controls` on the tab buttons, no `role="tabpanel"`/`aria-labelledby"` on the section below it in `CharacterTemplatesEditor.tsx:187-283`. Per WAI-ARIA APG, using `role="tab"` creates a behavioral contract (arrow-key navigation, single tab-stop for the group, tab↔panel association) that assistive-technology users rely on; none of it is implemented. Confirmed no test exercises arrow-key nav either (`TemplateSelector.test.tsx` only tests click + `role="tab"` presence).

The same ARIA-role-without-required-keyboard-model pattern also appears, slightly less severely, in:
- `ItemSearchCombobox.tsx:108-196` (`role="listbox"`/`role="option"`, per-option `tabIndex={0}` instead of roving tabindex)
- `MapPicker.tsx:75-124` (same `listbox`/`option` pattern)

All three remain **fully keyboard-operable** via plain Tab + Enter/Space (confirmed by reading the `onKeyDown` handlers, e.g. `ItemSearchCombobox.tsx:123-128`, `MapPicker.tsx:86-91`), so this is not a hard keyboard trap — a screen-reader or power-keyboard user who expects native/APG tab or listbox arrow-key behavior will be confused, but nothing is unreachable.

**Recommendation (pick one, both are cheap):**
1. Implement the real APG tabs pattern (roving `tabIndex`, `ArrowLeft`/`ArrowRight` handler, `aria-controls`/`role="tabpanel"`), or
2. Downgrade to non-widget semantics: drop `role="tablist"`/`role="tab"`/`aria-selected`, use a plain `role="group"` (or nothing) with `aria-pressed` on the buttons — matching the component's own doc comment ("Segmented control... no thumbnails by design") which already signals this was conceived as a segmented control, not a full ARIA tab widget. Same treatment for the two listbox instances if not fixed to a real listbox pattern.

Option 2 is the lower-risk fix given the component isn't actually a tabpanel-swap UI in the strict sense (selecting a template just re-renders the section below with new data, not a distinct panel).

## Triage of the other 4 pre-identified items

**2. `PreviewCard.tsx:118` hardcoded `drop-shadow-[0_6px_4px_rgba(0,0,0,0.25)]`.**
Confirmed present. **Verdict: acceptable, non-blocking.** It's a decorative elevation shadow, not a background/text/border color; it doesn't degrade dark-mode legibility (a black shadow on a `<img>` composited sprite reads the same in both themes, same as Tailwind's own `shadow-*` utilities which hardcode rgba(0,0,0,...) too). Optional polish: swap for a preset `drop-shadow-lg`/`drop-shadow-md` utility for consistency with the rest of the design system, but this is style-nit, not a guideline violation.

**3. `TemplateActionsMenu.tsx:70-77` "Remove" `AlertDialogAction` uses default variant, not destructive.**
Confirmed present — `AlertDialogAction` (`components/ui/alert-dialog.tsx:99-108`) hardcodes `buttonVariants()` (primary/default look) unless the caller overrides via `className`. Checked codebase precedent for destructive confirm actions: `pages/TemplatesPage.tsx:297-302` does exactly this override (`className="bg-destructive text-destructive-foreground hover:bg-destructive/90"`) for its own delete-confirmation `AlertDialogAction`. `TemplateActionsMenu.tsx`'s "Remove" action is semantically identical (destructive, irreversible-until-resave) and doesn't follow that established local pattern. **Verdict: should fix before PR** — it's a one-line `className` addition, and the current default-primary "Remove" button next to a `Cancel` reads ambiguous (which one is safe?), inconsistent with the codebase's own convention for this exact interaction shape.

**4. `StartingKitSection.tsx` `SkillRow` (lines 18-51) degrade/remove branches untested.**
Confirmed. `StartingKitSection.test.tsx` never clicks the skill row's `Remove skill {id}` button and never simulates the icon `onError` (Sparkles fallback). Cross-checked the claim "ItemRow's equivalent IS tested": `ItemRow` (a separate, exported, reused component) has its remove-click path tested via `EquipmentPoolSection.test.tsx:55-72` ("degrades to Unknown item... still removable"). `SkillRow`, by contrast, is a local, unexported component defined only inside `StartingKitSection.tsx` with **no other consumer** — so its remove-click and icon-fail fallback are 100% untested anywhere in the repo. **Verdict: real coverage gap, should close** — Minor/non-blocking (the component still has a test file and 3 passing cases; this is a missing-branch gap, not a missing-test-file gap), but cheap to add (one `userEvent.click` on the skill row's remove button, one `fireEvent.error` on its `<img>`).

**5. `cosmetics.service.ts:14-17` `Number.parseInt` partial-numeric leak.**
Confirmed: `Number.parseInt("30030abc", 10)` → `30030`, passes `Number.isFinite`, survives the filter. **Verdict: acceptable as documented, non-blocking.** The comment (`cosmetics.service.ts:5-6`) correctly scopes this to a live-verified backend contract (pure-numeric ids only), and the consequence of an actual malformed id (extremely unlikely given the backend never emits one) is a broken thumbnail render, not data corruption or a security issue. Low risk is accurately characterized. Optional hardening (`/^\d+$/.test(row.id)` instead of `Number.isFinite`) would be strictly more correct but isn't required.

## Additional finding beyond the pre-identified 5

**6. Search-query failures silently render as "no results" instead of being surfaced.**
`ItemSearchCombobox.tsx:67-73` has no `query.isError` branch — when `itemsService.searchItems` rejects, `query.data` stays `undefined`, `rows` resolves to `[]` (`ItemSearchCombobox.tsx:75-80`), and the UI falls through to the generic `"No matches."` message (`ItemSearchCombobox.tsx:188-195`), indistinguishable from a real empty search. Same pattern in `MapPicker.tsx`: `results` (`useMapsByName`) has no `isError` handling at all — only `current` (the single by-id lookup, `MapPicker.tsx:44`) surfaces an error state ("not found in map data for this version"). Confirmed no test in `ItemSearchCombobox.test.tsx` or `MapPicker.test.tsx` exercises a rejected search query. **Verdict: Minor, should fix but non-blocking** — the manual-id fallback (`manualId`, typing a raw numeric id) remains available as an escape hatch even when search is down, so this degrades gracefully rather than blocking the operator; still, a distinct "search failed, try an id" message would be more honest than a bare "No matches."

## Testing

- FE-17 (tests exist for changed components): **PASS** for all 18 new components/modules — either a dedicated `__tests__/*.test.{ts,tsx}` file, or (for `poolSearchConfig.ts`, `AppearanceThumb.tsx`) thorough exercise through consumer test files (`ItemSearchCombobox.test.tsx` for pool-search client-filtering; `AppearancePoolSection.test.tsx` + `AppearanceBrowserDialog.test.tsx` for `AppearanceThumb`'s select/remove/marked/pressed states).
- Tests use `vi.mock`/`vi.fn` (Vitest), not Jest — correct for this codebase (confirmed via `TemplateSelector.test.tsx:1-4`, `StartingKitSection.test.tsx:1-3`, etc.).
- FE-18 (mocks updated when services changed): `characterRender.service.ts` gained `isFemaleCosmeticId`; consumers (`AppearanceBrowserDialog.tsx:15`) import it directly (not mocked in that file's test — real function used), so no stale mock risk found.
- SaveBar's save/discard wiring is covered end-to-end via `CharacterTemplatesEditor.test.tsx:136-168` (save passes working array + resets dirty; discard reverts to baseline; discard restores nearest-valid tab), not via a standalone `SaveBar.test.tsx` — acceptable, this is genuine integration coverage of the same behavior, not a gap.

## Summary

### Blocking (must fix before PR)
- None. No FE-* mechanical anti-pattern failures, build/tests reported green, no cross-tenant cache leak, no missing tenant guards.

### Should fix (non-blocking, cheap, recommended before merge)
- **Item 3** — `TemplateActionsMenu.tsx:70-77`: give the "Remove" `AlertDialogAction` the destructive styling used elsewhere in the codebase (`pages/TemplatesPage.tsx:300` precedent).
- **Item 1 (a11y)** — `TemplateSelector.tsx:25-46` (+ same pattern in `ItemSearchCombobox.tsx:108-196`, `MapPicker.tsx:75-124`): resolve the ARIA-role/keyboard-model mismatch — either implement roving tabindex + arrow-key nav (and tabpanel association for the tablist), or drop the widget roles for plain segmented-control/list semantics.
- **Item 4** — `StartingKitSection.tsx` `SkillRow` (lines 18-51): add remove-click and icon-onError test coverage.
- **Finding 6** — `ItemSearchCombobox.tsx:67-80`, `MapPicker.tsx:38` (`results`): surface `isError` distinctly from "no results".

### Non-blocking / acceptable as-is
- Item 2 — `PreviewCard.tsx:118` arbitrary drop-shadow RGBA (decorative, theme-safe).
- Item 5 — `cosmetics.service.ts:14-17` lenient `Number.parseInt` (documented, low-risk, backend-guaranteed pure-numeric ids).
