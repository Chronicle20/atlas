# Review: post-merge defect-fix round (task-278)

Range: `9b7817a13..HEAD` (HEAD = `5efddd01b`)
Commits: `e050686d4`, `da20524d3`, `b3706768d`, `5efddd01b`

## Scope

Reviewed the diff of the four commits plus the seam files they call into
(atlas-channel consumer, atlas-maps environment processor/registry,
atlas-data map reader/resource, ingress `routes.conf`). Did not re-derive WZ
polarity — the observed in-client value (`103000800` gate visible state `0`,
declared default `l2=1`) is treated as ground truth per instructions. Widening
into atlas-data is explicit user-directed scope, not flagged.

`git diff --stat` matches the four commits described in the brief; no
unrelated files are touched. Scope confirmed.

## 1. Cross-service seam: `EnvironmentReset.Cleared[]`

- `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go` and
  `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go` define
  `EnvironmentObject{Kind, Name, State}` and `EnvironmentReset{Cleared []EnvironmentObject}`
  identically — same field names, same order, same `json` tags
  (`kind`/`name`/`state`, `cleared`). PASS.
- Producer: `services/atlas-maps/atlas.com/maps/map/environment/producer.go:33`
  now sets `State: e.DefaultState` (was previously omitted, so it was `0` by
  Go zero-value). `producer_test.go` asserts `State: 1` for an entry with
  `DefaultState: 1` distinct from its cleared-from `State: 2` — a real
  assertion that the emitted field is `DefaultState`, not `State`. PASS.
- Consumer: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:1297-1305`
  reads `o.State` per cleared entry and only zeroes it when
  `kind == field.ObjectKindObstacle` (consumer.go:1302-1304). This is exactly
  the fix described in the brief. PASS.
- Test honesty: `consumer_test.go`'s `TestHandleStatusEventEnvironmentReset_RestoresCarriedDefaultState`
  (added in `b3706768d`) decodes the actual `SetObjectState` wire body via a
  new `stubDoorAnnounceForObjectState` helper (captures `packet.Encode`
  output and decodes it with `fieldcb.SetObjectState.Decode`), then asserts
  `gate -> State 1` (from `EnvironmentObject{Name:"gate", State:1}`) and
  `obs3 -> State 0` despite the wire event carrying `State:7` for the
  obstacle. This test fails against the pre-fix hardcoded-`0` behavior and
  passes only against the fix. PASS — genuine regression test, not shape-only.
  (Note: the two pre-existing `_AllResetRouted` / `_AllResetUnrouted` /
  `_EmptyCleared` tests in the same file only assert writer *names*, via the
  older `stubDoorAnnounceCapture` that discards the encoded body, and their
  fixtures never set a non-zero `State` — so those three would pass unchanged
  whether the code used `o.State` or a hardcoded `0`. That's fine: they are
  pinning writer *selection*, and the new test is the one that actually pins
  the state value, per its own doc comment. Not a defect, just noting the
  division of responsibility so it isn't mistaken for the regression test.)

## 2. atlas-maps default-resolution path

`services/atlas-maps/atlas.com/maps/map/environment/processor.go`:
- `Set` (line 39-51): resolves `DefaultState` from the registry if the object
  is already tracked (`DefaultState` returns `(uint32, bool)` from
  `registry.go:63-73`), else calls `defaultStateOf`, which:
  - returns `0` without calling atlas-data for `ObjectKindObstacle` (`FieldObstacleAllReset` semantics — correct, matches the "intentionally preserved" carve-out in the brief).
  - calls `object.Processor.GetDefaultState(mapId, name)`; on any error
    (including atlas-data unreachable, or `ErrUnknownObject` when the map
    declares no such object) logs a `Warn` and falls back to `0`, never
    propagating the error to the caller. Comment states the rationale
    explicitly: "a reset must never fail." Reasonable and matches existing
    fail-open conventions elsewhere in the codebase (e.g.
    `announceActiveJukebox` in atlas-channel). PASS.
  - A map with no `obj` node at all: `object.Processor.inMapProvider` drains
    atlas-data's `/objects` collection, which returns an empty array (see
    `getObjects` fallback below), so `GetDefaultState` returns
    `ErrUnknownObject`, which is handled by the same fallback-to-0 path. PASS.
- Covered by five new tests in `processor_default_test.go`, one per path
  (declared default, cached-not-refetched, undeclared object, atlas-data
  unreachable, obstacle skips atlas-data entirely). All exercise the real
  HTTP round trip via `httptest.NewServer` + `DATA_SERVICE_URL`, not a
  processor mock. PASS.

## 3. atlas-data `obj` node reader

`services/atlas-data/atlas.com/data/map/reader.go` `getObjects`/`objectState`:
- Scans every numbered layer's `obj` child (`obj` entries are distributed
  across layers) rather than assuming a single location — matches the
  documented WZ shape. A layer without an `obj` child is skipped via the
  `ChildByName` error return, not a panic. A map with no numbered layers at
  all yields an empty `[]object.RestModel{}` (initialized via
  `make(..., 0)`), never `nil` from a nil-map crash. PASS — `TestGetObjectsWithoutObjNode`
  confirms this using the pre-existing `reactorTestXML` fixture (no `obj`
  node), asserting `len(os) == 0`.
- Entries with no `name` attribute are skipped (`if name == "" { continue }`)
  — matches the doc comment "cannot be addressed by `CField::OnSetObjectState`".
- `objectState` parses `l2` as `uint64` via `strconv.ParseUint`; empty or
  non-numeric values fall back to `0` with a `Debug` log rather than
  failing the whole map read. `TestGetObjects` exercises all three cases
  (numeric `l2=1`, non-numeric `l2=on`, absent `l2`) against a literal
  excerpt of map `103000800`'s layer-4 `gate`/`barricade`/`lever` objects,
  and layer-0's unnamed entry is confirmed skipped. This is the test that
  pins the ground-truth polarity claim (`gate` `l2=1`) at the source. PASS —
  no panic path found; no mis-parse beyond the documented "unparseable ->
  0" fallback.

## 4. New REST surface (atlas-data)

`map/object/rest.go`, `map/resource.go`'s `handleGetMapObjectsRequest`,
`map/processor.go`'s `GetObjects`/`objectProvider`, and the `docs/rest.md`
addition all mirror the existing `reactors` sub-resource byte-for-byte in
structure (pagination via `paginate.ParseParams`/`paginate.Slice`,
`server.MarshalPaginatedResponse`, `404` on map-not-found, resource id ==
object name, `GetReferences`/`GetReferencedIDs`/`GetReferencedStructs`/
`SetToManyReferenceIDs`/`SetReferencedStructs` wiring in the parent
`RestModel` in `rest.go`). Doc path convention (`/api/data/maps/...`)
matches every other endpoint in the same file. `resource_object_test.go`
performs a genuine end-to-end round trip (`s.Add` -> HTTP GET -> JSON:API
decode) and asserts the actual decoded `id`/`name`/`state`, not just
document shape. PASS.

`mock/processor.go`'s `GetObjectsFunc` addition matches the existing mock
pattern for every other `Get*` method (nil-safe default, override hook).
PASS.

## 5. Ingress route (`e050686d4`)

`deploy/shared/routes.conf` (and its `deploy/k8s/base/routes.conf.template.generated`
mirror) insert the new
`^/api/worlds/[^/]+/channels/[^/]+/maps/[^/]+/instances/[^/]+/environment(/.*)?$`
rule at line 505, routing to atlas-maps. The catch-all
`^/api/worlds(/.*)?$` -> atlas-world sits at line 696, i.e. *after* the new
rule in the same file. All `location` blocks in this file use plain `~`
(no `^~`, no priority modifiers observed in either file), so nginx evaluates
them in file order and the first regex match wins — confirmed by reading the
surrounding blocks (`weather`, `jukebox`, `monsters`, `summons` all use the
same unprefixed `~` form). The new rule therefore genuinely outranks the
catch-all it is meant to escape, not merely coexists with it. `deploy/compose/routes.conf`
is a symlink to `deploy/shared/routes.conf`, so no separate update was
needed there. PASS.

## 6. Test quality (overall)

All five new/changed test files decode real values (wire bytes for the
channel consumer's new test, JSON:API attributes for atlas-data's endpoint
test, parsed WZ properties for the reader test, and processor-level
`DefaultState`/call-count assertions for the atlas-maps fallback paths) —
none of the *new* assertions are shape-only. The one caveat noted in §1
(three pre-existing consumer tests remain writer-name-only) predates this
fix and is not a regression introduced by it.

## Build/test verification (module-local, not the full gate)

- `services/atlas-maps/atlas.com/maps`: `go build ./...` clean;
  `go test ./map/environment/... ./data/map/object/... ./kafka/message/map/...` — all PASS.
- `services/atlas-data/atlas.com/data`: `go build ./...` clean;
  `go test ./map/...` — PASS.
- `services/atlas-channel/atlas.com/channel`: `go build ./...` clean;
  `go test ./kafka/consumer/map/...` — PASS.

(Flagless `tools/verify.sh` was not re-run per instructions; it already
passed at `5efddd01b`.)

## Not evaluable

- The WZ ground-truth polarity itself (`Obj/effect.img/quest/gate/0` visible
  state `0` vs. declared `l2=1`) was explicitly excluded from re-derivation
  and taken as given per the task brief.
- Live-client verification of the actual PQ gate scenario end-to-end (i.e.
  running a real channel against a real client) is outside what a diff
  review can exercise; the review relies on the decoded-wire-body unit test
  as the strongest available evidence.

## Findings

None blocking. No non-blocking findings beyond the informational note in §1
about the division of labor between the writer-name tests and the new
state-value test (not a defect).

## Verdict

APPROVED. All four commits do what the brief describes, the seam agrees
field-for-field, the fallback paths in the default-resolution chain are
sane and tested, the WZ reader cannot panic and degrades gracefully, the new
REST surface follows existing convention, the ingress rule genuinely
precedes the catch-all it escapes, and the regression is pinned by a test
that decodes the real wire value and fails against the pre-fix code.
