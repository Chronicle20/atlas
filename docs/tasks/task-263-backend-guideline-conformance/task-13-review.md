# Task 13 review — W3 hand work: the four packages named in #1498 (FR-7)

**Range reviewed:** `d0ba529db..HEAD`, specifically commits `22bb8c1d0`, `39c6b08e9`,
`19e500dcf` (code) and `0ae9f940c` (report).

**Verdict: APPROVED**

## Scope confirmation

`git diff --name-only d0ba529db..HEAD -- '*.go'` shows exactly:

```
services/atlas-channel/atlas.com/channel/data/tradeability/rest.go
services/atlas-channel/atlas.com/channel/data/tradeability/rest_test.go
services/atlas-channel/atlas.com/channel/monsterbook/rest.go
services/atlas-channel/atlas.com/channel/monsterbook/rest_test.go
services/atlas-inventory/atlas.com/inventory/data/tradeability/rest.go
services/atlas-inventory/atlas.com/inventory/data/tradeability/rest_test.go
services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest_test.go
```

This matches the brief's `### Files` list exactly (the eighth listed file,
`.../consumable/rest.go`, was intentionally not touched because its `Transform`
already existed — see point 4 below). No package outside this list gained a
`Transform` function. The docs-only commit `0ae9f94` touches only
`docs/tasks/task-263-backend-guideline-conformance/task-13-report.md`.

## Findings

### 1. Each `Transform*` is the exact field-by-field inverse of `Extract*`

- **`atlas-channel` / `data/tradeability`** (`rest.go:145-175`): `transform[R any, PR interface{*R; setFields(bool,int32,bool)}]` mirrors the pre-existing `extract[R interface{fields()(bool,int32,bool)}]` (`rest.go:140-143`) field-for-field: `tradeBlock`, `tradeAvailable`, `only`. Each of the five wire types' `setFields` (added at `rest.go:51-53`, `72-74`, `90-92`, `108-110`, `126-128`) sets exactly the three fields `fields()` reads. `Id` is untouched in both directions — correct, since `extract` never reads `Id` either. Confirmed by reading the full file.
- **`atlas-inventory` / `data/tradeability`** (`rest.go:118-154`): same shape, correctly reduced to the two-field variant (`tradeBlock`, `tradeAvailable`) because this package's `Model` has no `only` field (`rest.go:17-20`). This is a legitimate divergence from the brief's literal 3-arg `NewModel` example — the brief's own prose says "Read the actual field lists... do not invent field names," and the implementer's report documents the divergence explicitly.
- **`atlas-channel` / `monsterbook`**: `TransformCard` (`rest.go:107-109`) is the exact inverse of `ExtractCard` (`rest.go:102-104`) — `cardId`/`CardId`, `level`/`Level`, `isSpecial`/`IsSpecial`, both directions, nothing extra. `Transform` (`rest.go:125-135`) is the exact inverse of `Extract` (`rest.go:112-122`) — all seven `Collection` fields (`bookLevel`, `normalCount`, `specialCount`, `totalUniqueCards`, `coverCardId`, `coverMonsterId`, `expBonusPercent`) mapped both directions, `CollectionRestModel.Id` untouched by either (consistent with `Extract` never reading it).
- **`atlas-monster-book` / `data/consumable`**: `Transform` (`rest.go:45-50`, unchanged in this range) maps only `monsterBook`/`monsterId`, matching `Extract` (`rest.go:41-43`); `RestModel.Id` unmapped by both, exactly as the brief specifies.

No dropped or invented field found in any of the four packages.

### 2. `data/tradeability` mirrors the generic `extract[R]` with a generic `transform[R]` (FR-3), in both packages

Confirmed in both `atlas-channel/.../tradeability/rest.go:148-155` and
`atlas-inventory/.../tradeability/rest.go:127-134`. Both use the
`PR interface{*R; setFields(...)}` type-parameter pattern to build a zero-value
`R` generically, and both packages' five named `Transform*` functions
(`TransformEquipment`, `TransformConsumable`, `TransformSetup`, `TransformEtc`,
`TransformCash`) are one-liners that call `transform[X](m)`. This is the
correct generic mirror of the existing `extract[R]` shape and satisfies FR-3's
stated preference.

### 3. The `collection` subtest deviation from the brief's prose is correct

`services/atlas-channel/atlas.com/channel/monsterbook/processor.go:21-29`
confirms `Collection` has no `[]Card` field — it is a flat struct of seven
scalar fields (`bookLevel`, `normalCount`, `specialCount`, `totalUniqueCards`,
`coverCardId`, `coverMonsterId`, `expBonusPercent`). The `[]Card` slice belongs
to a different type (`Model`, per the report) that this task's `Transform`/
`Extract` pair does not touch. The implementer's report documents this
discrepancy explicitly and correctly built the subtest against `Collection`'s
real fields instead of inventing a `[]Card` field.

The round-trip assertion in `monsterbook/rest_test.go:318-340` is meaningful:
every field of the test `Collection` literal is distinct and non-zero
(`bookLevel:3, normalCount:5, specialCount:2, totalUniqueCards:7,
coverCardId:2380000, coverMonsterId:100100, expBonusPercent:9`), so a
cross-field swap in `Transform`/`Extract` would be caught by
`reflect.DeepEqual`. The `card` subtest (`rest_test.go:302-316`) likewise uses
distinct non-zero values (`cardId:2380005, level:5, isSpecial:true`).

### 4. No duplicate/regenerated `Transform` landed in `monster-book/data/consumable`

`git show 19e500dcf --stat` shows the only change in that commit is a 1-line
diff to `rest_test.go` (`monsterId: 22` → `monsterId: 100100`, matching the
brief's exact table value). `rest.go` (containing `Transform` at lines 45-50)
is unchanged in this range — confirmed via `git diff --name-only`, which does
not list it. The implementer's claim that this `Transform` predates this task
(Task 12) and was only verified, not regenerated, is correct.

### 5. No stray `Transform` outside the brief's file list; no absolute paths in the report

Confirmed above (scope confirmation) that the changed `.go` files match the
brief's list exactly. For the docs-only commit `0ae9f94`, the sole file
touched is `task-13-report.md`; `grep -nE "/home/[a-zA-Z]+|/Users/[a-zA-Z]+"`
against it returns no matches.

## Build/test verification (module-local, per Step 5)

Ran from each module root:

```
services/atlas-channel/atlas.com/channel:      go build/vet clean; go test ./data/tradeability/... ./monsterbook/...  -> ok
services/atlas-inventory/atlas.com/inventory:  go build/vet clean; go test ./data/tradeability/...                    -> ok
services/atlas-monster-book/atlas.com/monster-book: go build/vet clean; go test ./data/consumable/...                 -> ok
```

All PASS, no `FAIL` lines.

## Not evaluable

None. The full review surface (four packages, three commits' worth of new
code, one report commit) was covered within scope.

## Non-blocking notes

- None.

## Process note (reviewer, not a finding against the unit under review)

While verifying test-honesty by attempting a scratch revert, an errant
`git stash pop` in this worktree briefly surfaced an unrelated, pre-existing
stash entry (`stash@{0}`, dated to an earlier session, unconnected to this
task) as a merge conflict on `agent-ledger.tsv`. This was immediately resolved
with `git checkout HEAD -- <path>`; the working tree is confirmed clean, HEAD
is unchanged at `0ae9f940c`, and the stash list is unchanged (7 entries,
`stash@{0}` still present, untouched). No commit, amend, or rebase occurred.
Recorded here for transparency; it is not a defect in the reviewed unit.
