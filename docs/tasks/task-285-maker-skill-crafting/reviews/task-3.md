# Review: Task 3 — `atlas-data` `itemmake` archive reader

Range reviewed: `cda7f3f1e..f6d91735e` (2 commits, as instructed):
- `cda7f3f1e` feat(data): add itemmake RestModel and document registry
- `f6d91735e` feat(data): read Etc.wz/ItemMake.img.xml into itemmake models

## Gap-closing context

This artifact was missing at review time; the plan ledger (`.superpowers/sdd/plan/progress.md:143-159`)
carried an APPROVED verdict for Task 3 with no file behind it — the one item the
Tasks 1-14 `plan-adherence-reviewer` shard flagged. This review re-derives the
verdict from the diff and tests directly, independent of the ledger's prose.

## Scope confirmation

`git diff --stat cda7f3f1e~1 f6d91735e` shows five files, all under
`services/atlas-data/atlas.com/data/itemmake/`:

- `registry.go` (+18), `rest.go` (+60), `rest_test.go` (+135) — all from `cda7f3f1e`,
  already reviewed and APPROVED in `docs/tasks/task-285-maker-skill-crafting/reviews/task-2.md`.
  Confirmed via `git diff --stat cda7f3f1e~1 cda7f3f1e -- .../itemmake/` that this is the
  exact same 213-line change already covered there, byte-identical — no re-review needed
  for those three files; not re-litigated here.
- `reader.go` (+94), `reader_test.go` (+320) — `f6d91735e`, this task's actual deliverable.
  This is what the rest of this review evaluates. Scope matches the brief
  (`.superpowers/sdd/plan/task-3-brief.md`) exactly: new `reader.go`/`reader_test.go` only,
  no other file touched. `scope_confirmed`.

## Requirement-by-requirement (`reader.go`)

1. **Signature matches `commodity/reader.go`'s exact shape** —
   `func Read(l logrus.FieldLogger) func(np model.Provider[xml.Node]) model.Provider[[]RestModel]`
   (`reader.go:12-13`), byte-identical pattern to `commodity/reader.go:11-12`, including the
   `np()` unwrap / `model.ErrorProvider` short-circuit on the top-level provider error
   (`reader.go:14-17`). PASS.
2. **Group digit from the top-level node's `Name`** — `reader.go:21`,
   `strconv.Atoi(groupNode.Name)`; non-numeric name is `l.Warnf`'d and `continue`s
   (`reader.go:22-25`), matching the brief's "logged and skipped" rule exactly. PASS.
   `TestReadCoversEveryTopLevelGroup` (`reader_test.go:141-159`) asserts all six
   `(id → group)` pairs, explicitly proving group `4` is not conflated with group `0` — this
   discharges the item Task 2's review left open (`task-2.md`'s "Not evaluable" item (b)).
3. **Entry id from the zero-padded key** — `reader.go:28`, `strconv.Atoi(entryNode.Name)`
   (handles `04260000` → `4260000` correctly, since `Atoi` strips no-op leading zeros for a
   valid decimal string); non-numeric key is `l.Warnf`'d naming the group and bad key, and
   `continue`s (`reader.go:29-32`). PASS. `TestReadSkipsMalformedEntryWithoutAborting`
   (`reader_test.go:309-320`) confirms all six well-formed entries are present and no
   zero-id entry exists despite the trailing `NOT_A_NUMBER` malformed entry.
4. **Every scalar via `GetIntegerWithDefault(name, 0)`** — `reader.go:38-45`, all eight
   fields (`ReqLevel`, `ReqSkillLevel`, `ItemNum`, `Tuc`, `Meso`, `Catalyst`, `ReqItem`,
   `ReqEquip`). PASS, FR-1.5. `TestReadScalars` (`reader_test.go:161-190`) asserts all eight
   on a fully-populated entry; `TestReadAbsentScalarsDefaultToZero`
   (`reader_test.go:193-212`) asserts five of them default to 0 on an entry that omits them,
   and that the entry is still present (not skipped just because scalars are missing) —
   this is the genuine FR-1.5 test the brief calls for, not a restatement.
5. **`recipe` — ordered child-list idiom, no sort** — `reader.go:47-55`. `ChildByName`
   on error leaves the slice initialized-but-empty (`make(..., 0)` at `reader.go:47`, no
   assignment on the `err != nil` branch since only the `err == nil` branch is populated).
   Ranges `recipeNode.ChildNodes` in document order and plain `append`s. PASS.
   `TestReadRecipeOrder` (`reader_test.go:215-235`) asserts three distinct-valued entries by
   index — this would catch an accidental sort or reversal, not just a length match.
   `TestReadRecipeAbsentIsEmpty` (`reader_test.go:296-306`) asserts `len == 0` on the
   group-`4` entry that declares no `recipe` child.
6. **`randomReward` — same shape** — `reader.go:58-66`. PASS.
   `TestReadRandomRewardOrder` (`reader_test.go:238-258`) — three distinct-valued entries by
   index. `TestReadRandomRewardAbsentIsEmpty` (`reader_test.go:261-270`) — `len == 0` on an
   entry with no `randomReward` child.
7. **`reqQuest` — keyed by quest id, not field name** — `reader.go:69-84`. Correctly reads
   `reqQuestNode.IntegerNodes` directly (not `GetIntegerWithDefault`), `strconv.Atoi`s both
   the node's `Name` (quest id) and `Value` (state), log-and-`continue`s on either failing.
   PASS, matches the brief's C-5 structural note precisely.
   `TestReadReqQuest` (`reader_test.go:273-293`) asserts the one `(21614, 3)` pair on
   `1082002` and `len == 0` on the entry that has none.
8. **Malformed entries are skipped, never fatal** — confirmed at both levels (group name
   and entry key); no path in `reader.go` returns a non-nil error except the single
   whole-archive `np()` failure at `reader.go:14-17`, which is a distinct, deliberate case
   (an actually-unparseable document, not a missing field/entry). PASS.
9. **Fixture mechanism** — `reader_test.go:12` declares a raw XML string constant and feeds
   it through `xml.FromByteArrayProvider([]byte(testXML))` into `Read(l)(...)`
   (`reader_test.go:126`), exactly as `commodity/reader_test.go` does; no hand-built
   `xml.Node` struct literals. `model.CollectToMap[RestModel, uint32, RestModel]`
   (`reader_test.go:127`) collects results keyed by `Id`. PASS.
10. **Nine `TestRead*` tests, one per brief acceptance row** — all nine present and named
    exactly as specified: `TestReadCoversEveryTopLevelGroup`, `TestReadScalars`,
    `TestReadAbsentScalarsDefaultToZero`, `TestReadRecipeOrder`, `TestReadRandomRewardOrder`,
    `TestReadRandomRewardAbsentIsEmpty`, `TestReadReqQuest`, `TestReadRecipeAbsentIsEmpty`,
    `TestReadSkipsMalformedEntryWithoutAborting`. PASS.

## Test execution (independently re-run, not taken from the implementer report)

```
cd services/atlas-data/atlas.com/data && go build ./... && go test ./itemmake/... -count=1 -v
```

All 9 new `TestRead*` plus Task 2's 5, and Task 4's already-landed tests in the same
package (this worktree is at a later HEAD than `f6d91735e`), pass. `gofmt -l ./itemmake`
and `go vet ./itemmake/...` both clean (no output).

## Test honesty

Spot-checked two tests for whether they would actually fail against a plausible wrong
implementation, not just pass either way:
- `TestReadRecipeOrder`/`TestReadRandomRewardOrder` use distinct values at each index
  (`4011001`/`4011002`/`4021007`; `70`/`25`/`5`), so a sort-by-item-id or reversed-append
  bug would flip the assertion. Not a length-only check.
- `TestReadCoversEveryTopLevelGroup` uses six *different* group digits with the same
  entry-loop code path; a bug that derived `Group` from the entry id's leading digit instead
  of the true top-level node name would still pass for `4260000`→group `0` and
  `1082002`→group `1` (leading digit matches) but the fixture doesn't actually distinguish
  "true top-level name" from "id's leading digit" in every case — checked by hand: id
  `2020000`'s leading digit is `2` and its group is `2` (also matches), `4030000`→`4`
  (matches), `8000000`→`8` (matches), `16000000`'s leading two digits are `16`, group is
  `16` (still matches). So this specific test would **not** distinguish a
  derive-group-from-id-digit regression from the correct implementation, since every
  fixture id's leading digit(s) happen to equal its true group. This is a latent test-fixture
  weakness, not a defect in `reader.go` itself — the code (`reader.go:21,35`) unambiguously
  reads `Group` from `groupNode.Name`, not from `id`. Noted below as non-blocking.

## R-2 (missing field → zero; missing archive → empty set, not startup failure)

Traced by hand, not assumed:
- **Missing field → zero**: `GetIntegerWithDefault(name, 0)` for every scalar
  (`reader.go:38-45`); confirmed by `TestReadAbsentScalarsDefaultToZero`. Satisfied and
  tested within this task's own scope.
- **Missing archive → empty result, not a startup failure**: `reader.go:14-17` returns
  `model.ErrorProvider[[]RestModel](err)` when the top-level `np()` provider itself errors
  (e.g. `xml.FromPathProvider` on a missing file returns `model.ErrorProvider[Node](err)` at
  `services/atlas-data/atlas.com/data/xml/reader.go:29-31`). `reader.go` does **not** turn a
  missing-file error into an empty slice itself — it propagates the error, correctly and by
  design (a genuinely unparseable/absent document is a different case from a missing field).
  The "does not fail startup" half of R-2 is discharged one layer up, by
  `RegisterFileData` (`services/atlas-data/atlas.com/data/data/processor.go:302-307`)
  discarding the `RegisterFunc` return value entirely — pre-existing, unchanged
  infrastructure, and explicitly called out as such in the brief itself. This is Task 4/5's
  wiring (the `RegisterFunc` that calls `Read` and populates the registry), not Task 3's.
  No `reader_test.go` test simulates a missing-file provider, and none is needed for Task 3
  to satisfy its own contract — `reader.go` never claims to swallow a whole-archive read
  failure; it is written to propagate it, which is correct given who's responsible for
  discarding it. Recorded under **Not evaluable** below since confirming the end-to-end
  "empty set, no crash" behavior requires Task 4/5's registration code, outside this diff.

## Consistency with implementer report

The report's claims (signature, group/entry-level skip behavior, `reqQuest`'s `IntegerNodes`
idiom, order preservation, FR-1.5 defaulting, 14/14 tests passing) all match the diff and the
independently re-run test output. No drift.

## Findings

No blocking findings.

Non-blocking:
- `reader.go:21-25` — the group-level non-numeric-name skip branch (a top-level `imgdir`
  whose `Name` doesn't parse as an integer) has no dedicated test. Only the entry-level skip
  is exercised (`TestReadSkipsMalformedEntryWithoutAborting`, via the `NOT_A_NUMBER` *entry*
  under group `"1"` — the group name there is still numeric). The branch is structurally
  identical to the tested entry-level path (`strconv.Atoi` + `Warnf` + `continue`), so this is
  a coverage gap, not a suspected correctness defect. This matches the ledger's own carried
  note (`.superpowers/sdd/plan/progress.md:155-159`) verbatim in substance.
- `TestReadCoversEveryTopLevelGroup`'s fixture happens to make every entry id's leading
  digit(s) equal its true `Group`, so the test cannot distinguish "Group read from the
  top-level node name" (what `reader.go` actually does) from a hypothetical "Group derived
  from the id's leading digit" regression. `reader.go` itself is correct (`Group` comes from
  `groupNode.Name` at `reader.go:21,35`, never from `id`), so this is a fixture-design
  observation, not a present defect.

## Not evaluable

- The end-to-end "missing archive ingests as an empty recipe set, doesn't fail startup" half
  of R-2 depends on `RegisterFileData`'s error-discarding wrapper and the `RegisterFunc` that
  calls `itemmake.Read` and populates `GetModelRegistry()` — both are Task 4/5's scope
  (`processor.go`/worker registration), not present in this diff. `reader.go`'s own
  contribution (propagate a genuine whole-archive parse error rather than swallowing it
  itself) is correct and traced above, but the full chain cannot be verified from this diff
  alone.

## Verdict rationale

All nine brief-mandated tests are present, independently re-run, and pass; each was checked
for whether it would fail against a plausible wrong implementation (one gap found and noted
above, non-blocking). The reader's structure matches `commodity/reader.go`'s pattern and the
brief's five-step pseudocode line for line. `Group`/`ReqQuest` — the two items Task 2's review
explicitly deferred here — are both confirmed correctly implemented and tested. No scope
creep; `rest.go`/`registry.go` untouched by `f6d91735e`. The prior ledger's APPROVED verdict
is confirmed, not overturned, with one coverage-gap non-blocking finding recorded (matching
what the ledger prose already claimed, now backed by file:line evidence).
