# Task 12 batch 1 report

## Scope

Ran the Task 10/11-approved AST codemod
(`docs/tasks/task-263-backend-guideline-conformance/codemod`) against the 9 services assigned to
this batch, one service at a time, committing per service on PASS. Wrote
`docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest-1.tsv` (43 rows,
matching the classify-dom04.tsv tier-A count for these 9 services) and committed it separately.

Steps 2-4 of the plan task (cross-batch ledger reconciliation, `handwork-dom04.tsv`) were **not**
run — per the brief, that is the controller's job after all three batches land.

## Invocation note (not in the brief, needed by continuation batches)

The brief's literal `go run ./docs/tasks/.../codemod transform ...` command fails with `go: cannot
find main module` because the codemod is its own Go module nested inside the outer worktree
(which is not itself a Go module at the root). It must be invoked with `go run -C <codemod-dir> .
transform ...` and absolute `-repo`/`-classification`/`-ledger` paths (or paths relative to the
worktree root, passed as absolute via `<repo-root>/...`). Example used for every service in this
batch:

```
GOWORK=off go run -C docs/tasks/task-263-backend-guideline-conformance/codemod . transform \
  -repo <repo-root> \
  -classification <repo-root>/docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv \
  -only services/<S> \
  -ledger /tmp/ledger-<S>.tsv
```

where `<repo-root>` is the worktree root (`git rev-parse --show-toplevel`).

Also: the tool has **no `-append` flag** (brief's example command included one that does not
exist and errors with "flag provided but not defined"). Each run overwrites `-ledger` fully, so I
wrote each service's ledger to a scratch path under `/tmp` and appended the scratch file's
contents onto `docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest-1.tsv`
myself between the per-service transform run and the per-service commit.

## Per-service results

| Service | APPLIED | SKIPPED | Reasons | Commit |
|---|---|---|---|---|
| atlas-monster-death | 8 | 0 | — | `81922b270` |
| atlas-messages | 6 | 2 | 1 single-return-builder (`character`); 1 "Extract maps no fields" (`data/map`) | `3c21ef059` |
| atlas-consumables | 6 | 1 | 1 unsupported field type `map[...]int32` (`cash`) | `7a7e489b0` |
| atlas-npc-shops | 3 | 1 | 1 single-return-builder (`character`) | `60c3ec452` |
| atlas-inventory | 4 | 0 | — | `e66e25ceb` |
| atlas-query-aggregator | 3 | 0 | — | `9185d8bbb` |
| atlas-pets | 3 | 0 | — | `8ed6edaf3` |
| atlas-monsters | 1 | 2 | 1 single-return-builder (`monster/consumable`); 1 unsupported field type `[]uint32` (`monster/mobskill`) | `0dddbda0d` |
| atlas-maps | 2 | 1 | 1 single-return-builder (`data/map/info`) | `f949577ac` |

Totals: 36 APPLIED, 7 SKIPPED, 43 tier-A packages — matches
`awk -F'\t' '$2=="A"' classify-dom04.tsv` filtered to these 9 services.

**Single-return-builder skips in this batch: 4** (`atlas-messages/character`,
`atlas-npc-shops/character`, `atlas-monsters/monster/consumable`, `atlas-maps/data/map/info`),
all matching the known `transform.go:401-406` limitation (Builder `Build()` returning a single
value, not `(Model, error)`). No service in this batch produced 0 APPLIED, so every service has a
commit.

## Per-service build/vet/test evidence

Each service was verified from its `atlas.com/<name>` module root with
`go build ./... && go vet ./... && go test ./...`; all 9 passed cleanly (no failures, no vet
warnings). Sample tails were captured for each run during the session; all reported `ok` for every
package with generated `rest_test.go` round-trip tests and no build/vet errors.

## Files changed

- `services/atlas-monster-death/atlas.com/monster/**/{rest.go,rest_test.go}` (8 packages)
- `services/atlas-messages/atlas.com/messages/**/{rest.go,rest_test.go}` (6 packages)
- `services/atlas-consumables/atlas.com/consumables/**/{rest.go,rest_test.go}` (6 packages)
- `services/atlas-npc-shops/atlas.com/npc/**/{rest.go,rest_test.go}` (3 packages)
- `services/atlas-inventory/atlas.com/inventory/**/{rest.go,rest_test.go}` (4 packages)
- `services/atlas-query-aggregator/atlas.com/query-aggregator/**/{rest.go,rest_test.go}` (3 packages)
- `services/atlas-pets/atlas.com/pets/**/{rest.go,rest_test.go}` (3 packages)
- `services/atlas-monsters/atlas.com/monsters/monster/drop/{rest.go,rest_test.go}` (1 package)
- `services/atlas-maps/atlas.com/maps/data/map/{reactor,script}/{rest.go,rest_test.go}` (2 packages)
- `docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest-1.tsv` (new, 43 rows)

## Self-review

- Did not hand-edit any generated `Transform`/round-trip test code; every commit's diff is
  entirely generator output plus the ledger file I wrote by hand-concatenation.
- Did not touch any SKIPPED package's `rest.go`/`rest_test.go` — skips are left for later plan
  tasks per the brief.
- Did not run `tools/verify.sh`, `gofmt -l`, or any repo-wide check; only module-local
  `go build`/`go vet`/`go test` per service, as Contract 2 requires. Formatting conformance
  (gofumpt gate) is the controller's concern per the brief's carry-forward note #2.
- Did not run plan Steps 2-4 (ledger reconciliation, `handwork-dom04.tsv`).
- Staged and committed only the paths named in the brief (`services/<S>` per commit, then the
  ledger file). Left `agent-ledger.tsv` and `progress.md` (modified by the dispatching
  infrastructure, not by me) uncommitted, since they are outside this brief's file list.

## Issues or concerns

None. All 9 services built, vetted, and tested clean on first generator pass; no service needed a
second attempt or produced a type-check failure outside the known single-return-builder and
unsupported-field-type skip categories already documented from Task 11.
