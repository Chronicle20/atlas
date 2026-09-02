# Follow-up brief: environment reset must restore each object's declared default

Seed document for a new `/spec-task`. Not implementable inside task-278's
worktree: it crosses atlas-data, atlas-maps and atlas-channel and changes a
Kafka event contract, so it needs its own task number and the four-phase flow.

## The defect, confirmed in-client

pr-1566, tenant `d1defcd0-9994-4ee2-bcad-9eaaa1377839`, map 103000800 (Kerning
PQ stage 1), object `gate`. After `DELETE .../environment` the gate remained
**visible** instead of returning to the map's initial appearance. Channel logged
`Environment reset in map [103000800] ...; restoring [1] tracked object(s)` at
2026-09-02T18:23:01.998Z.

For this object the visible state is `0` (observed in-client; do not re-derive
from the WZ tree) and the map declares `l2=1`. Reset announces `0`, so a PQ
stage resets into its *completed* look.

## Why it happens

`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`
(`handleStatusEventEnvironmentReset`) hardcodes the restore state:

```go
// Explicitly zero every tracked object
_ = announceObjectState(l, ctx, wp, kind, o.Name, 0, s)
```

`0` is correct only for `ObjectKindObstacle`, where the client's
`FieldObstacleAllReset` genuinely means "all off". It is not a safe default for
named environment objects, whose rest state is whatever `l2` the map declares.

## Why it is not a one-line fix

The declared default is not available to any service:

- `services/atlas-maps/atlas.com/maps/map/environment/registry.go:16`
  `ObjectEntry{Kind, Name, State}` tracks only the *current* state. atlas-maps
  first learns an object exists when something sets it, so it never observed the
  pre-change value.
- `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go:76`
  `EnvironmentObject{Kind, Name}` has no state field at all, so the reset event
  cannot currently carry an original state even if one were known.
- `services/atlas-data/atlas.com/data/map/reader.go` does not parse the map's
  `obj` node — its only match is `shipObj` at :319. No service exposes an
  object's `l2`.

## Fix shape (for design to confirm, not prescriptive)

1. **atlas-data** — parse map `obj` entries and expose them on the map REST
   surface, at minimum `name` and `l2` (the declared default state). Entries
   without a `name` property are not addressable by `SetObjectState` and can be
   skipped.
2. **atlas-maps** — when first tracking an object, resolve its declared default
   from atlas-data and retain it on `ObjectEntry`; add a state field to
   `EnvironmentObject` so `EnvironmentReset.Cleared` carries the value to
   restore.
3. **atlas-channel** — announce the carried per-object state instead of the
   hardcoded `0`. Preserve the existing `FieldObstacleAllReset` behaviour for
   `ObjectKindObstacle`.

## Contract change

`EnvironmentReset.Cleared[]` gains a state field. atlas-channel is the only
consumer (`consumer.go:134`). A test must assert the new contract — a green
`verify.sh` cannot see this seam.

## Verification

Reproduce the manual test in `manual-test-notes.md`: set `gate` to a non-default
state, confirm visually, `DELETE`, and confirm the gate returns to the map's
initial appearance rather than staying in the set state. Requires a live client;
the ingress route from `e050686d4` must be deployed.
