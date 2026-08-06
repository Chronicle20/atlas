# task-193: minor-fixes report (two non-blocking review findings)

Scope: test-only changes in `services/atlas-transports/atlas.com/transports/instance/`.
No production code was changed. `rest.go` is byte-identical to its pre-existing
committed state (verified with `git diff`, see below).

## Finding 1 — direct test for `TransformRoute` (layer 4)

Added `services/atlas-transports/atlas.com/transports/instance/rest_test.go`
(new file — no prior `rest_test.go` existed for this package) with three tests:

- `TestTransformRoute_MapsEffectFields` — builds a `RouteModel` via
  `NewRouteBuilder` with non-zero `EffectItemIds`/`ForcedReturnMapId`, calls
  `TransformRoute`, asserts both fields survive the projection.
- `TestTransformRoute_WithoutEffectFieldsProjectsCleanly` — a route declaring
  neither field projects to empty/zero, not garbage.
- `TestTransformRoute_EffectItemIdsIsNotAliased` — mutates the returned REST
  model's `EffectItemIds` slice and asserts the source `RouteModel`'s own
  accessor is unaffected, mirroring
  `TestRouteModel_EffectItemIdsIsDefensiveCopy` in `model_json_test.go`.

### Regression proof

Temporarily edited `rest.go`'s `TransformRoute` to comment out the
`EffectItemIds`/`ForcedReturnMapId` field mappings, then ran:

```
$ go test -run 'TestTransformRoute' -v ./instance/...
=== RUN   TestTransformRoute_MapsEffectFields
    rest_test.go:33:
        	Error Trace:	.../instance/rest_test.go:33
        	Error:      	Not equal:
        	            	expected: []item.Id{0x21b8e0}
        	            	actual  : []item.Id(nil)
        	Test:       	TestTransformRoute_MapsEffectFields
    rest_test.go:34:
        	Error Trace:	.../instance/rest_test.go:34
        	Error:      	Not equal:
        	            	expected: 0xe4e1c6e
        	            	actual  : 0x0
        	Test:       	TestTransformRoute_MapsEffectFields
--- FAIL: TestTransformRoute_MapsEffectFields (0.00s)
=== RUN   TestTransformRoute_WithoutEffectFieldsProjectsCleanly
--- PASS: TestTransformRoute_WithoutEffectFieldsProjectsCleanly (0.00s)
=== RUN   TestTransformRoute_EffectItemIdsIsNotAliased
--- FAIL: TestTransformRoute_EffectItemIdsIsNotAliased (0.00s)
panic: runtime error: index out of range [0] with length 0 [recovered, repanicked]
...
FAIL	atlas-transports/instance	0.010s
```

Two of the three new tests fail (one as a hard `assert.Equal` failure, one as
an index-out-of-range panic because the returned slice is now empty) — the
regression is caught. `rest.go` was then restored to its exact original
content; `git diff -- .../instance/rest.go` produces no output, confirming
byte-identity.

## Finding 2 — DOM-20 table-driven conversions (scoped)

Converted only clusters that are genuinely repeated input/output pairs over
the same call shape. Left everything else standalone, per the audit's own
finding that most new tests exercise materially different setup.

### Converted

| File | New table func | Subsumes | Reason it fits |
|---|---|---|---|
| `instance/builder_test.go` | `TestRouteBuilder_EffectFields` | `TestRouteBuilder_RejectsZeroEffectItemId`, `TestRouteBuilder_ZeroForcedReturnMapIdIsNotAnError`, `TestRouteBuilder_EffectFieldsAreOptional` | Same base builder setup (`test`, `[]_map.Id{100}`, capacity 6, 10s/30s), only the optional effect-field wiring and the error/success outcome differ — the audit's own named "strongest candidate". |
| `instance/producer_test.go` | `TestConsumableEffectProviders_WireShape` | `TestApplyConsumableEffectProvider_WireShape`, `TestCancelConsumableEffectProvider_WireShape` | Identical function signature (`func(world.Id, channel.Id, uint32, item.Id) model.Provider[[]kafka.Message]`) and identical decode/assert body; only the provider constructor and expected `Type` differ. Verified from `producer.go:121-149` that both providers set `TransactionId: uuid.Nil`, `WorldId`, `ChannelId` identically, so folding Apply's fuller assertion set onto Cancel's subtable case is not adding an unverified assumption. |
| `instance/config/rest_test.go` | `TestExtractRouteFor_EffectAttributes` | `TestExtractRouteFor_ThreadsEffectAttributes`, `TestExtractRouteFor_EffectAttributesAreOptional` | Same call shape (`config.ExtractRouteFor(quietLogger(), tm)(route)` then assert `EffectItemIds`/`ForcedReturnMapId`), differing only in the fixture's declared attributes and expected values. |

Every assertion from each original standalone test is preserved in its table
case (verified by inspection and by running `-v` with subtests, all listed
scenario names present and passing — see Verification below). Subtest names
match the original scenario names exactly (`t.Run("RejectsZeroEffectItemId", ...)`,
etc.), so the described scenario is not lost, only the boilerplate is shared.

### Left standalone (with reason)

- `instance/model_json_test.go` (`TestRouteModel_JSONRoundTripPreservesEffectFields`,
  `TestRouteModel_JSONRoundTripWithoutEffectFields`,
  `TestRouteModel_EffectItemIdsIsDefensiveCopy`) — the two round-trip tests
  build materially different route topologies (an 8-field seeded flight vs. a
  bare-minimum route) with different assertion sets, and the third test is a
  different mechanic entirely (direct slice mutation, no marshal/unmarshal).
  Forcing these into one table would blur exactly the setup differences the
  audit flagged as a weak fit. Left standalone.
- `instance/processor_test.go` (all 17 new tests) — each exercises a
  different processor method (`StartTransport`, `HandleMapEnter`,
  `HandleLogout`, `TickStuckTimeout`, `GracefulShutdown`,
  `forceCancelInstance`, `completeInstance`), a different route topology
  (`newEffectRoute` vs `newPlainRoute` vs a bespoke builder chain for one
  case), different character counts, and different assertion shapes (some
  assert `Error`, most assert varying combinations of consumables/warps/events
  counts and fields). This is exactly the "route topology, character counts,
  registry state" pattern the audit already characterized as a weak fit for
  tabling; converting it would produce a table with mostly single-use fields
  and per-case conditional assertion logic, which obscures more than it
  shares. Left standalone.
- `instance/producer_test.go`'s `TestConsumableProviders_ShareOneKeyPerCharacter`
  — different shape from the wire-shape pair: it calls both providers
  together and compares their `Key` values against each other, not a
  per-provider input/output assertion. Left standalone.

## Verification

From `services/atlas-transports/atlas.com/transports`:

```
$ go build ./...
(no output, exit 0)

$ go vet ./...
(no output, exit 0)

$ go test -race ./...
ok  	atlas-transports	(cached)
ok  	atlas-transports/channel	(cached)
ok  	atlas-transports/data/portal	(cached)
ok  	atlas-transports/instance	(cached)
ok  	atlas-transports/instance/config	(cached)
ok  	atlas-transports/kafka/consumer/channel	(cached)
ok  	atlas-transports/kafka/consumer/character	(cached)
ok  	atlas-transports/kafka/consumer/configuration	(cached)
ok  	atlas-transports/map	(cached)
ok  	atlas-transports/tenant	(cached)
ok  	atlas-transports/transport	(cached)
ok  	atlas-transports/transport/config	(cached)
(all other packages: [no test files])
```

```
$ gofmt -l instance/builder_test.go instance/rest_test.go instance/producer_test.go instance/config/rest_test.go instance/rest.go
(no output — all clean)
```

Subtest run confirming every converted scenario name is present and passing:

```
$ go test -run 'TestRouteBuilder_EffectFields|TestTransformRoute|TestConsumableEffectProviders_WireShape' -v ./instance/...
--- PASS: TestRouteBuilder_EffectFields (0.00s)
    --- PASS: TestRouteBuilder_EffectFields/RejectsZeroEffectItemId (0.00s)
    --- PASS: TestRouteBuilder_EffectFields/ZeroForcedReturnMapIdIsNotAnError (0.00s)
    --- PASS: TestRouteBuilder_EffectFields/EffectFieldsAreOptional (0.00s)
--- PASS: TestConsumableEffectProviders_WireShape (0.00s)
    --- PASS: TestConsumableEffectProviders_WireShape/ApplyConsumableEffectProvider_WireShape (0.00s)
    --- PASS: TestConsumableEffectProviders_WireShape/CancelConsumableEffectProvider_WireShape (0.00s)
--- PASS: TestTransformRoute_MapsEffectFields (0.00s)
--- PASS: TestTransformRoute_WithoutEffectFieldsProjectsCleanly (0.00s)
--- PASS: TestTransformRoute_EffectItemIdsIsNotAliased (0.00s)

$ go test -run 'TestExtractRouteFor_EffectAttributes' -v ./instance/config/...
--- PASS: TestExtractRouteFor_EffectAttributes (0.00s)
    --- PASS: TestExtractRouteFor_EffectAttributes/ThreadsEffectAttributes (0.00s)
    --- PASS: TestExtractRouteFor_EffectAttributes/EffectAttributesAreOptional (0.00s)
```

`git diff -- services/atlas-transports/atlas.com/transports/instance/rest.go`
produces no output — `rest.go` is unmodified.
