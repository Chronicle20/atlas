# Review — bug report §5 fix (`c4b04a674`)

Unit: commit `c4b04a674` ("fix(atlas-channel): grant controller for Player NPCs
so they speak"). Brief: `docs/tasks/task-251-player-npcs/bug-player-npc-no-chat-balloon.md`
(reverses design D-4, amends FR-7.4).

Scope reviewed: the 8-file diff of `c4b04a674^..c4b04a674`, plus the contracts
the diff depends on — `npc/controller/processor.go`, `npc/controller/registry.go`,
the three pre-existing `AnnounceGrant` call sites, and the two inbound NPC-action
paths that the new grant makes reachable for Player NPC oids
(`socket/handler/npc_action.go`, `movement/processor.go`). Scope matches the
stated unit; no mismatch.

## Verdict

CHANGES_REQUIRED — 2 blocking, 7 non-blocking, 3 not evaluable.

## Requirement-by-requirement

| Intent | Result | Evidence |
|---|---|---|
| Map-enter replay claims and grants per Player NPC | PASS | `kafka/consumer/map/player_npc.go:105-115` — `TryClaim` then `NpcSpawnRequestControllerWriter`, mirroring `spawnNPCForSession` at `kafka/consumer/map/consumer.go:690-700`; grant strictly after `SpawnNPC` (`:95`) and before the batched `IMITATED_NPC_DATA` (`:117`). |
| Grant on DEPLOYED for players already in the map | PASS | `kafka/consumer/playernpc/consumer.go:150` calls `electController` after the spawn broadcast; `:158-170` elects and announces. |
| Release on REMOVED | PASS | `kafka/consumer/playernpc/consumer.go:237` `Release(f, e.Body.ObjectId)`; `Registry.Release` deletes the hash field (`npc/controller/registry.go:68-80`). |
| Re-elect on REPOSITIONED | PARTIAL | `kafka/consumer/playernpc/consumer.go:309-314` releases then re-elects; but the grant payload's coordinates are re-read rather than taken from the event — blocking #2. |
| CHARACTER_EXIT re-election works for a Player NPC oid | PASS | `kafka/consumer/map/consumer.go:645-661` is object-id agnostic; `ReleaseFor`/`ElectFor` never inspect the oid (`npc/controller/processor.go:147-244`), and `AnnounceGrant` now dispatches on the band (`npc/controller/announce.go:29-31`). Covered end-to-end by `player_npc_test.go:259-327`. |
| Player NPC removed between election and grant degrades gracefully | PASS with caveat | `playernpc/processor.go:63-73` returns `ErrNotFound`; `announce.go:33-36` logs Warn and returns the error; both callers (`playernpc/consumer.go:166`, `map/consumer.go:659`) log and continue — no panic, no packet. Caveat is non-blocking #5 (the claim made by `ElectFor` is not undone). |
| No import cycle / `Processor` interface extended cleanly | PASS | `playernpc/processor.go` imports only model/field/logrus; `npc/controller` -> `playernpc` is a new one-way edge. `go test ./kafka/consumer/map/... ./npc/... ./playernpc/...` builds and passes. |

## Blocking

### B1 — inbound NPC action for a Player NPC oid hits an atlas-data lookup that cannot succeed

`services/atlas-channel/atlas.com/channel/socket/handler/npc_action.go:31` and
`services/atlas-channel/atlas.com/channel/movement/processor.go:128` both resolve
the inbound oid with `data/npc.GetInMapByObjectId(mapId, oid)` and, on miss, log
`Errorf("Unable to retrieve npc moving.")` and drop the packet.

Before this commit that path was unreachable for a Player NPC: no grant meant the
client never made the object local, and `IsController` returned false anyway.
After it, the controlling client owns the object — and the brief's own IDA extract
shows the enabled path is gated on `m_bMovePathSent`, i.e. the controlling client
sends the NPC move/action packet for it. The oid is guaranteed absent from
atlas-data's per-map list: `socket/handler/npc_start_conversation.go:26-34`
states exactly that, and `libs/atlas-object-id/reserved.go:44-58`'s placeholder
filter removes the WZ life entries.

Consequences: an error-level log per action tick per controlling client per
deployed Player NPC, and no relay of the action to other sessions in the map.
Neither path grew a Player NPC branch in this commit.

### B2 — the REPOSITIONED grant carries coordinates the same handler already treats as stale

`kafka/consumer/playernpc/consumer.go:284` deliberately builds the respawn from
the *event* body (`rn.X, rn.Cy, rn.Fh, rn.Rx0, rn.Rx1`) rather than from the
read-back model `n`, whose position it evidently does not trust. Three lines
later the re-election at `:313` routes through
`npc/controller/announce.go:31-37`, which discards both and performs a *second*
`playernpc.GetInMapByObjectId` read, then encodes `n.X(), n.Cy(), n.Fh(),
n.RX0(), n.RX1()` into the grant. `NpcControllerGrantBody`
(`libs/atlas-packet/npc/spawn_request_controller_body.go:27-31`) carries the full
position payload.

So the respawn packet and the grant packet for the same object, in the same
handler invocation, are sourced from two different snapshots with two different
notions of authority. The handler exists to honour design §5.4/§7.4 ("never leave
a client holding a stale position"). The model and the event coords are both
already in hand at `:313`; nothing forces the extra read.

Whether the client re-applies the grant payload's coordinates to an
already-materialized NPC is not evaluable from this surface (see N/E 1) — but the
correctness of the emitted packet is unproven, and the inconsistency is
unconditional.

## Non-blocking

1. `kafka/consumer/playernpc/consumer.go:130-131` — `broadcastSpawn`'s doc comment
   still reads "sends the plain `SPAWN_NPC` (design D-4 -- no controller grant,
   FR-7.4)" while the function's last statement is `electController`. Directly
   contradicts the code below it.
2. Two different predicates now answer "is this a Player NPC oid":
   `libs/atlas-object-id/reserved.go:41-43` (`IsPlayerNpcObjectId`, narrow band
   `[101000, 106599]`) and `socket/handler/npc_start_conversation.go:32` (open-coded
   `[PlayerNpcObjectIdBase, MinId)` = `[100000, 999999]`). An oid in the gap is a
   Player NPC to the click guard and an ordinary NPC to `grantBody`. The new
   helper's own doc claims "a movement guard" as a caller; no guard uses it.
3. No test exercises `grantBody`'s Player NPC branch
   (`npc/controller/announce.go:29-37`). Every fixture uses script ids
   `9900001`-`9900007` -> oids `100001`-`100007`, which are *below*
   `playerNpcImitateTemplateMin` (9901000 -> 101000) and therefore return `false`
   from `IsPlayerNpcObjectId`. The exit test (`player_npc_test.go:259`) also never
   reaches `AnnounceGrant`, because `ElectFor` has no eligible candidate by
   construction. The central new dispatch is untested.
4. Cost per grant is one paginated REST drain of the whole map
   (`playernpc/processor.go:63-67` -> `InMapModelProvider` ->
   `requests.DrainProvider(..., 250, ...)`). CHARACTER_EXIT re-election of K
   Player NPC oids = K drains (`map/consumer.go:658-661`); REPOSITIONED of K = 1
   + K drains, the first at `playernpc/consumer.go:262`. Acceptable at Hall of
   Fame-scale K, wasteful at pool-scale, and avoidable in the REPOSITIONED case
   where `byObjectId` already holds the model.
5. Object removed between election and grant leaves a dangling claim: `ElectFor`
   has already written the entry (`npc/controller/processor.go:234-241`) when
   `AnnounceGrant` fails with `ErrNotFound`; `playernpc/consumer.go:166` and
   `map/consumer.go:659` log only. The oid stays claimed until its holder exits,
   and a redeploy reusing the same script id in that window would lose its
   `SetNX` and be granted to nobody.
6. `kafka/consumer/buff/consumer.go:396-412` (GM-reveal) enumerates candidates
   from `data/npc.InMapModelProvider` only, so an uncontrolled Player NPC is never
   re-elected on reveal. Narrow (map-enter `TryClaim` covers the usual case) but
   now asymmetric with the hide path, which does re-elect Player NPCs via
   `ReleaseFor`.
7. `player_npc_test.go:127-138` — the `sync.Once` miniredis is never closed; the
   process keeps the server for the run. Trivial.

Docs: `design.md:508-528` (§7.2 D-4, §7.3) and `prd.md:274` (FR-7.4) still read as
current policy. The bug doc records the amendment; the design doc does not
reference it.

## Test honesty

- The new subtest `controller grant follows SpawnNPC when the session wins the
  claim` (`player_npc_test.go:186-198`) asserts an exact three-writer sequence
  including `NpcSpawnRequestControllerWriter`. The pre-change
  `spawnPlayerNpcsForSession` had no call site for that writer at all (the old
  subtest asserted its *absence*), so the test genuinely fails without the change.
- Subtest isolation: distinct oids per subtest (100001, 100002, 100007,
  100003-100005, 100006), no `t.Parallel`. Verified empirically — `go test -count=1
  -shuffle=on ./kafka/consumer/map/` passed three consecutive runs, and
  `-run 'TestSpawnPlayerNpcForSession/Player_NPC_object_id_enters_the_controller_registry'`
  passes in isolation. No order dependency found.
- The `sync.Once` registry is shared but keyed per `(tenant, field, oid)`; the
  exit test's `ReleaseFor(7777)` cannot see the spawn subtests' claims (held by
  the zero-value session's character id 0).

## Not evaluable

1. Whether `CNpcPool::OnNpcChangeController` re-applies the grant payload's
   coordinates to an NPC already present in the pool. Requires IDA; bears
   directly on B2's severity.
2. Whether a non-controlling client in the same map ever renders the balloon. The
   brief's evidence says the balloon is driven by the local `CNpc::Update` loop,
   which implies only the single elected controller sees it — but the relay side
   of that question is entangled with B1 and cannot be settled from this surface.
3. Live confirmation that a deployed Player NPC now speaks. No live run is part
   of this unit.
