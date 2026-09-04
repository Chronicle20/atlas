# Review: Task 5 — `atlas-data` `ITEM_MAKE` worker registration

Range: `5a3cdb3ed..2df70429f` (one commit: `feat(data): register ITEM_MAKE worker for Etc.wz/ItemMake.img.xml`)

Brief: `.superpowers/sdd/plan/task-5-brief.md`
Report: `.superpowers/sdd/plan/task-5-report.md`

## Diff summary

```
services/atlas-data/atlas.com/data/data/processor.go      | 6 +++++-
services/atlas-data/atlas.com/data/data/processor_test.go | 40 ++++++++++++++
services/atlas-data/docs/kafka.md                          | 2 +-
services/atlas-data/docs/rest.md                           | 50 +++++++++++++++
```

`main.go` is not in the diff — confirmed via `git diff --stat`. Consistent with the binding
ruling that Task 4 already wired `itemmake.InitResource` in `main.go` and Task 5 must not
touch it.

## Requirement-by-requirement

1. **`WorkerItemMake` const, `Workers` slice entry, dispatch branch** — present exactly as
   specified. `services/atlas-data/atlas.com/data/data/processor.go:60` adds
   `WorkerItemMake = "ITEM_MAKE"`; the `Workers` slice (line 65) appends it once, at the end;
   the dispatch branch at `processor.go:216-217` mirrors the `WorkerCharacterCreation` shape
   exactly (`RegisterFileData(path, filepath.Join("Etc.wz", "ItemMake.img.xml"),
   itemmake.NewProcessor(p.l, p.ctx, p.db).RegisterItemMake)()`). PASS.
2. **Import added** — `"atlas-data/itemmake"` added to the import block. PASS.
3. **`TestWorkersIncludesItemMake`** — asserts the const value, exactly-once membership, and
   `len(Workers) == 18`. Verified by reading the diff and re-running the count of the
   `Workers` slice literal by hand (17 pre-existing entries + 1 = 18). PASS.
4. **README / docs update** — see deviation adjudication below.
5. **No `main.go` change** — confirmed. PASS.

## Deviation 1 — README.md path substituted with `docs/kafka.md` / `docs/rest.md`

Verified independently, not taken on the implementer's word:

- `services/atlas-data/atlas.com/data/README.md` does not exist:
  `ls services/atlas-data/atlas.com/data/README.md` → "No such file or directory".
- The service's actual README is `services/atlas-data/README.md`, and it links to both
  substituted docs (`README.md:69-70`: `- [Kafka Integration](docs/kafka.md)`,
  `- [REST API](docs/rest.md)`), and contains no worker enumeration or per-endpoint table of
  its own (grepped for "worker"/"Worker" — only prose references to the ingest worker
  concept, no list to extend).
- `docs/kafka.md` diff (`services/atlas-data/docs/kafka.md:35`) appends `ITEM_MAKE` to the
  existing comma-separated `Worker names:` line, in the same position as the newly-appended
  `WorkerItemMake` (end of list) — consistent with the code change.
- `docs/rest.md` diff adds two new `###` sections, `GET /api/data/item-makes` and
  `GET /api/data/item-makes/{itemId}`, placed alphabetically between the `etcs` and
  `item-strings` sections (confirmed by reading the surrounding diff context — the hunk starts
  right after an `etcs` entry and ends right before `### GET /api/data/item-strings`).
- The JSON example in the new `docs/rest.md` section was checked field-by-field against
  `itemmake/rest.go`'s `RestModel` struct tags (`group`, `reqLevel`, `reqSkillLevel`,
  `itemNum`, `tuc`, `meso`, `catalyst`, `reqItem`, `reqEquip`, `recipe[].{itemId,count}`,
  `randomReward[].{itemId,itemNum,prob}`, `reqQuest`) — all present and correctly named.
  The example's numeric values (item `4260000`, group `0`, recipe `4000000×1`, three
  `randomReward` entries `4260000/4260001/4260002` with `prob` `70/25/5`) were cross-checked
  against the real fixture in `itemmake/resource_test.go` (lines 43-69, 284, 349-351) — an
  exact match, not invented data.
- Route path `/api/data/item-makes` matches `itemmake/resource.go:22`
  (`router.PathPrefix("/data/item-makes")`) composed with the service's `/api/` mount prefix
  (`main.go:73`, `prefix: "/api/"`).

**Verdict on deviation 1: accepted.** The brief's named path is stale/wrong for the current
tree; the substituted docs are the correct home for this content, and the added text is
accurate down to the fixture values.

## Deviation 2 — "unknown worker" error path claim and the resulting test

Verified independently:

- Read the full `StartWorker` `if/else if` chain,
  `services/atlas-data/atlas.com/data/data/processor.go:115-226`. There is no trailing `else`
  branch. If `name` matches none of the `if`/`else if` conditions, `err` (declared `var err
  error` at line 118, zero value `nil`) is never assigned, the `if err != nil` guard at line
  219 is skipped, and the function logs "completed", emits the data-updated event, and returns
  `nil` at line 225 — identically to the success path of every registered worker.
  `grep -rn "unknown worker" data/` returns zero matches. **The implementer's factual claim is
  correct**: no such error exists anywhere in this dispatch chain to assert against.
- Also confirmed: `RegisterFileData` (`processor.go:306-310`) unconditionally discards the
  return value of `rf(...)` (`rf(filepath.Join(rootDir, wzFileName)); return nil`), so even a
  hard failure inside `itemmake.RegisterItemMake` (e.g., a missing `ItemMake.img.xml` under a
  temp dir, which is exactly the test's setup) can never propagate as a non-nil error from
  `StartWorker` for this branch.

**However, the substituted test does not do what the report claims.** The report states the
test "exercis[es] the const, the dispatch branch, and the
`itemmake.NewProcessor(...).RegisterItemMake` call path end-to-end ... without erroring." That
is not correct, and this is a genuine finding:

Because (a) there is no default/`else` branch, and (b) `err` defaults to `nil` and is never
reassigned when no `if`/`else if` condition matches, `StartWorker(WorkerItemMake, tmpDir)`
returns `nil` **whether or not the `WorkerItemMake` dispatch branch exists at all**. Removing
the `else if name == WorkerItemMake { ... }` block entirely (hypothetically, leaving the const
defined so the test still compiles) would produce the exact same observable result: `err ==
nil`. The test in `services/atlas-data/atlas.com/data/data/processor_test.go` (new
`TestStartWorkerDispatchesItemMake`, ~lines 96-109) asserts only `err == nil`, which is true in
both the "branch reached" and "branch missing" worlds. It is a test that passes either way —
it does not prove the `ITEM_MAKE` branch is reached, contrary to the report's characterization
of it as an end-to-end exercise of the dispatch path.

A test that would actually discriminate is achievable within the same scope: e.g., write a
real (minimal) `Etc.wz/ItemMake.img.xml` fixture into the temp dir before calling
`StartWorker`, then query `itemmake.NewStorage(l, db).GetById(...)` (or the registry) afterward
and assert the recipe was actually registered. That would fail if the branch were removed or
mis-wired, and pass only when `RegisterItemMake` actually ran. The current test would not catch
a regression where the `WorkerItemMake` branch is accidentally deleted or the `itemmake` call
is swapped for a no-op, as long as the const/slice entries survive — which is exactly the class
of regression this step existed to guard against per the brief's Step 1 intent ("assert it does
not return the ... error the default branch produces" — i.e., prove dispatch happened).

**Verdict on deviation 2: the underlying factual claim (no "unknown worker" error exists) is
verified correct, but the chosen replacement assertion is vacuous and should not be presented
as proof the dispatch branch is reached. This is a blocking finding on test honesty.**

## Not evaluable

None — the full diff (4 files, 96 lines) was read in whole; both flagged deviations were
independently verified against the referenced source rather than taken on trust.

## Summary

- Functional change (const, slice, dispatch branch, import) is correct and matches the
  brief's exact patch instructions.
- `main.go` correctly left untouched.
- Doc substitution (deviation 1) is justified and accurate.
- `TestWorkersIncludesItemMake` is a real, meaningful test.
- `TestStartWorkerDispatchesItemMake` (deviation 2) is not vacuous by accident of laziness —
  the implementer correctly diagnosed that no distinguishing error exists — but the test as
  written still doesn't prove what the report says it proves, and a discriminating assertion
  was achievable in scope.
