# bug: Maple Life node shown (and crashes) on versions without Maple Life; derivation status must go

PR: atlas-pr-1534. Branch: `task-271-maple-life-config-ui`.

Two independent defects reported against the same feature; both are frontend-only
(`services/atlas-ui`).

## 1. Unsupported versions show the node, and it errors

### Reproduced

Static, not live: the PR-1534 namespace has no running `atlas-configurations`
pod (`kubectl -n atlas-pr-1534 get pods | grep configurations` → no rows), so
the browser repro was not re-run. The chain below is established from source and
seed data, both quoted.

### Observed

Opening `/tenants/<id>/character/maple-life` or
`/templates/<id>/character/maple-life` for a region/version that has no Maple
Life configuration throws instead of rendering. The sidebar offers the node on
every tenant and template regardless of version.

### Expected

The "Maple Life" sidebar entry is absent for a template/tenant whose client has
no Maple Life dialog, and reaching the route directly does not crash.

### Root cause

Two layers:

1. **The response always carries a `mapleLife` object, never `undefined`.**
   `services/atlas-configurations/atlas.com/configurations/tenants/rest.go:23`
   and `.../templates/rest.go:23` both declare
   `MapleLife maplelife.RestModel \`json:"mapleLife"\`` — a non-pointer struct
   with no `omitempty`. Its fields
   (`.../tenants/maplelife/rest.go`: `Looks []LookOptions`, `Classes []ClassEntry`)
   are nil slices for a version with no block, so the payload is
   `"mapleLife":{"looks":null,"classes":null}` — an object that is truthy in JS
   with `null` arrays.

2. **The editor dereferences those arrays unguarded.** In
   `services/atlas-ui/src/components/features/characters/maple-life/mapleLifeEditorState.ts`:
   - `buildDrafts` (line ~150): `config?.classes.find(...)` — the `?.` guards
     `config`, not `config.classes`.
   - `buildLooks` (line ~178): `config?.looks.find(...)` — same.
   - `isEmptyConfig` (line ~313): `config.classes.length`.

   With `classes: null` this is `TypeError: Cannot read properties of null
   (reading 'find')`, thrown from the `load` reducer, which is why the page "just
   errors" rather than showing the empty state.
   `SeedFromTemplateDialog.tsx:68-70` has the same shape
   (`mapleLife?.looks.length`, `mapleLife?.classes.length`).

**Version-support signal (do not invent a version cutoff).** Support is derivable
from data the UI already loads: the socket config. Cross-check of all eleven seed
templates in `services/atlas-configurations/seed-data/templates/` — presence of a
`socket.handlers` entry with `handler: "MapleLifeCheckNameHandle"` (plus the
`MapleLifeResult` / `MapleLifeError` writers) matches presence of the `mapleLife`
block exactly:

| template | MapleLifeCheckNameHandle handler | `mapleLife` block |
|---|---|---|
| gms_12_1, gms_48_1, gms_61_1, gms_72_1, gms_79_1, gms_84_1, jms_185_1 | absent | absent |
| gms_83_1, gms_87_1, gms_92_1, gms_95_1 | present | present |

Note `gms_84_1` has neither — so a `majorVersion >= 83` rule would be **wrong**.
Use the handler-name predicate, not a version number. The op *name* is the
implementation name `MapleLifeCheckNameHandle`
(`libs/atlas-packet/maplelife/serverbound/check_name.go:13`); opcodes differ per
version (0x100 / 0x10E / 0x12D / 0x137) and must not be used.

## 2. Derivation status must be removed

### Observed

Every ordinal tab in the class selector carries a `derived` / `unconfirmed`
badge, and the identity form carries a provenance note ("Derived from the
client's own step-skip…" for ordinals 0-1, a warning box "The ordinal→job order
for 2/3/4 is not derived from the client…" for ordinals 2-4).

### Expected

Both the badge and the note/warning box are gone. Nothing else in the editor
changes — the job field was already fully editable and stays so.

### Root cause

Not a defect; a product decision to drop the provenance surface added by the
original plan (FR-5.1..5.5 / FR-11.8).

## Fix

Worktree: `.worktrees/task-271-maple-life-config-ui`. All paths under
`services/atlas-ui/`.

**Part 1 — support predicate + null safety**

- **new** `src/components/features/characters/maple-life/mapleLifeSupport.ts`
  (or a similarly scoped module): export
  `MAPLE_LIFE_HANDLER = "MapleLifeCheckNameHandle"` and
  `supportsMapleLife(socket: SocketConfig | undefined): boolean` returning true
  iff `socket?.handlers` contains an entry whose `handler` equals that constant.
  Undefined/absent socket → `false`. Comment must state that the predicate is the
  handler *implementation name*, never an opcode, and cite the seed-data
  cross-check above.
- `src/components/features/templates/TemplateDetailLayout.tsx:25` — drop the
  hard-coded Maple Life nav item; include it only when
  `supportsMapleLife(useTemplate(id).data?.attributes.socket)`. The React Query
  cache is shared with `TemplatesMapleLifePage`, so this adds no extra request.
- `src/components/features/tenants/TenantDetailLayout.tsx:24` — same, via
  `useTenantConfiguration(id)` (`src/lib/hooks/api/useTenants.ts:228`).
- `src/pages/TemplatesMapleLifePage.tsx`, `src/pages/TenantsMapleLifePage.tsx` —
  when the query has settled and `supportsMapleLife` is false, render a short
  "this client version has no Maple Life dialog" notice inside the layout instead
  of `<MapleLifeEditor>`, so a bookmarked/deep-linked URL is inert rather than
  offering an editor that can never take effect. Keep the loading/error paths as
  they are.
- `src/components/features/characters/maple-life/mapleLifeEditorState.ts` —
  `buildDrafts`, `buildLooks`, `isEmptyConfig`: treat `null`/absent `classes` and
  `looks` as empty (`config?.classes?.find`, `(config?.classes?.length ?? 0) === 0`,
  etc.). This is the crash fix and must hold independently of the nav change.
- `src/components/features/characters/maple-life/SeedFromTemplateDialog.tsx:68-70`
  — same null-safe treatment for `looks`/`classes`.

**Part 2 — remove derivation status**

- `src/components/features/characters/maple-life/ClassSelector.tsx` — remove
  `badgeText` and the `<Badge variant="secondary">` render, and the now-unused
  `Badge` import. The `not configured` marker stays.
- `src/components/features/characters/maple-life/IdentitySection.tsx` — remove
  the `isDerived` const and the entire trailing `isDerived ? … : …` block
  (both the muted note and the `role="note"` warning box). Field errors and
  layout otherwise unchanged.
- `src/components/features/characters/maple-life/mapleLifeWarnings.ts` — remove
  `WARN.unconfirmedOrdinal` and the `draft.ordinal >= 2` push. `absentRow` and
  `unknownSpSkill` stay. (`MapleLifeEditor` only reads the `spSkillId` warnings,
  so nothing else changes visually.)

**Tests to update/add**

- `__tests__/ClassSelector.test.tsx` — drop badge assertions.
- `__tests__/IdentitySection.test.tsx` — drop the two derivation-note tests.
- `__tests__/mapleLifeWarnings.test.ts` — drop the unconfirmed-ordinal case.
- `__tests__/mapleLifeEditorState.test.ts` — add a case loading
  `{ looks: null, classes: null }` (cast through the JSON shape) and asserting no
  throw, ten absent drafts, and `isEmptyConfig === true`.
- New tests for `supportsMapleLife` and for both detail layouts: node hidden when
  the handler is absent, shown when present.

## Not yet answered

- Not reproduced against a live browser/cluster (PR-1534 has no running
  `atlas-configurations` pod). The fix must be re-tested live once that
  environment is up.
- The unsupported-version notice text is unspecified; any short, factual wording
  is acceptable.

## Resolution

_(to be filled in: fix commit, gate verdict, live re-test)_
