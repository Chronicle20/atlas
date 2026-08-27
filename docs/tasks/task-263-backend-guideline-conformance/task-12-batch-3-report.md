# Task 12 batch 3 of 3 — W3 apply `transform` to tier-A services

## Summary

Ran the DOM-04 `transform` codemod (`docs/tasks/task-263-backend-guideline-conformance/codemod`)
against the 9 services in this batch, one commit per service that produced a change. All
invocations used the corrected, non-`-append` form: each run wrote to a scratch
`/tmp/ledger-<S>.tsv`, whose contents were appended to
`docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest-3.tsv` after the
build/vet/test gate passed for that service.

Invocation used for every service (absolute paths, `GOWORK=off`):

```
GOWORK=off go run -C <worktree>/docs/tasks/task-263-backend-guideline-conformance/codemod . transform \
  -repo <worktree> \
  -classification <worktree>/docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv \
  -only services/<S> \
  -ledger /tmp/ledger-<S>.tsv
```

Then, from `services/<S>/atlas.com/<name>`: `go build ./... && go vet ./... && go test ./...`.
Only on PASS was the ledger scratch file appended and the service path committed.

## Per-service results

| Service | Package (tier-A) | Result | Commit |
|---|---|---|---|
| atlas-rankings | `atlas.com/rankings/character` | APPLIED | `449d2c0` feat(atlas-rankings): add Transform and round-trip tests |
| atlas-party-quests | `atlas.com/party-quests/guild` | APPLIED | `fcb2902` feat(atlas-party-quests): add Transform and round-trip tests |
| atlas-npc-conversations | `atlas.com/npc/petdata` | SKIPPED | none — see below |
| atlas-monster-book | `atlas.com/monster-book/data/consumable` | APPLIED | `7a5ed8621` feat(atlas-monster-book): add Transform and round-trip tests |
| atlas-marriages | `atlas.com/marriages/character` | APPLIED | `6c2a44a43` feat(atlas-marriages): add Transform and round-trip tests |
| atlas-guilds | `atlas.com/guilds/character` | APPLIED | `5d28ddf1b` feat(atlas-guilds): add Transform and round-trip tests |
| atlas-fame | `atlas.com/fame/character` | APPLIED | `5a92f7c01` feat(atlas-fame): add Transform and round-trip tests |
| atlas-doors | `atlas.com/doors/data/skill/effect` | APPLIED | `d651a8181` feat(atlas-doors): add Transform and round-trip tests |
| atlas-buddies | `atlas.com/buddies/character` | APPLIED | `0f4dc6e17` feat(atlas-buddies): add Transform and round-trip tests |

**Totals for this batch:** 8 APPLIED, 1 SKIPPED, 0 single-return-builder skips.

## APPLIED count / SKIPPED count with reasons (per brief format)

Each service in this batch has exactly one tier-A package, so the counts above are
per-package = per-service:

- APPLIED: 8 (atlas-rankings, atlas-party-quests, atlas-monster-book, atlas-marriages,
  atlas-guilds, atlas-fame, atlas-doors, atlas-buddies)
- SKIPPED: 1 (atlas-npc-conversations — `atlas.com/npc/petdata`)
  - Reason (generator-reported): "generated code does not type-check:
    `.../petdata/rest.go:96:37: cannot convert m.evolutions (variable of type int) to
    type []EvolutionRestModel`"
  - No files were modified for this service (`git status --short services/atlas-npc-conversations`
    was empty after the run); nothing was staged or committed, per the brief's instruction not
    to hand-write a `Transform` for a SKIPPED package.

## Single-return-builder skip count

**0** in this batch. All SKIPPED reasons were type-mismatch generation failures
(see below), not the `Build() Model` single-return category from batch 1.

## New skip category observed (not seen in prior batches)

`atlas-npc-conversations/atlas.com/npc/petdata`: the generator produced a `Transform`
whose body assigns `m.evolutions` (an `int` field on the domain model) directly into a
`[]EvolutionRestModel` slice field on the generated `RestModel`. This is a field-name
collision where a scalar count field (`evolutions int`) and a would-be nested-slice
field share the stem the generator inferred a name from, and the generator's field
matcher paired them incorrectly. This is distinct from every category catalogued so far
(single-return builders, unsupported field types map/[]uint32, Extract discarding
RestModel, a pre-existing hand-written Transform, and the byte-overflow fixture). The
codemod caught its own error via the type-check gate and skipped cleanly — no partial
file was left behind. Recorded as a new row for the controller's cross-batch tally:
"generated Transform assigns scalar field into mismatched slice field" — count 1.

## Verification

For each APPLIED service, ran from `services/<S>/atlas.com/<name>`:

```
go build ./... && go vet ./... && go test ./...
```

All 8 passed cleanly:

- atlas-rankings: `Go test: 70 passed in 7 packages`
- atlas-party-quests: `Go test: 142 passed in 26 packages`
- atlas-monster-book: `Go test: 51 passed in 12 packages`
- atlas-marriages: `Go test: 557 passed in 13 packages`
- atlas-guilds: `Go test: 135 passed in 25 packages`
- atlas-fame: `Go test: 37 passed in 11 packages`
- atlas-doors: `Go test: 61 passed in 18 packages`
- atlas-buddies: `Go test: 61 passed in 17 packages`

Per Contract 2 / the brief, `tools/verify.sh` was not run — the controller gates
formatting (gofumpt) and repo-wide checks separately.

## Files changed

- `services/atlas-rankings/atlas.com/rankings/character/rest.go` (Transform added)
- `services/atlas-rankings/atlas.com/rankings/character/rest_test.go` (new)
- `services/atlas-party-quests/atlas.com/party-quests/guild/rest.go` (Transform added)
- `services/atlas-party-quests/atlas.com/party-quests/guild/rest_test.go` (new)
- `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest.go` (Transform added)
- `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest_test.go` (new)
- `services/atlas-marriages/atlas.com/marriages/character/rest.go` (Transform added)
- `services/atlas-marriages/atlas.com/marriages/character/rest_test.go` (new)
- `services/atlas-guilds/atlas.com/guilds/character/rest.go` (Transform added)
- `services/atlas-guilds/atlas.com/guilds/character/rest_test.go` (new)
- `services/atlas-fame/atlas.com/fame/character/rest.go` (Transform added)
- `services/atlas-fame/atlas.com/fame/character/rest_test.go` (new)
- `services/atlas-doors/atlas.com/doors/data/skill/effect/rest.go` (Transform added)
- `services/atlas-doors/atlas.com/doors/data/skill/effect/rest_test.go` (new)
- `services/atlas-buddies/atlas.com/buddies/character/rest.go` (Transform added)
- `services/atlas-buddies/atlas.com/buddies/character/rest_test.go` (new)
- `docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest-3.tsv` (new — this batch's ledger)
- `docs/tasks/task-263-backend-guideline-conformance/task-12-batch-3-report.md` (this report)

No other files were touched. `agent-ledger.tsv` and `progress.md` showed as modified in
`git status` during this run but that was from a concurrent process (not this agent);
they were never staged or committed by this task, per the brief's instruction to leave
`progress.md`/`agent-ledger.tsv` untouched.

## Self-review

- Confirmed `-only services/<S>` scoped each run to exactly one service; the ledger
  scratch file for every run listed exactly one package, matching the one tier-A row per
  service in `classify-dom04.tsv` for this batch.
- Confirmed each commit's diff is limited to the service's own path (`git add
  services/<S>` per commit, never a broad `git add`).
- Confirmed branch stayed `task-263-backend-guideline-conformance` and worktree stayed
  the correct one after every commit.
- Did not hand-edit any generated `Transform` or test file; the one SKIPPED package was
  left untouched, not hand-patched.
- Did not run `tools/verify.sh`, repo-wide `go build`/`go vet`, or `-race` tests, per
  Contract 2.
- Did not touch `progress.md`, `agent-ledger.tsv`, or run Steps 2-4 of the plan task
  (cross-batch ledger reconciliation, `handwork-dom04.tsv`), per the controller's brief.

## Issues / concerns

- One SKIPPED package (`atlas-npc-conversations/atlas.com/npc/petdata`) needs
  hand-authored `Transform`/round-trip test work in a later plan task — it is a genuine
  field-type mismatch in the domain model (`evolutions int` vs. a slice-shaped REST
  field), not something producible by re-running the generator with different flags.
- No single-return-builder skips in this batch (all 9 packages are "flat literal" per
  `classify-dom04.tsv`, no nested `Build()` builder types), so batch 3 contributes 0 to
  that cross-batch tally.
