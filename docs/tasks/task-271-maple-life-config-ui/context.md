# task-271 — Planning Context

Companion to `plan.md`. What the plan assumes, why, and where it deviates from
`design.md`.

## Scope

Frontend-only, entirely inside `services/atlas-ui`. **No Go file changes.** Both
REST models already carry the block
(`services/atlas-configurations/atlas.com/configurations/tenants/rest.go:23`,
`.../templates/rest.go:23`), and `atlas-character-factory` is a read-only source
of validation rules. 13 tasks: 15 new files, 9 modified.

## Key files (verified against the branch, not assumed)

| File | Role |
|---|---|
| `src/types/models/template.ts` (129 lines) | Domain types. `TemplateAttributes` ends at `:124`; no `mapleLife` today. The two preset entry types are declared at `:32,:37` and used at `:63,:64`. |
| `src/services/api/tenants.service.ts` | `TenantConfigAttributes` at `:75-132`; `cashShop?` precedent at `:115-131`; the whole-attributes merge at `:312`. |
| `src/components/DetailActionBarContext.tsx` | `DetailActionBarConfig` at `:19-24`; the push effect's dependency list at `:82`; `<SaveBar>` at `:93-100`. Only two production callers: `CharacterTemplatesEditor.tsx:159`, `CharacterPresetsEditor.tsx:181`. |
| `src/components/features/characters/templates/AppearancePoolSection.tsx` | Props at `:15-27`; couples to `CharacterTemplate` at `:49`, `PICK_KEY_BY_POOL` at `:48`, `buildVariantLoadout` at `:75`. Only caller is `CharacterTemplatesEditor.tsx:219-253`. |
| `src/components/features/characters/templates/CharacterTemplatesEditor.tsx` | The reference editor: seed-once guard `:66-70`, deep-link effect `:86-109`, action-bar registration `:159-169`, two-column layout `:194-203`. |
| `src/components/features/characters/presets/presetEditorState.ts` | The `project` / `projectForSave` / `isDirty`-by-serialisation precedent at `:126-151`. |
| `src/lib/schemas/character-presets.schema.ts` | The Zod convention: private element schemas composed into one export, `safeParse` at the call site. |
| `src/services/api/__tests__/templates-update.test.ts:1-47` | The service-test mock shape every FR-1.3/1.4 test copies. |

Hooks live in `src/lib/hooks/api/`, **not** in the service files:
`useTenantConfiguration` / `useUpdateTenantConfiguration` in `useTenants.ts`;
`useTemplate` / `useUpdateTemplate` / `useTemplates` /
`useTemplatesByRegionAndVersion` in `useTemplates.ts`. `usePresetJobOptions` is
at `src/lib/hooks/usePresetJobOptions.ts:38`, one level up from `api/`.

## Confirmed facts the plan depends on

- `WarriorImprovedMaxHpIncreaseId = 1000001`
  (`libs/atlas-constants/skill/constants.go:2933`),
  `MagicianImprovedMaxMpIncreaseId = 2000001` (`:3023`). These are the only two
  ids with a coded prerequisite (`factory/maple_life.go:33-42`).
- The validator the page mirrors is
  `factory/processor.go:388-437`, read in full. FR-11.6's `spPool[0] < 6`
  threshold is derived from `needed := int(in.SP); if hasPrereq && in.SP > 0 {
  needed += 5 }; if ... needed > pool[0] { return ErrSPInvalid }` at `:428-437`
  — with the smallest non-zero investment (`in.SP == 1`), `needed == 6`.
- `parseSPPool` (`factory/maple_life.go:142`) accepts **any** comma count and
  reads only `pool[0]`. "Exactly ten" is a UI convention matching what
  atlas-character persists, stricter than the server. The FR-11.5 message says
  so.
- Real seed data exists at
  `services/atlas-configurations/seed-data/templates/template_gms_{83,87,92,95}_1.json`
  (`attributes.mapleLife`): 2 look rows, 10 class rows. The gender-0 look row is
  `faces [20000,20001,20002], hairs [30030,30020,30000], hairColors [0,7,3,2],
  skinColors [0,1,2,3]` — all hairs are already multiples of 10, so the shipped
  data validates clean. Class 0 is `jobId 100, level 30, ap 123, sp
  "61,0,0,0,0,0,0,0,0,0", spSkillId 1000001, meso 100000`, 7 equips, 3
  inventory items. The plan uses this as the canonical fixture rather than an
  invented one.
- Dependency versions: Zod **4**, React **19**, react-router-dom **7**, Vitest
  **4**. `npm run test` is `vitest run`.

## Deviations from `design.md`

Four, each because the code says otherwise. `plan-context.sh` + a full read of
the touched files caught all four at plan time.

1. **§7.1 is wrong that `AppearancePoolSection.test.tsx` assertions "do not
   change."** The test at `templates/__tests__/AppearancePoolSection.test.tsx:47`
   asserts `onPick` was called with `("faceIdx", 1)` — two arguments. The
   design's new prop is single-argument `onPick: (index: number) => void`, so
   that assertion becomes `toHaveBeenCalledWith(1)`, and the `renderSection`
   helper's `template`/`picks` props become `pool`/`selectedIndex`/
   `variantLoadout`. The *behaviour* asserted is unchanged; the argument list is
   not. Task 3 rewrites the helper and states this explicitly.
2. **`looks` is a fixed two-slot draft array, not a variable list.** The design
   describes `looks: MapleLifeLookOptions[]` as "one per configured gender", but
   an operator on a gender with no `looks` row still has to be able to add a
   pool entry. Task 4 mirrors the class treatment: `looks` is always length 2,
   indexed by gender, carrying its own `present` flag; `projectForSave` emits
   only present rows, and `addLookEntry` materialises. This is a strict
   refinement of the design, not a change of behaviour — FR-11.7 still fires for
   a gender with class rows and no look row.
3. **`baseline` is stored as a projection, never as the raw loaded config.**
   The design specifies
   `isDirty = JSON.stringify(projectForSave(state)) !== JSON.stringify(baseline)`.
   That compares serialised key ORDER: the loaded config arrives in the server's
   key order and the projection emits in the draft's declaration order, so an
   untouched load would read as dirty. Task 4 sets
   `baseline = projectForSave(<state just built>)` on `load` and on `savedOk`,
   so both sides pass through the same projection. Consequently the "round-trips
   byte-identically" test is written as deep equality (`toEqual`), and key order
   is deliberately not asserted.
4. **Two line-number citations in the design are off.** The tenant merge is at
   `tenants.service.ts:312`, not `:305` (same function, which starts at `:303`).
   And §3.2's parenthetical undercounts `tenants.service.ts`: its private preset
   entry copies are referenced at `:34,:39` (declarations) **and** `:65,:66`
   (usages) — 4 lines, not 2. Neither changes the scope conclusion; both are
   corrected in the plan.

## Decisions carried forward from the design without change

- **`useTemplatesByRegionAndVersion` cannot serve FR-12.3.**
  `templatesService.getByRegionAndVersion` returns `[sortTemplate(response)]`
  (`templates.service.ts:457-473`) — a one-element array by construction, so a
  multi-match branch is unreachable through it. Task 10 uses `useTemplates()`
  and filters client-side. Confirmed by reading the service.
- **`spSkillId` stays optional** (`json:"spSkillId,omitempty"`), so a class with
  no SP step round-trips as an absent key, not `0`.
- **`AppearancePoolSection` is generalised, not duplicated** (FR-6.3), and the
  templates page's rendered output is unchanged.
- **`BaseStatsSection` is not reused as a component** — it is typed against
  `CharacterPresetAttributes` and its footer copy is wrong for Maple Life. What
  is reused is `useSyncedNumberInput`, the mechanism FR-7.1 actually names.
- **A separate `mapleLifeLoadout.ts`.** `previewLoadout.buildPreviewLoadout`
  composes hair as `baseHair + colorDigit` (`previewLoadout.ts:40`), which
  equals the client's expression only when the base is already a multiple of
  ten. FR-11.1 exists precisely to catch a base that is not, so Maple Life
  normalises explicitly with `hairColor + 10 * Math.floor(hairStyle / 10)`. A
  preview that silently agreed with a bad value would hide the error the page
  exists to surface. Task 6 tests the `(30035, 2) → 30032` case for exactly this
  reason.

## Task sizing

Thirteen tasks, largest is 6 files (Tasks 1 and 9). No task crosses a service
boundary — the whole change is `services/atlas-ui`. `plan-lint.sh` exits 0 with
no findings.

Task 9 (`ProgressionSection`, `SpSkillSection`, `StartingKitSection` plus their
three tests) sits exactly at the 6-file limit and was **deliberately not
split**: the three are sibling presentation components over the same draft,
sharing the `useSyncedNumberInput` idiom and one progression concern, and
splitting them would hand a reviewer three near-identical surfaces to gate
separately for no additional signal. None of the three does any discovery — the
plan spells out every label string, option list, and assertion table.

The one place a boundary was drawn on review surface rather than file count is
Task 11 (`MapleLifeEditor`, 2 files): it is the only unit that owns behaviour
rather than presentation, and its 15 test cases are the acceptance surface for
FR-3, FR-4.5, FR-11.12, FR-12.4 and FR-13. Everything it composes is already
built and tested by Tasks 4–10, so its implementer does discovery on wiring
only.

Tasks 1–3 are prerequisites for everything after them and are independent of one
another; 4–6 are pure modules and can run in parallel with each other once Task
1 lands; 7–10 depend on 3, 4 and 6; 11 depends on all of 1–10; 12 depends on 11;
13 depends on 12.

No AST codemod is warranted. The one repeated mechanical change — the
`CharacterPresetEquipmentEntry` / `CharacterPresetInventoryEntry` rename — is
eight lines across three files (verified by grep, zero test references), well
under the break-even in `docs/codemod-vs-agents.md`.

## Open items carried into implementation

1. **Ordinal→job order for 2/3/4 stays unpinned** (PRD OQ-1, task-246 `design.md`
   §A6). The page surfaces the uncertainty (FR-5.3, FR-11.8) and does not
   resolve it; resolving it is a seed-data change.
2. **Does `mapleLife` survive the existing PATCH?** Task 1 decides this with a
   test rather than an argument. Reading the code says yes — both write paths
   are whole-attributes merges (`tenants.service.ts:312`,
   `templates.service.ts:325-336`), `sortTemplate` (`:61-79`) spreads
   `...template.attributes` and rewrites only `npcs`/`worlds`/`socket`, and
   `validateTemplate` (`:96`) only appends to an error list. If a Task 1
   subtest fails, the fix is in scope (FR-1.3) and a data-repair note goes into
   `progress.md` for the operator — some tenants may already have lost their
   block.
3. **Skin-tone enumeration** is inherited from `AppearanceBrowserDialog`
   unchanged (offers 0–9; seed uses 0–3). No new answer proposed.
4. **`level` 1–200** is a UI input bound, not a clamp. `ClassEntry.Level` is a
   Go `byte`, so anything above 255 is unrepresentable regardless; the backend
   enforces no upper bound. A loaded out-of-range level is displayed as loaded
   (asserted in Task 7).

## Verification

`tools/verify.sh` with **no flags** must exit 0 before the branch is done.
Per-task gates are `npx vitest run <path>` from `services/atlas-ui`, plus
`npm run build` (`tsc -b && vite build`) and `npm run lint` where a task changes
a shared type or a shared component. Before the PR: `frontend-guidelines-reviewer`
(the whole change is TypeScript — `backend-guidelines-reviewer` has nothing to
audit) and `plan-adherence-reviewer`.
