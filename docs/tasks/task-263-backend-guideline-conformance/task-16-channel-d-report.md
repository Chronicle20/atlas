# Task 16 batch `channel-d` report

Six `atlas-channel` packages (tier B2): `mts/configuration`, `parcel`, `pet`, `quest`, `reactor`, `trade`.
Each gets a hand-written `Transform` (RestModel <- Model) in `rest.go`, plus a new `rest_test.go` with
`TestTransformRoundTrip`.

## Implementation

For each package, `Transform` reads the `Model`'s exported accessors and builds the corresponding
`RestModel`, mirroring the existing `Extract`'s field mapping in reverse — same pattern as
`services/atlas-ban/atlas.com/ban/ban/rest.go:36-46` and `services/atlas-channel/atlas.com/channel/door/rest.go:52-75`.

- `mts/configuration`: `Extract`/`Transform` have no `error` return (matches the existing `Extract`
  signature — `func Extract(r RestModel) Model`). Straightforward field-for-field mapping; all 11
  economic knobs.
- `parcel`: `Transform(m Model) (RestModel, error)`, `Id` emitted via `m.Id().String()`. Full field
  parity, including the two `*time.Time`/`*uint32` optional pointers (`ItemId`, `LastNotified`).
- `pet`: `Transform` converts `m.Excludes()` via the already-Transform'd `exclude.Transform` (a
  prior batch landed `exclude.Transform`, `pet/exclude/rest.go:28-33`) — no `atlas-model` SliceMap
  needed since `Transform` doesn't require the parallelism `Extract` uses. `Lead` is deliberately
  not populated (see handwork-notes.md below).
- `quest`: `Transform` converts `m.Progress()` into `[]ProgressRestModel`, mapping only
  `InfoNumber`/`Progress` (matching what `Extract` reads back — `ProgressRestModel.Id` is never
  read by `Extract` either, so nothing round-trips through it and nothing needed adding).
- `reactor`: full field mapping except `UpdateTime`, which is genuinely lossy — see below.
- `trade`: single-field `Transform`, `Id` via `m.Id().String()`.

## Lossy / non-emitted fields (resolution #3 and #4)

Recorded in `docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md` under the new
`## Batch channel-d (Task 16, tier B2)` heading (uncommitted — per the brief, that file is the
controller's to land):

- **`pet`** (resolution #3, not lossy): `RestModel.Lead` (`rest.go:21`) is never read by `Extract`
  (`rest.go:49-74`); `Model` has no `lead` field — `Model.Lead()` (`model.go:77-79`) derives
  `slot == 0` instead. `Transform` does not populate `Lead`. `TestTransformRoundTrip` still asserts
  a full `reflect.DeepEqual` over `Model` — nothing is dropped from `Model`, so the full assertion
  holds.
- **`reactor`** (resolution #4, genuinely lossy): `Model.updateTime` (`model.go:25`, set to
  `time.Now()` by `NewBuilder`, `builder.go:26-33`) has no corresponding `RestModel` field, so
  `Extract` can never restore it. `TestTransformRoundTrip` builds `Model` via the package's builder
  (not via `Extract`, so `updateTime` starts genuinely non-zero), asserts field-by-field over every
  field `RestModel` carries, explicitly names `UpdateTime` as omitted from the comparison, and
  additionally asserts it is dropped to the zero value (not invented) after the round trip.

No `RestModel` field was added to close either gap (PRD §5).

## TDD Evidence

RED — `go test ./mts/configuration/... ./parcel/... ./pet/... ./quest/... ./reactor/... ./trade/... -run TestTransformRoundTrip -v` (before `Transform` existed):

```
# atlas-channel/quest [atlas-channel/quest.test]
quest/rest_test.go:34:14: undefined: Transform
# atlas-channel/pet [atlas-channel/pet.test]
pet/rest_test.go:47:14: undefined: Transform
# atlas-channel/mts/configuration [atlas-channel/mts/configuration.test]
mts/configuration/rest_test.go:30:9: undefined: Transform
FAIL	atlas-channel/mts/configuration [build failed]
# atlas-channel/trade [atlas-channel/trade.test]
trade/rest_test.go:23:14: undefined: Transform
# atlas-channel/parcel [atlas-channel/parcel.test]
parcel/rest_test.go:45:14: undefined: Transform
FAIL	atlas-channel/parcel [build failed]
FAIL	atlas-channel/pet [build failed]
FAIL	atlas-channel/quest [build failed]
# atlas-channel/reactor [atlas-channel/reactor.test]
reactor/rest_test.go:36:13: undefined: Transform
FAIL	atlas-channel/reactor [build failed]
FAIL	atlas-channel/trade [build failed]
FAIL
```

Failure was expected: `Transform` did not exist yet in any of the 6 packages.

GREEN — after implementing all 6 `Transform` functions:

```
=== RUN   TestTransformRoundTrip
--- PASS: TestTransformRoundTrip (0.00s)
PASS
ok  	atlas-channel/mts/configuration	0.046s
=== RUN   TestTransformRoundTrip
--- PASS: TestTransformRoundTrip (0.00s)
PASS
ok  	atlas-channel/parcel	0.023s
=== RUN   TestTransformRoundTrip
--- PASS: TestTransformRoundTrip (0.00s)
PASS
ok  	atlas-channel/pet	0.034s
=== RUN   TestTransformRoundTrip
--- PASS: TestTransformRoundTrip (0.00s)
PASS
ok  	atlas-channel/quest	0.045s
=== RUN   TestTransformRoundTrip
--- PASS: TestTransformRoundTrip (0.00s)
PASS
ok  	atlas-channel/reactor	0.036s
=== RUN   TestTransformRoundTrip
--- PASS: TestTransformRoundTrip (0.00s)
PASS
ok  	atlas-channel/trade	0.050s
```

## Mutation proof of non-tautology

Broke `reactor`'s `Transform` (`Delay: m.Delay() + 1`) and re-ran:

```
=== RUN   TestTransformRoundTrip
    rest_test.go:74: Delay mismatch. Expected 9, got 10
--- FAIL: TestTransformRoundTrip (0.00s)
FAIL
FAIL	atlas-channel/reactor	0.008s
```

Reverted the mutation (`sed` back to `Delay: m.Delay()`), confirmed clean diff against the committed
`rest.go`, and re-ran to GREEN:

```
=== RUN   TestTransformRoundTrip
--- PASS: TestTransformRoundTrip (0.00s)
PASS
ok  	atlas-channel/reactor	(cached)
```

## Module-local gate

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test ./...
```

Both `go build` and `go vet` were clean (no output). `go test ./...` — all packages `ok` or
`[no test files]`, no failures (full output inspected; the 6 touched packages: `ok atlas-channel/mts/configuration`,
`ok atlas-channel/parcel`, `ok atlas-channel/pet`, `ok atlas-channel/quest`, `ok atlas-channel/reactor`,
`ok atlas-channel/trade`).

## Files changed

- `services/atlas-channel/atlas.com/channel/mts/configuration/rest.go` — added `Transform`
- `services/atlas-channel/atlas.com/channel/mts/configuration/rest_test.go` — new, `TestTransformRoundTrip`
- `services/atlas-channel/atlas.com/channel/parcel/rest.go` — added `Transform`
- `services/atlas-channel/atlas.com/channel/parcel/rest_test.go` — new, `TestTransformRoundTrip`
- `services/atlas-channel/atlas.com/channel/pet/rest.go` — added `Transform`
- `services/atlas-channel/atlas.com/channel/pet/rest_test.go` — new, `TestTransformRoundTrip`
- `services/atlas-channel/atlas.com/channel/quest/rest.go` — added `Transform`
- `services/atlas-channel/atlas.com/channel/quest/rest_test.go` — new, `TestTransformRoundTrip`
- `services/atlas-channel/atlas.com/channel/reactor/rest.go` — added `Transform`
- `services/atlas-channel/atlas.com/channel/reactor/rest_test.go` — new, `TestTransformRoundTrip`
- `services/atlas-channel/atlas.com/channel/trade/rest.go` — added `Transform`
- `services/atlas-channel/atlas.com/channel/trade/rest_test.go` — new, `TestTransformRoundTrip`
- `docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md` — appended `## Batch channel-d`
  section (left **uncommitted**, per brief — the controller's file to land)

Commit: `d4731b0` — "feat(atlas-channel): add Transform and round-trip tests for mts/configuration,
parcel, pet, quest, reactor, trade" — touches only `services/atlas-channel`.

## Self-review

- Every `Transform` reads only exported `Model` accessors (no unexported-field reach-across, matching
  D1's inverse-direction constraint since `rest.go` is in-package with `Model`).
- No `Build()` validation rule was touched (reactor's `NewBuilder(...).Build()` still requires
  `id != 0`; the test fixture uses `SetId(400)`).
- No `RestModel` field was added anywhere to close a gap; the two carve-outs (`pet.Lead`,
  `reactor.UpdateTime`) are both pre-existing structural facts, verified at the source
  (`Extract`'s body / `Model`'s field list), not test-comfort narrowing.
- Fixtures use distinct, non-zero values throughout; `parcel`/`trade`/`reactor` use fresh
  `uuid.New()` values; sign/pair fields (X/Y, AreaX/AreaY-equivalents where present) are
  distinguishable.
- `quest`'s `Progress`/`ProgressRestModel.Id` mismatch was checked against `Extract`'s body — `Id`
  is genuinely unread there too, so nothing needed adding to `Transform`'s progress mapping either;
  not called out separately in handwork-notes since it's the same non-lossy resolution-#3 shape as
  `pet.Lead` and doesn't affect the full `Model` `reflect.DeepEqual`.

## Concerns

None. All 6 packages build, vet, and test clean; RED/GREEN/mutation evidence captured above.
