# Review: Task 12 batch 3 of 3 — W3 apply `transform` to tier-A services

Commit range: `2a84f41f9..d0ba529db`
Brief: `.superpowers/sdd/plan/task-12-batch-3-brief.md`
Report: `docs/tasks/task-263-backend-guideline-conformance/task-12-batch-3-report.md`

## Scope

Reviewed all 9 commits in range: 8 per-service `feat(<S>): add Transform and round-trip
tests` commits plus the closing `chore(task-263): record Task 12 batch 3 ledger and
report` commit. Verified `git diff --stat 2a84f41f9..d0ba529db` touches only
`rest.go`/`rest_test.go` pairs in the 9 named services plus the new ledger and report
files — no hand-edited files, no non-tier-A package touched, no generator source
touched. Scope matches the brief. `scope_confirmed`: full diff read, all 9 per-service
commits inspected individually, one SKIPPED package's skip characterized from source,
build/vet/test independently re-run for 2 of the 8 applied services.

## Findings

### 1. Ledger integrity — PASS

`docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest-3.tsv` as
committed at `d0ba529db` contains exactly 9 rows, one per service in the brief's list,
no duplicates, no drops:

```
services/atlas-rankings/atlas.com/rankings/character	APPLIED
services/atlas-party-quests/atlas.com/party-quests/guild	APPLIED
services/atlas-npc-conversations/atlas.com/npc/petdata	SKIPPED	...
services/atlas-monster-book/atlas.com/monster-book/data/consumable	APPLIED
services/atlas-marriages/atlas.com/marriages/character	APPLIED
services/atlas-guilds/atlas.com/guilds/character	APPLIED
services/atlas-fame/atlas.com/fame/character	APPLIED
services/atlas-doors/atlas.com/doors/data/skill/effect	APPLIED
services/atlas-buddies/atlas.com/buddies/character	APPLIED
```

Cross-checked against `awk -F'\t' '$2=="A"'
docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv` filtered to these
9 services: exactly 9 tier-A rows, all "flat literal", matching 1:1 with the ledger
rows. No tier-A package for these services was missed or double-counted.

(Working tree currently shows a 1-line diff on this ledger file changing an absolute
path to a relative path — `git diff d0ba529db -- .../ledger-transform-rest-3.tsv`. That
is a later, uncommitted controller edit outside this batch's commit range, per the task
note; not attributed to this batch.)

### 2. Per-service commit scoping — PASS

Every one of the 8 `feat(<S>): ...` commits touches exactly 2 files (`rest.go` +
`rest_test.go`) under that service's own path, confirmed via `git show <sha> --stat` for
all 8 (e.g. `449d2c093` → only `services/atlas-rankings/atlas.com/rankings/character/{rest.go,rest_test.go}`;
`0f4dc6e17` → only the `atlas-buddies` pair). No cross-service leakage.

### 3. Generated `Transform` shape — PASS

Spot-checked `atlas-rankings` (`449d2c093`) and `atlas-guilds` (`5d28ddf1b`) diffs:
`Transform(m Model) (RestModel, error)` field-for-field mirrors of the existing
`Extract`, matching the Task 11/`atlas-channel` generator shape. Round-trip tests
(`TestTransformRoundTrip`) construct a `Model`, `Transform` it, `Extract` it back, and
`reflect.DeepEqual` compare — same pattern as the approved Task 11 output.

### 4. Build/vet/test claim — PASS (re-verified independently)

Re-ran `go build ./... && go vet ./... && go test ./...` from
`services/atlas-guilds/atlas.com/guilds/character` and
`services/atlas-monster-book/atlas.com/monster-book/data/consumable` (without
`GOWORK=off`, since `GOWORK=off` in this environment produces an unrelated
"missing go.sum entry" error from a workspace-resolution quirk, not a defect in the
generated code). Both passed cleanly (`ok atlas-guilds/character`, `ok
atlas-monster-book/data/consumable`), consistent with the report's per-service test
counts. Did not re-run all 8 given the pattern is mechanically identical and Task
10/11's gate already validated the generator's shape; this is a reasonable sampling
given the review-surface scope.

### 5. `atlas-npc-conversations/atlas.com/npc/petdata` SKIPPED — judged independently, PASS

- **Clean skip, confirmed from git, not just the report:** `git diff --stat
  2a84f41f9..d0ba529db -- services/atlas-npc-conversations` is empty. `git log --oneline
  2a84f41f9..d0ba529db -- services/atlas-npc-conversations` returns no commits. No
  partial file, no stray generated fragment, nothing staged or committed for this
  service. The report's claim of "zero diff in range" is correct.
- **Mismatch characterized from source, confirmed correct:** `model.go:9` declares
  `evolutions int` on the domain `Model` (a scalar count), populated via
  `NewModel(..., evolutions int)`. `rest.go` declares `RestModel.Evolutions
  []EvolutionRestModel` (a slice of nested structs) and the existing hand-written
  `Extract` does `evolutions: len(rm.Evolutions)` — a lossy, one-directional reduction
  from slice to count. The generator's inverse `Transform` naively tried `Evolutions:
  m.evolutions` (assigning `int` into `[]EvolutionRestModel`), which does not
  type-check, and its own type-check gate caught it before any file was written. This is
  a **new** skip category, distinct from the four prior categories (single-return
  builder, unsupported field type, `Extract` discarding `RestModel`, pre-existing
  hand-written `Transform`, byte-overflow fixture) — correctly flagged by the report as
  requiring hand-authored work later (a real slice-vs-count model change, not a
  re-runnable flag tweak).

### 6. Single-return-builder skip count — verified 0, PASS

Checked the classification for all 9 packages in this batch: all 9 are tagged "flat
literal" in `classify-dom04.tsv` (no `Build()` builder types at all), and the batch's
one SKIPPED row's reason text (`cannot convert m.evolutions ... to type
[]EvolutionRestModel`) does not match the single-return-builder failure signature
(`assignment mismatch: 2 variables but ... Build returns 1 value`).

Cross-checked the cross-batch claim ("currently 4, all from batch 1") against
`ledger-transform-rest-1.tsv` and `ledger-transform-rest-2.tsv` directly:
`ledger-transform-rest-1.tsv` contains exactly 4 rows with that literal
`Build returns 1 value` signature (`atlas-messages/character`,
`atlas-npc-shops/character`, `atlas-monsters/monster/consumable`,
`atlas-maps/data/map/info`); `ledger-transform-rest-2.tsv` contains 0. Batch 3 correctly
contributes 0. The cross-batch tally of 4 (all batch 1) that later plan tasks will size
against is accurate as of this batch.

## Not evaluable

- gofumpt formatting was not independently re-verified — `gofumpt` is not installed in
  this environment (`command not found: gofumpt`), and per the brief and task
  instructions the controller gates formatting separately; `tools/verify.sh` was
  correctly not run by the implementer or by this review.
- Only 2 of the 8 APPLIED services' `go build/vet/test` were independently re-run; the
  other 6 are accepted on the report's stated per-service counts plus the structural
  consistency of every diff (mechanically identical generator shape across all 8,
  already gated once by the generator's own build/vet/test step per the brief's Step 1).

## Verdict

APPROVED. No blocking findings. The batch does exactly what the brief specified: 9
tier-A packages processed, 8 applied with clean per-service commits matching the
Task 11 shape, 1 cleanly skipped on a genuine, correctly characterized new failure mode,
ledger has no drops/duplicates, and the single-return-builder tally the controller needs
for later task sizing is accurate (0 this batch, 4 total, all attributable to batch 1).
