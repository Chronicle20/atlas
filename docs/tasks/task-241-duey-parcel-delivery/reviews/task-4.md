# Review: Task 4 — atlas-parcel REST surface (`rest.go`, `resource.go`, `main.go` wiring)

Range: `91dc1cae9..cae523317` (1 commit)
Files reviewed: `services/atlas-parcel/atlas.com/parcel/parcel/rest.go`,
`services/atlas-parcel/atlas.com/parcel/parcel/resource.go`,
`services/atlas-parcel/atlas.com/parcel/parcel/resource_test.go`,
`services/atlas-parcel/atlas.com/parcel/main.go` (diff only), plus the
`Processor`/error-taxonomy contract in `processor.go`/`errors.go` (read for the
correctness of what this diff calls, not re-audited).

## Scope confirmation

Diff matches the brief exactly: two new files (`rest.go`, `resource.go`), a new
test file, and a 4-line additive change to `main.go`'s `server.New` chain. No
scope creep, no touched files outside the brief's list.

## Requirement-by-requirement

- `RestModel` with JSON:API tags for every `Model` field, `Transform`,
  `GetID`/`SetID` — present, `rest.go:14-58`. Field-for-field match against
  task-2's documented getter list; spot-checked, no field dropped.
- `InitResource(si) func(db) server.RouteInitializer` — present,
  `resource.go:31`, signature matches the brief exactly.
- Four routes registered — present, `resource.go:37-41`:
  `GET /parcels`, `GET /parcels/{parcelId}`,
  `GET /characters/{characterId}/parcel-status`. (Brief lists the two `/parcels`
  filter shapes as if separate GETs; they are correctly implemented as one
  route dispatching on which filter is set — `resource.go:53-119`.)
- `main.go` wiring — additive, in the position the brief specifies (before
  `/debug/consumers` and `/readyz`), `main.go` diff hunk at line ~64. Correct.
- Builder pattern used for test seeding (`seedParcel` → `parcel.NewBuilder()`),
  no `*_testhelpers.go` file. Confirmed.
- `libs/atlas-constants` reused (`world.Id`), no new constant introduced.
  Confirmed by grep — no new type/const declarations in either new file
  besides `parcelStatusRestModel`, which is a REST wire model, not a domain
  constant.
- "Never disconnect on malformed request" — every malformed-input path in
  `resource.go` (bad `filter[recipientId]`/`filter[senderId]`/`filter[worldId]`
  value, bad `filter[status]` value, bad `parcelId` uuid, missing both list
  filters) returns `400` via `server.WriteBadRequest`, logged at `Warnf`.
  Confirmed by reading all four handlers end to end — no path panics or
  reaches an unguarded type assertion.

## Item 1 — the defaulting judgment call

**Finding: real defect, not a safe variant of the pattern task-2 and task-3
were flagged for. It is the same class of bug, at the REST boundary instead
of the query/provider boundary.**

`resource.go:79-88`:

```go
worldId := world.Id(0)
if v := q.Get("filter[worldId]"); v != "" {
    ...
    worldId = world.Id(byte(parsed))
}
ms, err := p.GetForRecipient(uint32(recipientId), worldId)
```

`world.Id` is `type Id byte` (`libs/atlas-constants/world/constants.go:3`) —
there is no sentinel "unset" value distinct from world 0; `world.Id(0)` is an
ordinary, real world. A client that calls
`GET /parcels?filter[recipientId]=100` without `filter[worldId]` gets a clean
`200` scoped silently to world 0. If the recipient's parcels are on world 3,
the response is an empty (or wrong) list with no error, no warning, and
nothing in the response to indicate the query was implicitly narrowed. That is
exactly the shape of defect Task 2's reviewer called "a real multi-world
isolation bug" and Task 3's reviewer just filed blocking against
(`HasInFlight` hardcoding `world.Id(0)`) — the difference is that here the
caller *can* supply the correct value, but a caller who doesn't know to is
silently given the wrong-scoped answer instead of a `400` telling them the
parameter is required.

The implementer's stated justification — the brief's own "tenant isolation"
test row supplies only `filter[recipientId]=100` and expects `200` — does not
hold up under scrutiny:

- That test row exists to test tenant isolation (cross-tenant leakage under
  the same header machinery), not world defaulting. `seedParcel` in this test
  file always seeds `SetWorldId(0)` (`resource_test.go:69`), so the test
  passes identically whether `filter[worldId]` is required-and-defaulted or
  required-and-missing-would-400 — it cannot distinguish the two designs, and
  therefore cannot be read as "pinning" the defaulting resolution.
- The one-line alternative that satisfies both the letter of that test row
  *and* closes the isolation gap is trivial: make `filter[worldId]` required
  (`400` when absent) and add `&filter[worldId]=0` to the tenant-isolation
  test's URL, exactly as the "list by recipient" test row already does
  (`resource_test.go:120`, which supplies `filter[worldId]=0` explicitly).
  Nothing in the brief's Step 3 prose or route signature
  (`filter[recipientId]=&filter[worldId]=&filter[status]=`) reads as "these
  are optional with defaults" — it reads as "these are the filters this route
  accepts," and the one worked example in the table that isn't testing world
  scoping happens to omit it.

The `filter[status]`-defaults-to/only-accepts-`"pending"` half of the same
judgment call is **not** a defect: `GetForRecipient` and `GetPendingForSender`
are permanently `StatusPending`-scoped in the Processor (task-3's own
documented scope decision, `processor.go:72,78`), so there is no non-pending
data this route could ever expose regardless of what `filter[status]` says.
Rejecting a non-`"pending"` value with `400` rather than silently ignoring it
is correct and matches the "never silently drop a caller-supplied filter"
convention the implementer cites. No finding here.

**Impact on task-26 gate 12**: the `/characters/{characterId}/parcel-status`
route does not take or default a `worldId` at all — it forwards straight to
`Processor.HasInFlight(characterId)`, which has no `worldId` parameter in its
signature. Gate 12 is therefore not affected by this REST-layer defaulting
bug; it is affected only by Task 3's already-filed blocking finding
(`HasInFlight`'s inbound check hardcoding `world.Id(0)`), which is out of this
task's scope and already in a fix round. Flagging this only so the two are not
conflated: this task's defect is the `/parcels?filter[recipientId]=` route
silently mis-scoping to world 0 when a caller omits `worldId`, not gate 12.

## Item 2 — `ErrNotFound` mapping

**Finding: the 404 path is not dead, but it bypasses the Processor's error
taxonomy entirely, and that is a real risk for Task 3's in-flight fix round.**

`resource.go:145-150`:

```go
m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetById(parcelId)
if errors.Is(err, gorm.ErrRecordNotFound) {
    w.WriteHeader(http.StatusNotFound)
    return
}
```

`Processor.GetById` (`processor.go:65`) forwards the raw provider error
unwrapped: `return ById(id)(p.db.WithContext(p.ctx))()`. Task 3's reviewer
correctly found that this never produces `parcel.ErrNotFound` — but
`resource.go` never checks for `parcel.ErrNotFound` in the first place. It
checks `gorm.ErrRecordNotFound` directly, reaching past the Processor's
documented error taxonomy (`errors.go:11-12`, unused outside
`processor.go:151`'s unrelated branch and its own test) straight into the
persistence library's sentinel error. Functionally, today, this works: GORM's
`First`/`Find`-style lookups return `gorm.ErrRecordNotFound` on a missing row,
`errors.Is` matches it, and the `resource_test.go` "get by id missing" subtest
passes for real (confirmed by reading the assertion — it hits a genuinely
empty DB and asserts `404`, not a stub).

The concern is forward-compatibility, not present correctness:

- This is a layering violation — the REST handler should depend on the
  domain's own `ErrNotFound`, which is exactly what `errors.go` was written
  for. Depending on `gorm.ErrRecordNotFound` directly means this handler now
  knows about the store's implementation detail, and any future provider
  swap or wrapping change breaks it silently unless someone remembers this
  coupling exists.
- Task 3's fix round is specifically going to change `GetById` (or the
  `ById` provider) to map a missing row to `parcel.ErrNotFound`. Depending on
  how that wrap is implemented, `errors.Is(err, gorm.ErrRecordNotFound)` may
  or may not continue to match afterward — e.g. if the fix returns
  `parcel.ErrNotFound` as a fresh sentinel rather than
  `fmt.Errorf("%w: %w", parcel.ErrNotFound, gorm.ErrRecordNotFound)` (multi-
  wrap), this handler's `errors.Is(err, gorm.ErrRecordNotFound)` check stops
  matching and the `404` silently degrades to a `500` (falls through to the
  generic `server.WriteErrorResponse` branch below it, `resource.go:152-155`).
  Nothing in this diff or Task 3's report constrains that outcome.

This is not blocking against Task 4 as written — the 404 path works, is
tested, and is not dead code. It is flagged as a coordination note for
whoever executes Task 3's fix round: after that fix lands, `resource.go:148`
must be updated to check `errors.Is(err, parcel.ErrNotFound)` (the domain
sentinel this diff's own doc comment on `handleGetParcel` implies it should
be using), not `gorm.ErrRecordNotFound`, or the fix round must guarantee the
gorm sentinel stays reachable via `errors.Is`. Either way, this file needs a
follow-up touch when Task 3's fix lands — it should not be assumed
untouched.

## Correctness of the change itself

- `writeParcels`/`handleGetParcel`/`handleGetParcelStatus` all use
  `model.SliceMap`/`model.Map` + `server.MarshalResponse` consistently with
  the referenced `holding`/`frederick` patterns. No error path swallowed.
- `parcelStatusRestModel.Id` is the character id (`strconv.FormatUint`), not a
  parcel id — documented and consistent with "the resource identifier is the
  character."
- Tenant scoping: every route goes through `rest.RegisterHandler(l)(db)(si)`
  (task-1 utility, unmodified by this diff), which threads the tenant-scoped
  context into `d.Context()`/`d.DB()`; no handler adds or needs a manual
  tenant filter. The "tenant isolation" test exercises this for real (two
  distinct tenant UUIDs, cross-tenant query returns empty). Confirmed correct.
- `main.go` diff is purely additive — one new `AddRouteInitializer` call,
  no restructuring of the existing chain. Confirmed by reading the full
  hunk.

## Test honesty

All 7 subtests in `resource_test.go` exercise real HTTP round trips against
an in-memory tenant DB with real seeded rows; none are trivially
true/pass-either-way except the caveat under Item 1 above — the "tenant
isolation" subtest's specific omission of `filter[worldId]` does not
distinguish between "worldId optional-with-default" and "worldId
required-and-this-row-should-have-included-it," which is exactly why it
cannot be read as validating the defaulting design decision it was cited to
justify.

## Not evaluable

- `rest.RegisterHandler`, `server.WriteBadRequest`, `server.WriteErrorResponse`,
  `server.MarshalResponse`, `ParseParcelId`, `ParseCharacterId` — all task-1
  utilities, unmodified by this diff. Their correctness is assumed from
  task-1's own review, not re-audited here (out of this unit's scope).
- Whether Task 3's fix round will preserve or break the
  `gorm.ErrRecordNotFound` coupling in `handleGetParcel` cannot be evaluated
  now — that fix does not exist yet. Recorded as a coordination note above,
  not scored as a defect against this diff.

## Verdict rationale

Item 1 (the `filter[worldId]` silent default to world 0 on the recipient-list
route) is a genuine, in-scope defect in this diff — it is caller-facing REST
API behavior, not a hypothetical, and it repeats a defect class this same
plan has twice already treated as blocking. It is blocking.

Item 2 is real but not blocking against this diff as shipped (the 404 path
works and is tested); it is a coordination note for the Task 3 fix round,
filed as non-blocking here.
