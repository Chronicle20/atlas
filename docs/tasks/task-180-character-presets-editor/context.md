# Character Presets Editor — Implementation Context

Companion to [`plan.md`](plan.md). Read this first for the mental model, then execute the plan task-by-task. Consumes [`prd.md`](prd.md) and [`design.md`](design.md).

## What this task is

Replace two wholesale-duplicated raw-ID character-preset forms with **one shared, adapter-parameterized visual editor** under `services/atlas-ui/src/components/features/characters/presets/`, structurally mirroring the landed task-177 character-**templates** editor but with a **card-library landing view** in front of a focused per-preset editor. **atlas-ui only** — zero Go/backend/seed changes, zero routing/shell/layout changes.

**Preset vs. template (the shape difference that drives every decision):**
- A **template** (task-177) stores per-dimension **pools** (`faces[]`, `hairs[]`, `tops[]`, …) keyed by `(jobIndex, subJobIndex, gender)`, addressed by array **index** (`?tpl=<index>`), with a UI-only `previewPicks` layer for "which pool entry to preview".
- A **preset** (this task) stores **a single concrete character** (`face`, `hair`, `hairColor`, `skinColor`, one worn loadout, a verbatim stat block, granted inventory/skills) with a stable backend **`id`**, addressed by id/**key** (`?preset=<id|key>`). Appearance edits write **straight to `attributes`** — there is NO `previewPicks` layer, which removes an entire class of index-remap bugs the template reducer carries.

Because the shapes diverge, this is a sibling `presets/` folder that **reuses task-177's leaf modules** (renderers, pickers, thumbnails, combobox, browser) but not its container.

## Where things live

- **Worktree:** `.worktrees/task-180-character-presets-editor` (branch `task-180-character-presets-editor`). Run everything from here.
- **New code:** `services/atlas-ui/src/components/features/characters/presets/` (+ colocated `__tests__/`).
- **Task-177 reuse source:** `services/atlas-ui/src/components/features/characters/templates/`.
- **Prototype (visual source of truth):** `docs/tasks/task-180-character-presets-editor/prototype.html`, "A · Card library → editor" tab.

## Build order rationale

Bottom-up so each task consumes only already-built modules, TDD throughout:
1. **Pure logic first** (Tasks 1–3): `presetJobs.ts`, `presetEditorState.ts` (reducer — the correctness core), `presetLoadout.ts`. Fastest, highest-value unit tests.
2. **Shared-file additive edits** (Tasks 4–6): generalize `AppearanceBrowserDialog`, add `ItemRow.trailing`, add `ApplyPresetDialog.initialPresetId`. Task-177 tests are the regression gate.
3. **Leaf UI** (Tasks 7–14): preview card + 7 sections, each independently testable.
4. **Assembly** (Tasks 15–19): focused editor → library card → library → account picker → container.
5. **Wiring + deletion** (Task 20), **verification gate** (Task 21).

## Key files read during planning (ground truth)

| Concern | File | Note |
|---|---|---|
| Reducer pattern to mirror | `templates/editorState.ts` | index-addressed + `previewPicks`; ours is key-addressed, no picks |
| Container/URL-sync pattern to mirror | `templates/CharacterTemplatesEditor.tsx` | seed-once + deep-link-on-load effects + `syncSelection`; DO copy the router-race avoidance |
| Adapter shape + page wiring | `pages/TenantsCharacterTemplatesPage.tsx`, `TemplatesCharacterTemplatesPage.tsx` | our adapter adds optional `apply` |
| Type model (single source) | `types/models/template.ts` | `CharacterPreset`, `CharacterPresetAttributes`, nested entry types |
| Validation schema (reuse) | `lib/schemas/character-presets.schema.ts` | `presetSchema` — name ≤64, desc ≤512, level 1..250 |
| Forms being deleted | `pages/tenants-character-presets-form.tsx` (+ templates twin) | source of `DEFAULT_PRESET_ATTRIBUTES` and the `meta.path → presets[<id>].<field>` error-mapping logic |
| Cosmetics browser (generalize) | `templates/AppearanceBrowserDialog.tsx` | currently coupled to `CharacterTemplate`; Task 4 adds add/replace modes |
| Thumbnail (reuse + `selected`) | `templates/AppearanceThumb.tsx` | |
| Item row (add `trailing`) | `templates/ItemRow.tsx` | icon + name + mono id + × |
| Preview card to mirror | `templates/PreviewCard.tsx` | `useCharacterImage` machinery, worn-icon strip |
| Loadout math to mirror | `templates/previewLoadout.ts` | `RENDER_DEFAULT_*`, v83 hair = base+color |
| Slot placement | `lib/utils/maplestory.ts` | `getDefaultSlotForTemplateId(id): number \| null` (100xx=hat −1 … weapons 130–159 = −11) |
| Cash/pet/mount drop | `services/api/characterRender.service.ts` | `filterEquipment(Record<string,number>)`, `CharacterLoadout` type, `isFemaleCosmeticId`, `generateCharacterUrl` |
| Apply dialog (add `initialPresetId`) | `characters/ApplyPresetDialog.tsx` | requires a live `Tenant`, worlds, account service — tenant-only |
| Account search | `lib/hooks/api/useAccounts.ts:129` | `useAccountSearch(tenant, namePattern) → Account[]`; `Account.id` is a **string** → `Number(id)` for `ApplyPresetDialog.accountId: number` |
| Apply-tenant source | `lib/hooks/api/useTenants.ts:66` | `useTenant(id): TenantBasic`; `Tenant = TenantBasic` (usable directly) |
| Search filters | `templates/poolSearchConfig.ts` | `POOL_SEARCH_CONFIGS`, `SearchPoolKey` = tops/bottoms/shoes/weapons/items |

## Decisions locked (from design §3 + planning verification)

- **Skin range 0–9**, hair-color **0–7** — matches task-177's `AppearanceBrowserDialog` constants exactly.
- **Apply a dirty preset:** warn "applies last saved" + offer save-first (non-blocking); a preset with **no persisted `id`** cannot be applied (disable + hint "Save this preset before applying").
- **Apply-to-account is tenant-context-only** — templates have no `Tenant`/account service/worlds. The adapter's `apply?: { tenant }` is present only on the tenant page; the template page omits it and the Apply affordances hide.
- **Job names** = curated client map (`presetJobs.ts`); no atlas-data job-name endpoint exists. Unknown → `Job <id>`.
- **`?preset=` key vs. backend id after save:** a freshly-added preset keeps its `local-<n>` key in the URL for the session (the `loaded` guard prevents reseed) — same accepted limitation as task-177's post-save index.

## Critical gotchas (do not relearn the hard way)

1. **Page export names are load-bearing.** `App.tsx:180-182,218-220,341,367` lazy-imports `TenantsCharacterPresetsPage` / `TemplatesCharacterPresetsPage` **by name**. Rewrite their bodies; never rename the exports.
2. **URL-sync router race (task-177's scar).** Do NOT add a `presets.length`-watching effect that re-derives `?preset=`. Each mutation handler calls `syncSelection` with the reducer's own post-mutation selection (`open`/`duplicate`/`add` → known target key; `remove`/`discard` → `null`). The deep-link effect runs **once** on load (`deps:[state.loaded]`) with an `eslint-disable exhaustive-deps` note, exactly like `CharacterTemplatesEditor.tsx:86-109`.
3. **`key` must never leak into persistence.** `isDirty`, `projectForSave`, and the `save` PATCH all project each `WorkingPreset` to `{ id?, attributes }` — no `key`. The reducer test asserts this.
4. **Save must spread `attributes.characters`.** Both adapters PATCH `{ characters: { ...existing, presets } }` so the sibling `templates` array survives. Both page tests assert this.
5. **Additive-only shared-file edits.** `AppearanceBrowserDialog` (Task 4), `ItemRow` (Task 5), `ApplyPresetDialog` (Task 6) get backward-compatible props with defaults; task-177's tests are the green gate. If the browser prop refactor proves costlier than the tests can cover, design §8 authorizes an Option-B `PresetCosmeticsBrowser` fallback and §9 a `PresetItemRow` fallback — but Option A (generalize) is the plan of record.
6. **`getDefaultSlotForTemplateId` returns `number | null`** — unplaceable ids (use/etc/cash) return `null` and are skipped for the render but still shown in their section list. `filterEquipment` additionally drops cash (−101..−114)/pet/mount slots.
7. **Delete the orphaned form test.** `pages/__tests__/templates-character-presets-form.test.tsx` imports the deleted form and must be removed with it (Task 20). The `pages/character-presets-schema.ts` re-export shim has no importers (verified) and is deleted too.
8. **Node 22 via nvm** for all `npm` commands. `npm run build` type-checks new test files (green test ≠ green build). Verification gate also needs `tools/lint.sh --check` clean at the worktree root.

## Verification gate (Task 21 / PRD Acceptance)

From `services/atlas-ui` (nvm node 22): `npm run test`, `npm run lint` (no new errors vs. baseline), `npm run build` all clean; and `tools/lint.sh --check` clean from the worktree root.

## Out of scope (do not do)

Character Templates section; any `DetailSidebar`/layout/routing/nav change; any Go service change; seed preset JSON changes; a skill *browser*; client-side library sort/reorder; the Option-B/Option-C prototype directions.
