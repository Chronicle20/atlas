# Task 12 batch 2 report

## Scope

Ran the Task 10/11-approved AST codemod
(`docs/tasks/task-263-backend-guideline-conformance/codemod`) against the 11 services assigned
to this batch, one service at a time, committing per service on PASS. Wrote
`docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest-2.tsv` (19 rows,
matching the classify-dom04.tsv tier-A count for these 11 services).

Steps 2-4 of the plan task (cross-batch ledger reconciliation, `handwork-dom04.tsv`) were **not**
run — per the brief, that is the controller's job after all three batches land. `progress.md` and
`agent-ledger.tsv` were not touched.

Invocation used the corrected form established by batch 1 (`go run -C <codemod-dir> . transform
...` with absolute `-repo`/`-classification`/`-ledger` paths, ledger written to `/tmp` per
service then appended by hand) — not re-derived, per the brief.

## Per-service results

| Service | APPLIED | SKIPPED | Reasons | Commit |
|---|---|---|---|---|
| atlas-trades | 2 | 0 | — | `0efabf17f` |
| atlas-reactors | 2 | 0 | — | `fa083d6ea` |
| atlas-mini-games | 2 | 0 | — | `4b152c1fd` |
| atlas-login | 2 | 0 | — | `2283a9ef8` |
| atlas-character | 2 | 0 | — | `a9baaece7` |
| atlas-chairs | 2 | 0 | — | `e33a308f2` |
| atlas-cashshop | 1 | 1 | 1 type-check failure (`character`) — see "New finding" below | `e68bd9a05` |
| atlas-ban | 2 | 0 | — | `885e987ed` |
| atlas-summons | 1 | 0 | — | `eb8422431` |
| atlas-storage | 0 | 1 | `asset` already has a hand-written `Transform` ("Transform already declared") — 0 APPLIED, no commit | none |
| atlas-saga-orchestrator | 1 | 0 | — | `6707f13e3` |

Totals: 17 APPLIED, 2 SKIPPED, 19 tier-A packages — matches
`awk -F'\t' '$2=="A"' classify-dom04.tsv` filtered to these 11 services.

**Single-return-builder skips in this batch: 0.** Neither skip in this batch was the
`transform.go:401-406` single-return-builder limitation carried forward from Task 11/batch 1.

**New finding — byte-field overflow in generated round-trip fixture (not previously seen):**
`atlas-cashshop/atlas.com/cashshop/character` was SKIPPED with:
```
generated code does not type-check: services/atlas-cashshop/atlas.com/cashshop/character/rest_test.go:38:23:
cannot use 319 (untyped int constant) as byte value in struct literal (overflows)
```
The model has several `byte`-typed fields (`Level`, `SkinColor`, `Gender`, `Stance`, etc. — see
`model.go`/`rest.go`). The generator's round-trip test fixture generator picked a value (319)
outside `byte` range (0-255) for one of these fields when synthesizing test data, so the
generated `rest_test.go` fails to compile. The generator rolled back cleanly (confirmed via
`git status --porcelain services/atlas-cashshop` showing only `data/pet` touched before the
`character` package's files were reverted) — no partial/broken output was left in the tree. This
is a distinct generator limitation from the single-return-builder case and should be reported to
the controller for sizing later hand-work: any tier-A package with a `byte`-typed field is a
candidate for the same failure.

`atlas-storage/atlas.com/storage/asset` reported "Transform already declared" — this package
already has a hand-written `Transform` method (pre-existing, not generator-produced), so the
generator correctly declined to touch it. Confirmed via `git status --porcelain
services/atlas-storage` showing no changes. No commit for this service.

## Per-service build/vet/test evidence

Each service (except `atlas-storage`, which had no changes) was verified from its
`atlas.com/<name>` module root with `go build ./... && go vet ./... && go test ./...`. All 10
services with APPLIED changes passed cleanly: no build failures, no vet warnings, every package
with a generated `rest_test.go` reported `ok`. Full command tails were captured for each run
during the session (visible in the tool-call transcript); `atlas-saga-orchestrator`'s output was
additionally grepped for `fail|error` (case-insensitive) with zero matches, confirming a clean
pass across its large package tree.

## Files changed

- `services/atlas-trades/atlas.com/trades/data/{character,map}/{rest.go,rest_test.go}` (2 packages)
- `services/atlas-reactors/atlas.com/reactors/reactor/data/{item,point}/{rest.go,rest_test.go}` (2 packages)
- `services/atlas-mini-games/atlas.com/mini-games/data/{character,map}/{rest.go,rest_test.go}` (2 packages)
- `services/atlas-login/atlas.com/login/{guild/member,ranking}/{rest.go,rest_test.go}` (2 packages)
- `services/atlas-character/atlas.com/character/{data/skill/effect/statup,skill}/{rest.go,rest_test.go}` (2 packages)
- `services/atlas-chairs/atlas.com/chairs/data/{map,setup}/{rest.go,rest_test.go}` (2 packages)
- `services/atlas-cashshop/atlas.com/cashshop/data/pet/{rest.go,rest_test.go}` (1 package; `character` SKIPPED, untouched)
- `services/atlas-ban/atlas.com/ban/{character,chat}/{rest.go,rest_test.go}` (2 packages)
- `services/atlas-summons/atlas.com/summons/effectivestats/{rest.go,rest_test.go}` (1 package)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/state/{rest.go,rest_test.go}` (1 package)
- `docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest-2.tsv` (new, 19 rows)

`atlas-storage` has no file changes (0 APPLIED); not listed above.

## Self-review

- Did not hand-edit any generated `Transform`/round-trip test code; every commit's diff is
  entirely generator output.
- Did not touch either SKIPPED package's `rest.go`/`rest_test.go` — left for later plan tasks.
- Did not hand-write a `Transform` for the SKIPPED `atlas-storage/asset` package, and did not
  commit anything for `atlas-storage` since there was nothing to commit.
- Did not run `tools/verify.sh`, `gofmt -l`, or any repo-wide check; only module-local
  `go build`/`go vet`/`go test` per service, as Contract 2 requires. Formatting conformance
  (gofumpt gate) is the controller's concern.
- Did not run plan Steps 2-4 (ledger reconciliation, `handwork-dom04.tsv`).
- Staged and committed only the paths named in the brief (`services/<S>` per commit). The ledger
  file `ledger-transform-rest-2.tsv` is written but **not yet committed** — see note below.
  Left `agent-ledger.tsv` and `progress.md` untouched.
- Sanitized one absolute worktree path the generator wrote into the ledger's SKIPPED reason
  column (`atlas-cashshop/character` row) to a repo-relative path before committing, per
  CLAUDE.md's rule against literal absolute paths in committed files under `docs/`.

## Note: ledger file not committed

The brief lists `docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest-2.tsv`
as a file this batch writes, but does not give an explicit commit instruction for it separately
from the per-service commits (unlike batch 1's report, which committed it — that commit is not
visible in this batch's scope and batch 1's own commit history shows it committed the ledger
separately). To stay consistent with batch 1's pattern, committing the ledger file now.

## Issues or concerns

1. **New generator limitation found**: byte-typed model fields can cause the generated
   round-trip fixture to use an out-of-range literal (see `atlas-cashshop/character` above). The
   controller should track this alongside the single-return-builder limitation when scoping later
   hand-work tasks (`handwork-dom04.tsv`). Zero single-return-builder skips were hit in this
   batch specifically.
2. `atlas-storage/asset` already had a hand-written `Transform`, so this batch produced no diff
   for that service — flagging so the controller's cross-batch reconciliation counts it correctly
   (0 APPLIED, 1 SKIPPED, but the SKIPPED reason is "already conformant," not a generator defect).
