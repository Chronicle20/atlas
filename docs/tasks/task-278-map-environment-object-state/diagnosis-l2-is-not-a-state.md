# task-278 — diagnosis: WZ `l2` is a path component, not a declared default state

Date: 2026-09-03. Field: pr-1566 (`pr-1566-aa9548d`), tenant
`d1defcd0-9994-4ee2-bcad-9eaaa1377839`, GMS 83.1, map 103000800, object `gate`.

## Summary

The follow-up fix (`da20524d3` + `b3706768d`) rests on a false premise. A map
`obj` entry's `l2` property is **not** the object's declared default state — it
is the third component of the WZ resource path. `CMapLoadable::MakeObj`
(v83 `0x63ad16`) fetches `oS` / `l0` / `l1` / `l2` (StringPool `SP_1496_OS`,
`SP_1497_L0`, `SP_1498_L1`, `SP_1499_L2`) from the map entry, formats them into
one path string, and hands it to `IWzResMan::GetObjectA`. For map 103000800's
gate (`oS=effect`, `l0=quest`, `l1=gate`, `l2=1`) the object is
`Obj/effect.img/quest/gate/1`. The `1` names the graphic, not a state.

So `atlas-data`'s `getObjects` exports `state: 1` for the gate, atlas-maps
carries it as the object's default, and the reset announces
`SetObjectState("gate", 1)`.

## Why the client ignores it

`CMapLoadable::SetObjectState` (v83 `0x642aca`, reached from
`CField::OnSetObjectState` `0x537a1e`):

1. look up the name in `m_mNamedObj`; miss ⇒ `return 0`;
2. `if (stateCount <= state) return 0;` — **silent no-op**;
3. otherwise alpha 0 on the current state's layer, store the new index,
   alpha 255 on the new layer, restart the animation, play the state's sound.

In `MakeObj`'s named-object loop, state 0 is the object node itself and state N
is its numbered child `"N"`; the loop stops at the first missing child, and
**every** state layer is created with alpha 0. The gate has only state 0, so
states 1 and 2 both fail the bounds check at step 2.

Live confirmation, same session, all three through the identical announce path:

| announced | channel log | client |
|---|---|---|
| `state 0` | `set to state [0]` @ 15:26:40.550Z | gate becomes visible |
| `state 1` (reset, `cleared[].state=1`) | `restoring [1] tracked object(s)` @ 15:27:06.632Z | **unchanged, still visible** |
| `state 1` (direct) | `set to state [1]` @ 15:28:18.403Z | **unchanged, still visible** |
| `state 2` (direct) | `set to state [2]` @ 15:31:09.754Z | **unchanged, still visible** |

The reset handler is therefore not at fault — the same packet via the
state-change path is equally inert. The bug is the value, and the concept
behind the value.

## The deeper problem

For a named object there is no state meaning "hidden". Layers start at alpha 0
and the first `SetObjectState` raises one; nothing lowers all of them again.
`SetObjectState(name, -1)` re-shows the current state rather than clearing it.
`CMapLoadable::OnSetMapObjectVisible` (`0x6449d2`) *can* hide, but it keys
`m_mTagedObj` — the WZ `tags` keyspace — not `m_mNamedObj`, so it cannot
address `gate` by name.

Net: "restore the object's declared default on reset" is not expressible for
ENVIRONMENT objects through `SetObjectState` as currently designed. Restoring
`0` (the pre-`b3706768d` behaviour) was wrong for a different reason; restoring
`l2` is wrong because `l2` is not a state and, for this object, is out of range.

## Blast radius

- `services/atlas-data/.../map/reader.go` `getObjects` / `objectState` — the
  `l2`-as-state parse, and the `object.RestModel.State` field it feeds.
- `EnvironmentObject.State` in atlas-maps and the `cleared[].state` field on
  `EnvironmentReset`.
- `handleStatusEventEnvironmentReset` in atlas-channel and
  `TestHandleStatusEventEnvironmentReset_RestoresCarriedDefaultState`, which
  asserts the wrong contract (it pins `want [{SetObjectState gate 1} ...]`).

Not affected: the ENVIRONMENT_STATE_CHANGED path, the ingress route fix
(`e050686d4`), reset-on-empty, and replay-on-enter — all still verified.

## Note on the environment

pr-1566's atlas-data documents were seeded 2026-08-28 and predated the `obj`
parser, so `GET /data/maps/{id}/objects` returned `[]` until a tenant-scoped
ingest was run on 2026-09-03 13:46–13:56Z and atlas-data was restarted to drop
its in-memory registry cache. Any future live test of this path needs both
steps, or it silently tests the old data.

---

# Fix

**Decision (user-approved 2026-09-03):** a reset announces **nothing** for named
(non-obstacle) objects. Obstacles keep their existing behaviour. Delete the
`l2`-as-default-state plumbing end to end.

Rationale, so the implementer does not re-litigate it: a named object's
load-time state is hidden (all state layers are created at alpha 0; only an
`autoActive` object is revealed at load), and no packet returns it to hidden.
Clearing tracking is therefore already the correct reset — a player who enters
the reset field loads it hidden and receives no override. A player standing in
the field at reset time keeps the stale visual until re-entry; that is a client
limitation, not something to work around.

## Files

Revert `da20524d3` in full, plus the parts of `6c9acd433` / `5efddd01b` /
`0054591a5` that touch the same surface:

- `services/atlas-data/atlas.com/data/map/object/` — delete the package.
- `services/atlas-data/atlas.com/data/map/reader.go` — drop `getObjects`,
  `objectState`, and the `m.Objects = ...` assignment.
- `services/atlas-data/atlas.com/data/map/rest.go` — drop the `Objects` field
  and its `objects` entries in `GetReferencedIDs` / `GetReferencedStructs` /
  `SetToOneReferenceID`-side wiring.
- `services/atlas-data/atlas.com/data/map/processor.go`, `mock/processor.go` —
  drop `GetObjects` / `objectProvider`.
- `services/atlas-data/atlas.com/data/map/resource.go` — drop the
  `/{mapId}/objects` route and `handleGetMapObjectsRequest`.
- `services/atlas-data/atlas.com/data/map/reader_object_test.go`,
  `resource_object_test.go` — delete.
- `services/atlas-data/docs/rest.md` — drop the objects endpoint section.

Then unwind the seam:

- `services/atlas-maps/atlas.com/maps/data/map/object/` — delete the package
  (`builder.go`, `model.go`, `processor.go`, `processor_test.go`,
  `requests.go`, `rest.go`).
- `services/atlas-maps/atlas.com/maps/map/environment/registry.go` — drop
  `ObjectEntry.DefaultState` and the `Registry.DefaultState` method; fix the
  `ObjectEntry` doc comment.
- `services/atlas-maps/atlas.com/maps/map/environment/processor.go` — drop
  `defaultStateOf`, the `op object.Processor` field, and the `atlas-maps/data/map/object`
  import; `Set` builds `ObjectEntry{Kind, Name, State}`.
- `services/atlas-maps/atlas.com/maps/map/environment/producer.go:36` — emit
  `EnvironmentObject{Kind, Name}` only.
- `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go` and
  `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go` — revert
  `EnvironmentObject` to `{Kind, Name}` (both copies must stay identical) and
  restore the `EnvironmentReset` doc comment to describe "which objects were
  cleared", not "and to what state".
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`
  `handleStatusEventEnvironmentReset` — keep `FieldObstacleAllReset`; in the
  `Cleared` loop announce `SetObjectState(name, 0)` **only** when
  `kind == field.ObjectKindObstacle`, and skip every other kind. Comment must
  say why: no state hides a named object, so the reset cannot restore one.

## Tests that must change

- `services/atlas-channel/.../consumer_test.go`
  `TestHandleStatusEventEnvironmentReset_RestoresCarriedDefaultState` — currently
  pins `want [{SetObjectState gate 1} {SetObjectState obs3 0}]`, which is the
  wrong contract. Rewrite it (rename accordingly) to assert that a cleared
  ENVIRONMENT object produces **no** `SetObjectState`, and a cleared OBSTACLE
  still produces state 0. Keep decoding the real `fieldcb.SetObjectState` wire
  body rather than asserting on writer names.
- `services/atlas-maps/.../environment/processor_default_test.go` — delete or
  rewrite; every assertion is about `DefaultState`.
- `services/atlas-maps/.../environment/producer_test.go:50` — drop the
  `DefaultState` field from the fixture and the `state` expectation.

## Docs

- `docs/tasks/task-278-map-environment-object-state/design.md` §1.2 — the claim
  that non-obstacle objects must be restored explicitly is what this diagnosis
  disproves. Correct it in place and point at this file.
- `docs/tasks/task-278-map-environment-object-state/progress.md` — record the
  revert and supersede the "Remaining work → live re-test of reset" item.
- Note the `autoActive` follow-up: an object whose `Obj` node carries
  `autoActive` *is* visible at load and its correct reset would be
  `SetObjectState(name, 0)`. Reading that flag means resolving into the `Obj`
  tree, not the map's `obj` entry. No object in play needs it; document, do not
  build.

## Not to be touched

`e050686d4` (the ingress route for `.../environment`) is an independently
verified fix and stays. The ENVIRONMENT_STATE_CHANGED path, reset-on-empty, and
replay-on-enter were all live-verified and must keep working.
