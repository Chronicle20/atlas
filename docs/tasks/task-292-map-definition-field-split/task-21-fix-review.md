# Review: task-21 fix round (`9f6679907`)

Range reviewed: `a47f63a2a..9f6679907` (single commit, 6 files, +16/-13).
Brief: `.superpowers/sdd/plan/task-21-fix-brief.md`.
Implementer report: `docs/tasks/task-292-map-definition-field-split/task-21-fix-report.md`.

## Scope confirmation

`git diff --stat a47f63a2a..9f6679907` and `git show 9f6679907 --stat` both show exactly the 6
files named in the brief's "Files" section, +16/-13. No other file is touched. Matches scope.

## The main question: `MapDetailPage.tsx:91` / `MapDetailTabs.tsx` deviation

Verified each link in the implementer's chain against source, not the report's summary:

1. **Are the sibling props really typed `unknown`?** Confirmed —
   `MapDetailTabs.tsx:33-38`: `portalsError?: unknown; monstersError?: unknown; reactorsError?:
   unknown;`. Neither of the brief's two offered hypotheses (`?? undefined` at the call site, or
   `Error | null` on the receiving prop) is what actually makes those three siblings compile. The
   implementer's report is accurate on this point, and it is a legitimate reason the brief's two
   hypotheses could not be followed literally.

2. **Is `Error | null | undefined` really `useMapObjects`'s error type?** `useMapEntities.ts:70-79`
   declares `useMapObjects(mapId: string): UseQueryResult<MapObjectData[], Error>`. Checked
   `@tanstack/query-core`'s `UseQueryResult` shape directly
   (`node_modules/@tanstack/query-core/build/modern/hydration-Bjs0MSgg.d.ts`): the query-result
   `error` field is typed `TError | null` — never `| undefined`. So the true type flowing out of
   `useMapObjects()` is `Error | null`, not `Error | null | undefined`. The `| undefined` the
   implementer added to `MapDetailTabsProps.objectsError` is not literally required by
   `useMapObjects`'s return type — it is there only because the prop is itself optional
   (`objectsError?:`) and `exactOptionalPropertyTypes` needs the explicit admission for the
   pattern to be uniform with the rest of the file. Minor imprecision in the report's phrasing
   ("this is literally the type `useMapObjects()`'s `error` field has") but not a functional
   defect — the resulting type is a safe superset, not an unsound narrowing.

3. **Is the null→undefined collapse-at-one-seam argument sound?** Checked
   `MapObjectsTable.tsx:17`: `if (error) { return <p>Failed to load objects.</p>; }` — a bare
   truthy check. `null` and `undefined` are both falsy and take the identical branch. Grepped the
   full touched surface (`grep -rn "objectsError\|error === null\|error === undefined" src/components/features/maps
   src/pages/MapDetailPage.tsx`) — no code anywhere in this branch inspects `null` vs `undefined`
   on an error field distinctly; every consumption is a truthy/falsy check. The brief's
   "never launder at the call site" prohibition was scoped explicitly to the loading-vs-empty
   distinction on *array* props (`MapObjectsTable.tsx:21`'s `objects === undefined` branch, which
   this commit does not touch and remains reachable — confirmed by reading the file). No such
   distinction exists for error props. The implementer's argument is correct and the `?? undefined`
   on an `Error | null` value is not the array-laundering anti-pattern the brief warned against.

4. **Is one-layer-down (`MapDetailTabs.tsx:232`) the right seam vs. `MapDetailPage.tsx:91`?**
   Both are defensible; this is a genuine judgment call, not a correctness bug. One point worth
   surfacing for the controller's awareness (non-blocking): `MapDetailTabs` has exactly one call
   site (`MapDetailPage.tsx:82`, confirmed via `grep -rn "MapDetailTabs" src`), so there is no
   reuse pressure requiring `MapDetailTabsProps.objectsError` to carry the hook's full `Error |
   null` shape. Doing the `?? undefined` collapse at `MapDetailPage.tsx:91` instead (matching the
   `?? undefined`-at-call-site pattern the brief explicitly offered) would have let
   `MapDetailTabsProps.objectsError` stay `Error | undefined` — uniform with every other prop
   widened in this commit, and arguably the "smaller surface" the review guidance asks to weigh.
   The implementer's report acknowledges this alternative explicitly and explains the choice
   (typing the prop after the hook's actual return type). Both are internally consistent and
   type-safe; I do not treat this as blocking. It is a legitimate, declared judgment call
   correctly flagged for reviewer attention, and the reasoning behind it holds up against source.

**Verdict on the main question: the deviation is sound.** All three factual claims underpinning
it check out against source (with the one minor imprecision noted in point 2, which does not
change the conclusion). Non-blocking stylistic note only.

## Routine checks

5. **Array props widened by declaration only, never `?? []` at call sites.** `git show 9f6679907`
   confirms every array-prop change is `T[]?` → `T[] | undefined` in an interface; grepped the
   full diff for `?? []` — none present. `MapObjectsTable.tsx:21`'s `if (objects === undefined)
   return "Loading objects..."` line is untouched by this commit and still reachable. PASS.

6. **`LiveFieldsSection.test.tsx:136` fix preserves the assertion.** Read
   `LiveFieldsSection.test.tsx:128-146`: only the mock-callback parameter changed
   (`w` → `_w`); the test body (`useLiveMonstersMock.mockImplementation(...)`, the
   `expect(cells[3]).toHaveTextContent("1")` / `expect(cells[4]).toHaveTextContent("—")`
   assertions) is byte-identical to before. PASS.

7. **`tsc -b` and `npm test`, run in the foreground myself:**
   - `cd services/atlas-ui && npx tsc -b` → exit 0, no output. Clean, matches report. PASS.
   - `cd services/atlas-ui && npm test` → `Test Files 291 passed (291)`, `Tests 2439 passed
     (2439)`. Matches report exactly, no regression. PASS.

8. **Commit hygiene.** `git show 9f6679907 --stat` lists exactly the 6 intended files. The
   untracked `docs/tasks/task-292-map-definition-field-split/{agent-ledger.tsv,
   task-21-fix-report.md, task-21-report.md, task-21-review.md}` shown by `git status --porcelain`
   are untracked working-tree files, not part of this commit — confirmed by diffing
   `git show 9f6679907 --stat` (6 files) against `git status --porcelain` (the report/ledger files
   are `??`, never staged into `9f6679907`). PASS.

## Open question noted for the record (explicitly out of scope, not blocking)

Per the task instructions: `FieldMonstersTab.tsx:46` (`if (!monsters || monsters.length === 0)`)
and the equivalent empty-check in `FieldObjectsTab.tsx` still collapse "loading" and "genuinely
empty" into the same false "No monsters/objects" empty-state copy on first paint. This commit did
not touch either function body — only the prop *declarations* changed
(`monsters?: LiveMonsterData[]` → `monsters?: LiveMonsterData[] | undefined`); the runtime
`if (!monsters ...)` logic is byte-identical before and after this commit (confirmed by reading
both files in full). The type being now explicitly `| undefined` did not make a loading-branch
fix "trivial" in this commit — no such branch was added or touched. This remains exactly as
open as it was before `9f6679907`, correctly out of this commit's scope, and not a finding here.

## Findings

None blocking. One non-blocking note (item 4 above) on seam placement, offered as the smaller
alternative surface for future reference, not a defect.

## Not evaluable

None — the full 6-file diff, the two hook/interface files load-bearing to the deviation
(`useMapEntities.ts`, `MapDetailTabs.tsx`), and both verification commands were checked directly
against source and run in the foreground.
