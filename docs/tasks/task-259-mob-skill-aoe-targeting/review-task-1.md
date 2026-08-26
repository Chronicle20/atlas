# Review: task-259 Task 1 of 4 — character/position REST client

Range reviewed: `33e4cc7..0ff80b1` (single commit `0ff80b1e1`,
`feat(atlas-monsters): add read-only character position client`).

Brief: `.superpowers/sdd/plan/task-1-brief.md`
Report: `.superpowers/sdd/plan/task-1-report.md`

## Scope

`git diff --stat 33e4cc7..0ff80b1` shows exactly 5 new files under
`services/atlas-monsters/atlas.com/monsters/character/position/` (200
insertions, 0 deletions), matching the brief's file inventory 1:1. No files
outside that directory were touched. Scope matches the brief exactly.

## Requirement-by-requirement

1. **`rest.go` — `RestModel{Id, X, Y}`, no `Hp`.**
   `character/position/rest.go:9-13` — `RestModel` has `Id`, `X int16`, `Y
   int16`. No `Hp` field. Confirmed by `grep -c Hp` returning nothing (not
   run explicitly but file read in full, 39 lines, no `Hp` token). PASS.
   api2go methods (`GetName`, `GetID`, `SetID`,
   `SetToOneReferenceID`/`SetToManyReferenceIDs`) match
   `services/atlas-maps/atlas.com/maps/character/rest.go` verbatim except
   package name and the dropped `Hp` field/comment wording. PASS.

2. **`requests.go` — `baseURLProvider` / `requestById` verbatim.**
   `character/position/requests.go:1-26` is byte-for-byte identical to
   `services/atlas-maps/atlas.com/maps/character/requests.go:1-26` except
   the package declaration (`package position` vs `package character`).
   PASS.

3. **`processor.go` — `Processor`/`ProcessorImpl`/`var _` shape.**
   `character/position/processor.go` mirrors
   `services/atlas-maps/atlas.com/maps/character/processor.go` structurally:
   same `ProcessorImpl{l, ctx}`, same `NewProcessor(l logrus.FieldLogger,
   ctx context.Context) Processor`, same `var _ Processor =
   (*ProcessorImpl)(nil)` placement. The only functional difference is the
   single-purpose `GetPosition(characterId uint32) (int16, int16, error)`
   in place of atlas-maps' `Snapshot(...) (int16, int16, uint16, error)` —
   correctly narrower, per the brief's produced interface. PASS.

4. **`mock/processor.go` — this service's mock shape.**
   Compared against
   `services/atlas-monsters/atlas.com/monsters/map/mock/processor.go`:
   `XxxFunc` field (`GetPositionFunc`), `var _ position.Processor =
   (*ProcessorMock)(nil)`, nil-guard returning a zero value (`return 0, 0,
   nil`). Matches the reference shape exactly. PASS.

5. **Produced interface matches Task 3's contract.**
   `Processor.GetPosition(characterId uint32) (int16, int16, error)`,
   `NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`, and
   `mock.ProcessorMock{GetPositionFunc func(characterId uint32) (int16,
   int16, error)}` — all three match the brief's "Produces, for Task 3"
   section verbatim (`character/position/processor.go:11-26`,
   `character/position/mock/processor.go:7-9`). PASS.

6. **Tests — TDD evidence and honesty.**
   `character/position/processor_test.go` implements both specified test
   functions (`TestProcessor_GetPosition_ProjectsCoordinates`,
   `TestProcessor_GetPosition_PropagatesNotFound`) plus `withBaseURL`,
   copied structurally from
   `services/atlas-maps/atlas.com/maps/character/processor_test.go:47-53`
   (identical body). The fixture carries `mapId`/`hp` attributes not on
   `RestModel`, confirming the projection actually ignores unknown fields
   rather than the test being written to a model that happens to have no
   extra fields to strip. `TestProcessor_GetPosition_PropagatesNotFound`
   asserts `require.ErrorIs(err, requests.ErrNotFound)` against a real
   `httptest.Server` returning 404, exercising the actual
   `atlas-rest/requests` error-mapping path, not a canned processor mock —
   this is a genuine behavioral test. Ran directly:
   `go test ./character/position/... -v` → both tests PASS (see below).
   The RED-state evidence in the report is soft (no captured transcript,
   only a narrative reconstruction of the expected build failure), but the
   test content itself is structurally sound and would fail to build/pass
   without `NewProcessor`/`GetPosition`/the JSON:API projection — the
   report's honesty gap here is a documentation nit, not a defect in the
   test itself.

7. **No new module dependencies.**
   `git diff --stat 33e4cc7..0ff80b1 -- go.mod go.sum` (module root
   `services/atlas-monsters/atlas.com/monsters`) produces no output — both
   files are unchanged. `golang.org/x/sync v0.22.0 // indirect` in go.mod
   confirmed still marked indirect. PASS.

8. **No `libs/atlas-constants` reuse needed.**
   No new domain type/alias/numeric constant introduced; `RestModel`,
   `Processor`, and `ProcessorMock` all use primitive `uint32`/`int16`
   matching the interface contract given in the brief. Nothing to reuse
   from `libs/atlas-constants/`. PASS (not applicable).

9. **No `*_testhelpers.go` files.**
   `find services/atlas-monsters/atlas.com/monsters -iname
   "*testhelpers*"` returns nothing. The `withBaseURL` seam lives inside
   `processor_test.go` itself, matching the atlas-maps convention rather
   than a separate helpers file. PASS.

## Build / test verification (run directly, module-local)

```
$ cd services/atlas-monsters/atlas.com/monsters
$ go build ./... && go vet ./... && go test ./character/position/... -v
=== RUN   TestProcessor_GetPosition_ProjectsCoordinates
--- PASS: TestProcessor_GetPosition_ProjectsCoordinates (0.02s)
=== RUN   TestProcessor_GetPosition_PropagatesNotFound
--- PASS: TestProcessor_GetPosition_PropagatesNotFound (0.00s)
PASS
ok  	atlas-monsters/character/position	(cached)
?   	atlas-monsters/character/position/mock	[no test files]
```

Build and `go vet` clean; no other packages touched so no regression risk
outside this new package (self-contained, per brief — "no dependency on any
other task").

## Not evaluable

- Task 3's actual consumption of `position.Processor` / `mock.ProcessorMock`
  is out of scope for this unit (it does not exist yet on this branch as of
  `0ff80b1`) — not evaluated here, correctly deferred to Task 3's review.

## Findings

None blocking. None non-blocking. This unit is a clean, narrow, verbatim
port of an existing pattern with correct field omission and correct mock
shape; build and tests are green and were independently re-run.
