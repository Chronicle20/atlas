# Task 21 fix round: `tsc -b` under `exactOptionalPropertyTypes`

## What I implemented

Widened prop declarations (never narrowed call sites) so `foo?: T` explicitly
admits a passed `undefined`, per the brief's diagnosis.

1. `services/atlas-ui/src/components/features/fields/FieldMonstersTab.tsx`
   `FieldMonstersTabProps.monsters` → `LiveMonsterData[] | undefined`,
   `.error` → `Error | undefined`.
2. `services/atlas-ui/src/components/features/fields/FieldObjectsTab.tsx`
   `FieldObjectsTabProps.defined`, `.definedError`, `.tracked`,
   `.trackedError` all widened with `| undefined`.
3. `services/atlas-ui/src/components/features/fields/FieldHeader.tsx`
   `FieldHeaderProps.worldName` → `string | undefined`.
4. `services/atlas-ui/src/components/features/maps/MapObjectsTable.tsx`
   `MapObjectsTableProps.objects` → `MapObjectData[] | undefined`,
   `.error` → `Error | undefined`.
5. `services/atlas-ui/src/components/features/maps/MapDetailTabs.tsx`
   `MapDetailTabsProps.objects` → `MapObjectData[] | undefined`. See
   "Deviation / judgment call" below for `objectsError`.
6. `services/atlas-ui/src/components/features/maps/__tests__/LiveFieldsSection.test.tsx`
   line 136: unused mock-callback parameter `w` renamed to `_w` (matches the
   file's own `argsIgnorePattern: "^_"` convention used elsewhere in
   `eslint.config.js`, and TS `noUnusedParameters` exempts leading-underscore
   params by default).

`FieldDetailPage.tsx` and the three test files named in the brief
(`FieldMonstersTab.test.tsx`, `FieldObjectsTab.test.tsx`,
`MapObjectsTable.test.tsx`) needed **no edits** — their errors resolved
purely from the declaration widenings above, exactly as the brief predicted.
`git status --porcelain` confirms this: only 6 files changed, matching the
inventory.

## Deviation / judgment call — `MapDetailPage.tsx:91` / `MapDetailTabs.tsx`

The brief pointed me at `MapDetailPage.tsx:91` and told me to match how the
sibling props (`portalsError`, `monstersError`, `reactorsError`) already flow
without erroring, offering two hypotheses: either they work via a `?? undefined`
at the call site, or because the receiving prop is typed `Error | null`.

I read the actual code. Neither hypothesis matched exactly: the real reason
`portalsError`/`monstersError`/`reactorsError` don't error is that
`MapDetailTabsProps` declares them as `portalsError?: unknown` /
`monstersError?: unknown` / `reactorsError?: unknown` (confirmed by reading
`MapDetailTabs.tsx` lines 30-40) — `unknown` accepts the `Error | null` that
`useMapPortals`/`useMapMonsters`/`useMapReactors` actually return (React
Query's `UseQueryResult<T, Error>.error` is `Error | null`, not
`Error | undefined`). Those three props are then only used in truthy checks
(`{portalsError ? ... }`), never forwarded into a component with a stricter
`Error`-typed prop.

`objectsError`, by contrast, IS forwarded downstream into
`MapObjectsTable`'s `error?: Error | undefined` prop (line 230, pre-existing
in the diff before my change). The brief's own file-by-file section
explicitly scopes the `MapObjectsTable` fix to "add `| undefined` to
`objects` and `error`" — it does not say to make `MapObjectsTable.error`
accept `unknown`. So mechanically copying the sibling `unknown` convention
onto `objectsError` would have broken that already-specified, narrower
`MapObjectsTable` contract (an `unknown` cannot flow into an
`Error | undefined` parameter without a cast).

My resolution: widen `MapDetailTabsProps.objectsError` to
`Error | null | undefined` — this is literally the type
`useMapObjects()`'s `error` field has (`UseQueryResult<MapObjectData[], Error>`
→ `error: Error | null`), so `MapDetailPage.tsx`'s
`objectsError={objectsError}` now type-checks without inventing a new
convention on the caller side. Then, at the one place `objectsError` flows
into the strictly `Error`-typed `MapObjectsTable.error` prop
(`MapDetailTabs.tsx:230`), I added `error={objectsError ?? undefined}`.

I want to flag this explicitly because the brief said "if the siblings work
because of a `?? undefined` at the call site, do the same" — I did add a
`?? undefined`, but not at the `MapDetailPage.tsx` call site the brief
pointed at; I added it one layer further in, at `MapDetailTabs.tsx`, where
the value actually needs to become `Error | undefined` to satisfy
`MapObjectsTable`. This is NOT the array-laundering anti-pattern the brief
warned against (`monsters ?? []`): `null` and `undefined` are both already
falsy in `MapObjectsTable`'s `if (error)` check, so there is no
loading/empty-style distinction being destroyed — React Query's `error` is
either "no error" (`null`/`undefined`, indistinguishable in this codebase's
usage) or a concrete `Error`. I could not find a place in this branch where
`null` vs `undefined` on an error field is treated differently. If a
reviewer disagrees with collapsing `null`→`undefined` at that one seam, the
alternative is widening `MapObjectsTable.error` itself to accept `unknown`
or `Error | null | undefined`, but that goes beyond what the brief's file
inventory specified for that file.

## TDD

Not applicable — this is a type-declaration fix round, not new behavior;
the brief's own framing confirms all 2439 tests already pass and must
continue to.

## Verification (both run in the foreground, from the brief's Verification section)

```
cd services/atlas-ui && npx tsc -b
```
Output: none (clean exit). This was the failing check; it is now clean.

```
cd services/atlas-ui && npm test
```
Output (tail):
```
 RUN  v4.1.10 <worktree>/services/atlas-ui

Not implemented: navigation to another Document

 Test Files  291 passed (291)
      Tests  2439 passed (2439)
```
291/291 files, 2439/2439 tests — matches the brief's expectation exactly,
no regression. The "Not implemented: navigation to another Document" line
is jsdom's standard non-fatal stderr noise from a `location.assign`/`href`
test elsewhere in the suite; it is not a failure (run exits with all tests
passed) and pre-exists this change.

## Files changed

- `services/atlas-ui/src/components/features/fields/FieldMonstersTab.tsx`
- `services/atlas-ui/src/components/features/fields/FieldObjectsTab.tsx`
- `services/atlas-ui/src/components/features/fields/FieldHeader.tsx`
- `services/atlas-ui/src/components/features/maps/MapObjectsTable.tsx`
- `services/atlas-ui/src/components/features/maps/MapDetailTabs.tsx`
- `services/atlas-ui/src/components/features/maps/__tests__/LiveFieldsSection.test.tsx`

## Self-review

- Every widening is a declaration-only change (`| undefined`, or in one
  case `| null | undefined`); no call site launders `undefined`/`null` into
  an array (`?? []` does not appear anywhere in this diff).
- `MapObjectsTable.tsx:21`'s `if (objects === undefined) return "Loading
  objects..."` branch is untouched and still reachable — the loading vs.
  empty distinction the brief called out is preserved.
- Confirmed via `git status --porcelain` that only the 6 files above
  changed; no stray edits, no test files edited unnecessarily.
- Ran `git diff --stat` and read every hunk before writing this report.

## Issues or concerns

- The `objectsError ?? undefined` judgment call above (declared prominently,
  not buried) is the only place I diverged from a literal reading of the
  brief's two offered hypotheses for `MapDetailPage.tsx:91`. I believe it is
  correct and minimal, but a reviewer should weigh in given the brief's
  explicit flag that this exact spot required a judgment call.
