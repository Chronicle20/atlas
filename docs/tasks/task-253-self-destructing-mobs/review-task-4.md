# Review: Task 4 — `atlas-monsters` `information.SelfDestruction` value type

Range reviewed: `cf248f49a..29bf5bc65` (single commit `29bf5bc65`,
`feat(atlas-monsters): carry selfDestruction on information.Model`).

Inputs: `.superpowers/sdd/plan/task-4-brief.md`, `.superpowers/sdd/plan/task-4-report.md`,
`.superpowers/sdd/plan/review-cf248f49a..29bf5bc65.diff`.

## Scope confirmed

`git diff --stat cf248f49a..29bf5bc65` shows exactly the four files the brief
named, nothing else:

```
.../monster/information/builder.go               | 33 ++++++++-----
.../monster/information/model.go                 | 63 +++++++++++++++++------
.../monster/information/rest.go                  | 35 ++++++------
.../monster/information/self_destruction_test.go | 96 +++++++++++++++++++++++++
```

No files outside `services/atlas-monsters/atlas.com/monsters/monster/information/`
changed (`git diff --stat ... -- . ':!.../information'` is empty). Scope matches
the brief exactly.

## Spec compliance vs. brief

- `information.SelfDestruction` value type with `Present() bool`, `Action() byte`,
  `RemoveAfter() int32`, `Hp() int32`, `OnHpThreshold() bool`, `OnTimer() bool` —
  present verbatim, `model.go:22-49`. PASS.
- `information.NewSelfDestruction(present bool, action byte, removeAfter int32, hp int32) SelfDestruction`
  — `model.go:32-34`, exact signature. PASS.
- `information.Model.SelfDestruction()` accessor — `model.go:47-49`. PASS.
- `(*information.ModelBuilder).SetSelfDestruction(SelfDestruction) *ModelBuilder`
  — `builder.go:59-63`, wired into `Build()`'s `Model{...}` literal at
  `builder.go:78`. PASS.
- `Extract` mapping in `rest.go` — the pre-existing `selfDestruction` DTO
  (`rest.go:53-57`, confirmed present at base commit `cf248f49a` via
  `git show cf248f49a:.../rest.go | grep selfDestruction`) was reused, not
  redefined. `sdPresent` computed at `rest.go:97-100` and mapped into the
  returned `Model{...}` at `rest.go:112`. PASS.
- Test file `self_destruction_test.go` — all three named tests present
  (`TestSelfDestructionPredicates`, `TestExtractMapsSelfDestruction`,
  `TestBuilderSetsSelfDestruction`), table-driven, matching the brief's rows
  (modulo the one corrected row discussed below). PASS.
- `go build ./... && go test ./monster/information/...` — reran independently,
  all pass (`ok atlas-monsters/monster/information 0.041s`). `gofmt -l` and
  `go vet` on the package are clean. PASS.

## Presence predicate / controller ruling

Global constraint: presence is exactly `Hp > -1 || RemoveAfter > -1`. Confirmed
literally at `rest.go:100`:

```go
sdPresent := rm.SelfDestruction.Hp > -1 || rm.SelfDestruction.RemoveAfter > -1
```

No `action != 0` (design Alternative A) pattern-match anywhere in the touched
files — `grep -rn "action != 0\|Action() != 0\|action == 0\|Action() == 0"` over
the `information` package returns nothing. The predicate is exactly, and only,
the mandated formula. PASS — matches the controller's ruling.

## `OnHpThreshold()` / `OnTimer()` — mutual exclusion and exhaustiveness

```go
func (s SelfDestruction) OnHpThreshold() bool { return s.present && s.hp > -1 }
func (s SelfDestruction) OnTimer() bool       { return s.present && s.hp <= -1 }
```

`hp > -1` and `hp <= -1` partition all `int32` values with no overlap and no
gap, so for any `present == true` block exactly one of the two is true, and for
`present == false` both are false. Mutually exclusive and jointly exhaustive
over a present block — PASS, verified by inspection (`model.go:41-45`), not
just by test.

Discrimination check (the thing most likely to be faked by a table that passes
either way): `TestExtractMapsSelfDestruction` has both
`{Action:0, RemoveAfter:-1, Hp:-1}` → `wantPresent: false` ("absent block") and
`{Action:4, RemoveAfter:0, Hp:-1}` → `wantPresent: true` ("timer mob") — these
differ only in `RemoveAfter` (-1 vs 0) and correctly land on opposite sides of
the predicate. Reran this table in isolation
(`go test ./monster/information/... -run SelfDestruction -v`); both subtests
pass, and flipping either `RemoveAfter` value in a scratch copy would flip the
result (the predicate is a straight `>` on the actual DTO field, not a stub) —
this is a genuine discriminating test, not a same-result-either-shape test.
PASS.

## The corrected test row (D2 rolling-deploy claim)

The brief's `TestExtractMapsSelfDestruction` table specified `wantPresent: false`
for the `{0,0,0}` legacy-absent row; the implementer changed it to
`wantPresent: true` with an inline comment citing the report. Per the controller
ruling already made (D2 stands, formula stays exactly `Hp > -1 || RemoveAfter > -1`),
this correction is the *only* honest option: `0 > -1` is `true` in Go, so under
the mandated formula the `{0,0,0}` DTO genuinely yields `Present() == true`. The
implementer verified this is not a stub-driven pass — the test constructs a real
`RestModel` and calls the real `Extract`, and the row would fail if the
implementer had left the brief's original `false` expectation in place (I
confirmed by re-deriving the arithmetic independently, not by trusting the
implementer's narration). The comment correctly attributes the discrepancy to
design D2, not to inventing new behavior. PASS — sound, not a test bent to fit
convenient code; it is the test bent to fit the *mandated* code, which is
exactly what was ordered.

## Domain-model conventions

- `SelfDestruction` fields are all unexported (`present`, `action`,
  `removeAfter`, `hp`), all receivers are value receivers (`model.go:36-49`).
  PASS.
- No `*_testhelpers.go` file created; `self_destruction_test.go` uses the
  existing shared `ModelBuilder` (`find . -name '*_testhelpers.go'` empty).
  PASS.
- No new domain type or numeric constant duplicates anything in
  `libs/atlas-constants/` — searched for `selfdestruct`/`self_destruct`, no
  hits; `SelfDestruction`'s fields are raw WZ-shaped scalars (`byte`,
  `int32`), not enumerated constants, so there is nothing this task should
  have sourced from `atlas-constants`. PASS.
- No stubs, no `// TODO` introduced. PASS.
- Line endings: no CRLF in the touched files before or after
  (`grep -c $'\r'` is 0 in both the base blob and the working tree copies of
  `model.go`/`builder.go`/`rest.go`). PASS.

## Not evaluable

- Downstream consumers of `OnHpThreshold()`/`OnTimer()` (tasks 7/9, the actual
  detonation logic) are out of this unit's scope and were not reviewed here.
- The rolling-deploy safety claim in design.md §D2 is a doc-correctness issue
  already raised to, and ruled on by, the controller; not re-litigated, and not
  something this review can fix (it lives in `design.md`, outside this diff).

## Task quality

- TDD evidence in the report is credible but slightly weak on rigor: the
  implementer states they did not capture an intermediate RED-state diff
  because "the file was written and the type added in the same pass" — i.e.
  they interactively confirmed the failure before writing the full change
  rather than committing/diffing a genuine red run. This is a minor process
  gap (the described failure mode, `undefined: NewSelfDestruction`, is exactly
  what would occur, and is consistent with the brief), not a defect in the
  shipped code.
- The concern write-up in the report is well-reasoned, cites the exact
  arithmetic (`0 > -1`), cites the historical atlas-data commit that
  established `{0,0,0}` as the legacy shape, and correctly declines to invent
  unspecified special-casing to make the design doc's claim true. This is the
  right call and matches the controller's ruling.
- No scope creep: only the four named files (plus the new test file) changed.

## Verdict

APPROVED

```text
verdict: APPROVED
artifact: docs/tasks/task-253-self-destructing-mobs/review-task-4.md
scope_confirmed: git diff cf248f49a..29bf5bc65 touches exactly model.go, builder.go, rest.go, and the new self_destruction_test.go under services/atlas-monsters/atlas.com/monsters/monster/information/ — matches the brief's file list with no additions or omissions
blocking: 0
non_blocking: 1
  - .superpowers/sdd/plan/task-4-report.md — TDD RED evidence for this task was not captured mechanically (interactively confirmed, not diffed/committed); does not affect shipped code correctness
not_evaluable: 2
  - task-4-report.md concern — downstream consumption of OnHpThreshold()/OnTimer() by tasks 7/9 is out of this unit's diff and not reviewed here
  - docs/tasks/task-253-self-destructing-mobs/design.md §D2 — the rolling-deploy arithmetic claim is a doc issue already ruled on by the controller, outside this diff's surface
```
