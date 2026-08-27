# Task 14, batch B review — `NO-RESTMODEL` hand work (D2)

verdict: APPROVED_WITH_FINDINGS

## Scope

Reviewed commits `0642cd0`, `1e6c650`, `3c489d8` against
`.superpowers/sdd/plan/task-14-brief-b.md`. `git diff --name-only 0642cd0^..3c489d8`
touches exactly the three package pairs named in the brief plus
`handwork-notes.md` and `task-14b-report.md` — no file outside the declared
scope was touched, and no package outside the three gained a `Transform`.

## 1. Exact-inverse check (D1)

### `services/atlas-pets/atlas.com/pets/data/position` (`0642cd0`)

- `Model{id, x1, y1, x2, y2}` (`model.go:3-8`).
- `Extract` (`rest.go:31-38`) maps all five fields from `FootholdRestModel{Id, First{X,Y}, Second{X,Y}}`.
- `Transform` (`rest.go:41-48`) maps all five fields back, reading `m.id/m.x1/m.y1/m.x2/m.y2` directly (unexported, same package) — no accessor minted.
- `PositionRestModel` has no `Extract` in this package; correctly left untouched and called out in the handwork note.
- PASS.

### `services/atlas-saga-orchestrator/.../rates` (`1e6c650`)

- `Model{expRate, mesoRate, itemDropRate, questExpRate}` (`model.go:3-8`).
- `Extract` (`rest.go:24-30`) reads only `body.Attributes.*`; ignores `DataBody.Id`/`Type`.
- `Transform` (`rest.go:34-44`) emits only `Attributes{...}` from `m.expRate` etc., leaving `Id`/`Type` at zero value — correctly matches "field the Extract does not map must not be emitted."
- Bare `Extract`/`Transform` naming, matching FR-2 for a bare (unsuffixed) pair. PASS.

### `services/atlas-saga-orchestrator/.../reactor/drop` (`1e6c650`)

- `Model{reactorId, itemId, questId, chance}` (`model.go:3-9`).
- `Extract` (`rest.go:131-137`) maps all four from `DropRestModel`.
- `Transform` (`rest.go:140-147`) maps all four back; `DropRestModel.Id` is not emitted (not read by `Extract`), matching the handwork note.
- `ReactorRestModel`, `DropPositionInputModel`, `PositionRestModel` have no `Extract` in this package; correctly left untouched. PASS.

## 2. Naming (FR-2)

All three follow the mirrored form: bare `Extract` → bare `Transform` in all
three packages (none of the three has an `Extract<X>` suffix form in this
batch, unlike batch A's `ExtractForeign`/`TransformForeign`). Consistent with
the brief. PASS.

## 3. Round-trip test honesty

Every test constructs a `Model` with five/four distinct, non-zero field
values (`42,100,200,300,400`; `1.5,2.5,3.5,4.5`; `10,20,30,40`).

Confirmed non-tautological by direct mutation (not just trusting the report):

- `position`: swapped `First.X` from `m.x1` to `m.x2` → test failed with a
  clear field-level diff (`x1:100` expected vs `x1:300` got). Reverted; `git
  status --short services/atlas-pets` clean afterward.
- `rates`: swapped `MesoRate` from `m.mesoRate` to `m.expRate` → test failed
  (`mesoRate:2.5` expected vs `mesoRate:1.5` got). Reverted; `git status
  --short services/atlas-saga-orchestrator` clean afterward.

The report's RED transcript (`rest_test.go:13:13: undefined: Transform` for
position, `:13:10` for rates, `:18:8` for reactor/drop) is consistent with
where `Transform` is first referenced in each committed test file. GREEN
re-run confirmed independently: `go test ./data/position/... ./rates/...
./reactor/drop/... -run TestTransformRoundTrip -v` all PASS.

PASS — the RED-run requirement (brief Step 3, called out because the two
prior tasks skipped it) was genuinely met, not just claimed.

## 4. `handwork-notes.md` coverage

`docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md`
(`3c489d8`) has one line for all three packages under a `## Batch B (Task 14)`
heading, in the same form as the batch-A entries (wire types, `Extract`
location, what `Transform` provides, what's out of scope and why). Cited
`Extract` line numbers verified exact: `rest.go:31` (position), `rest.go:24`
(rates), `rest.go:131` (reactor/drop) — all match `grep -n '^func Extract'`
output. All paths are repo-relative; `grep -n '/home/\|~/'` over both the
notes file and the report returned nothing. PASS.

## 5. Comparison with batch-A `atlas-drops/.../data/foothold`

Read `d633c9a`. The two `Extract`/`Transform` pairs and their `Model`s are
structurally identical (`id,x1,y1,x2,y2` → nested `First`/`Second`). The only
difference: `atlas-pets/position/model.go` exposes `NewModel(...)`, so the
batch-B test builds via `NewModel(42,100,200,300,400)`
(`rest_test.go:11`), while `atlas-drops/foothold` has no constructor and the
batch-A test uses a composite literal. That divergence is explained by the
package's own API surface, not an unexplained inconsistency — no finding.

## 6. Scope confirmation

`git diff --name-only 0642cd0^..3c489d8` lists exactly the 6 source/test
files plus `handwork-notes.md` and the report. No other package in the repo
gained a `Transform` in this range.

## Findings

### Non-blocking

- `services/atlas-pets/atlas.com/pets/data/position/rest.go` fails
  `gofumpt -l` (import-grouping: `strconv` and `atlas-pets/point` need a
  blank line between std and local imports). Verified this predates the
  batch: `git show 0642cd0^:.../rest.go` piped through `gofumpt -l` (with a
  synthetic `go.mod` alongside it) already flags the same file before this
  commit's diff — the diff only appended the `Transform` function after the
  existing import block and did not touch the imports. Pre-existing debt,
  not introduced by this unit; not a defect of this batch, but the module's
  `gofumpt -l ./...` gate will not be clean until it (or a broader sweep) is
  fixed elsewhere. The implementer's report claim that `tools/lint.sh
  --check --fmt --go services/atlas-pets/atlas.com/pets` returned `OK` should
  be read as "no *new* file needed a rewrite" rather than "the module gate is
  fully clean" — worth a note for whoever runs the final module-wide
  `verify.sh` gate so this pre-existing gap isn't mistaken for something this
  batch should have caught.

## Not evaluable

None — the full diff surface (6 source files + notes + report) was read and
independently exercised (build, vet, round-trip test, mutation test).
