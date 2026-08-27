# Report: bug-unsupported-version-node-and-derivation-status

## What I implemented

**Part 1 — support predicate + null safety**

- New `services/atlas-ui/src/components/features/characters/maple-life/mapleLifeSupport.ts`:
  exports `MAPLE_LIFE_HANDLER = "MapleLifeCheckNameHandle"` and
  `supportsMapleLife(socket: SocketConfig | undefined): boolean`, with a
  comment stating the predicate is the handler implementation name (never an
  opcode) and citing the seed-data cross-check from the brief.
- `TemplateDetailLayout.tsx` — calls `useTemplate(String(id ?? ""))` (shares
  the React Query cache with `TemplatesMapleLifePage`) and includes the
  "Maple Life" nav item only when `supportsMapleLife(templateQuery.data?.attributes.socket)`.
- `TenantDetailLayout.tsx` — same via `useTenantConfiguration(id ?? "")`.
- `TemplatesMapleLifePage.tsx` / `TenantsMapleLifePage.tsx` — once the query
  has settled (not loading, no error) and the resource has loaded but
  `supportsMapleLife` is false, render "This client version has no Maple Life
  dialog." inside the layout instead of `<MapleLifeEditor>`. Loading/error
  paths pass through to the editor unchanged (the editor already owns those
  states via the adapter).
- `mapleLifeEditorState.ts` — `buildDrafts`/`buildLooks` now use
  `config?.classes?.find` / `config?.looks?.find`; `isEmptyConfig` uses
  `(config.classes?.length ?? 0) === 0`. This is the crash fix and does not
  depend on the nav-item change.
- `SeedFromTemplateDialog.tsx` — `mapleLife?.looks?.length ?? 0` /
  `mapleLife?.classes?.length ?? 0`.

**Part 2 — remove derivation status**

- `ClassSelector.tsx` — removed the `Badge` import, `badgeText`, and the
  `<Badge variant="secondary">` render. The "not configured" marker is
  unchanged.
- `IdentitySection.tsx` — removed `isDerived` and the entire trailing
  derived-note / unconfirmed-warning block; updated the component doc
  comment to drop the FR-5.1..5.5/FR-11.8 provenance description. Field
  layout, errors, and the job field's full editability are otherwise
  unchanged.
- `mapleLifeWarnings.ts` — removed `WARN.unconfirmedOrdinal` and the
  `draft.ordinal >= 2` push. `absentRow` and `unknownSpSkill` are unchanged.

## Tests updated/added

- `__tests__/ClassSelector.test.tsx` — dropped the two badge-assertion tests
  ("badges ordinals 0 and 1 as derived", "badges ordinals 2, 3 and 4 as
  unconfirmed").
- `__tests__/IdentitySection.test.tsx` — dropped the two derivation-note
  tests, replaced with one test confirming the job field stays editable for
  ordinal 2 and that no `role="note"` element renders.
- `__tests__/mapleLifeWarnings.test.ts` — dropped the two unconfirmed-ordinal
  cases; fixed the "stops warning about a row once it is materialised" test,
  which previously asserted `WARN.unconfirmedOrdinal`, to assert the message
  list is now empty for that row.
- `__tests__/mapleLifeEditorState.test.ts` — added "load tolerates null
  looks/classes (the shape the API sends for an unconfigured version)": casts
  `{ looks: null, classes: null }` through `as unknown as MapleLifeConfig`
  (documented as the wire shape, since the TS type says non-null arrays),
  asserts `load` does not throw, all ten drafts are absent, and
  `isEmptyConfig` is `true`.
- New `__tests__/mapleLifeSupport.test.ts` — absent socket → false; handler
  absent → false (mirrors gms_84_1's shape from the brief's table); handler
  present at each of the four opcodes cited in the brief → true (proves the
  predicate is opcode-independent); empty handlers list → false.
- `templates/__tests__/TemplateDetailLayout.test.tsx` — updated the existing
  test to mock `useTemplate` (now a dependency of the layout) and added two
  new cases: nav item hidden when the handler is absent, shown when present.
- `tenants/__tests__/TenantDetailLayout.test.tsx` — same pattern via
  `useTenantConfiguration`, plus updated the existing Export-control test to
  mock the hook.

## What I did not add

No new page-level tests for `TemplatesMapleLifePage`/`TenantsMapleLifePage`
(none existed before, and the brief's test list did not request them).
Those pages remain covered indirectly through the layout tests and through
`mapleLifeEditorState`/`mapleLifeSupport` unit coverage; the notice branch
itself is exercised by hand-tracing (`!isLoading && !error && data &&
!supportsMapleLife(...)`), not by a rendered test.

## Verification (module-local, from services/atlas-ui)

`npm run lint` — 0 errors, 9 pre-existing warnings unrelated to this change
(react-hooks/incompatible-library in coupons/reward-pools/tenants dialogs,
react-hooks/exhaustive-deps in AccountsPage/QuestsPage). No new warnings
introduced.

`npx tsc -b` — clean, no output.

`npx vitest run` limited to touched test files plus the rest of the
maple-life test directory for a wider net:

```
npx vitest run \
  src/components/features/characters/maple-life/__tests__/ClassSelector.test.tsx \
  src/components/features/characters/maple-life/__tests__/IdentitySection.test.tsx \
  src/components/features/characters/maple-life/__tests__/mapleLifeWarnings.test.ts \
  src/components/features/characters/maple-life/__tests__/mapleLifeEditorState.test.ts \
  src/components/features/characters/maple-life/__tests__/mapleLifeSupport.test.ts \
  src/components/features/characters/maple-life/__tests__/SeedFromTemplateDialog.test.tsx \
  src/components/features/templates/__tests__/TemplateDetailLayout.test.tsx \
  src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx
```
Result: `Test Files 8 passed (8)`, `Tests 74 passed (74)`.

Then, for a wider net over the whole maple-life feature directory:
```
npx vitest run src/components/features/characters/maple-life
```
Result: `Test Files 13 passed (13)`, `Tests 142 passed (142)`. Output pristine
(no console warnings/errors surfaced by these runs).

## Files changed

- `services/atlas-ui/src/components/features/characters/maple-life/mapleLifeSupport.ts` (new)
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/mapleLifeSupport.test.ts` (new)
- `services/atlas-ui/src/components/features/characters/maple-life/mapleLifeEditorState.ts`
- `services/atlas-ui/src/components/features/characters/maple-life/SeedFromTemplateDialog.tsx`
- `services/atlas-ui/src/components/features/characters/maple-life/ClassSelector.tsx`
- `services/atlas-ui/src/components/features/characters/maple-life/IdentitySection.tsx`
- `services/atlas-ui/src/components/features/characters/maple-life/mapleLifeWarnings.ts`
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/ClassSelector.test.tsx`
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/IdentitySection.test.tsx`
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/mapleLifeEditorState.test.ts`
- `services/atlas-ui/src/components/features/characters/maple-life/__tests__/mapleLifeWarnings.test.ts`
- `services/atlas-ui/src/components/features/templates/TemplateDetailLayout.tsx`
- `services/atlas-ui/src/components/features/templates/__tests__/TemplateDetailLayout.test.tsx`
- `services/atlas-ui/src/components/features/tenants/TenantDetailLayout.tsx`
- `services/atlas-ui/src/components/features/tenants/__tests__/TenantDetailLayout.test.tsx`
- `services/atlas-ui/src/pages/TemplatesMapleLifePage.tsx`
- `services/atlas-ui/src/pages/TenantsMapleLifePage.tsx`
- `docs/tasks/task-271-maple-life-config-ui/bug-unsupported-version-node-and-derivation-status.md` (added to the commit — the bug brief itself; was untracked in the worktree before this task)

Commit: `dc8b47630` — "fix(atlas-ui): hide Maple Life on unsupported
versions, drop derivation status"

## Self-review

- **Completeness**: both parts of the brief's `## Fix` section are done file
  by file, matching every bullet under "Part 1" and "Part 2". Both detail
  layouts, both maple-life pages, `mapleLifeEditorState.ts`,
  `SeedFromTemplateDialog.tsx`, `ClassSelector.tsx`, `IdentitySection.tsx`,
  and `mapleLifeWarnings.ts` are all touched exactly as specified. All five
  "Tests to update/add" bullets are covered.
- **Quality**: `supportsMapleLife` doc comment explicitly warns against an
  opcode-based check and points at the seed-data cross-check, matching the
  brief's caution about `gms_84_1`. Layout components reuse the React Query
  hook the corresponding page already calls, so no extra network request is
  introduced (verified by reading `useTemplate`/`useTenantConfiguration`
  query keys — both layouts and pages now share `templateKeys.basicDetail`/
  `tenantKeys.configDetail`).
- **Discipline**: did not invent a version-number cutoff; used the handler
  name exactly as specified (`MapleLifeCheckNameHandle`). Did not touch any
  Go file — `mapleLife` staying non-pointer in the REST payload is
  intentional per the ruling. Did not add speculative tests beyond the
  brief's list, aside from the layout- and predicate-level tests explicitly
  requested.
- **Testing**: `mapleLifeEditorState.test.ts`'s new case casts through
  `as unknown as MapleLifeConfig`, as instructed, since the TS model type
  doesn't reflect the wire's nullable-array reality — this is the same
  pattern the brief itself uses to describe the defect. Reused the existing
  `useTemplate`/`useReseedTemplate`-style `vi.mock` pattern already present
  in the codebase (`TemplateReseedButton.test.tsx`) for the two layout test
  files, rather than inventing a new mocking convention.

## Issues / concerns

- Not re-tested against a live browser/cluster — the brief notes PR-1534 has
  no running `atlas-configurations` pod, so this remains unverified live per
  the brief's own "Not yet answered" section. The fix is grounded in
  reading the actual REST model/reducer source, not guessed.
- The unsupported-version notice text ("This client version has no Maple
  Life dialog.") was left as short, factual wording per the brief's explicit
  allowance ("any short, factual wording is acceptable").
