# task-278 — manual test notes (pr-1566, live session)

Status: **feature verified end-to-end against a live client.** One defect found
in the reset path (see "Defect 1"). Superseded findings from the first attempt
are kept at the bottom for provenance.

## Field under test

- Namespace `atlas-pr-1566`, ingress host `1566.atlas.home`, env id `6a89`,
  image `pr-1566-e050686`.
- Tenant serving the live client: `d1defcd0-9994-4ee2-bcad-9eaaa1377839` (GMS 83.1).
  NOT `8fca08b3-…` and NOT the bootstrap `00000000-…-0001`; a POST to the wrong
  tenant is accepted and consumed but announces to nobody.
- Field: world 0, channel 0, map 103000800 (Kerning PQ stage 1), instance
  `00000000-0000-0000-0000-000000000000`.
- Object: `name="gate"`, kind `ENVIRONMENT`. Map obj entry
  (`Map.wz/Map/Map1/103000800.img`, layer 4, entry 34): `oS=effect`, `l0=quest`,
  `l1=gate`, `l2=1`, x=715, y=34.
- **Observed in-client: state `0` renders the visible gate; states `1` and `2`
  render nothing.** This is the empirical result from the live client and it is
  what the notes rely on. Do not re-derive it from the WZ tree — an earlier pass
  read `Obj/effect.img/quest/gate/0` (a 1x1 canvas) as "invisible" and predicted
  the opposite polarity three times over. The client disagrees; the client wins.

## Verified behaviours

All four confirmed on 2026-09-02, tenant `d1defcd0…`, map 103000800.

1. **Direct state change → visible client change.** `POST` state 0 → 202 →
   `EVENT_TOPIC_MAP_STATUS-6a89` `ENVIRONMENT_STATE_CHANGED` → channel logs
   `Environment object [gate] kind [ENVIRONMENT] set to state [0]` (18:20:36.785Z)
   → gate visible in client. Opcode `0x99` / `CField::OnSetObjectState`, matching
   the IDA-verified v83 codec (`set_object_state_test.go` `ida=0x537a1e`);
   body = len-prefixed name + 4-byte LE state.
2. **Reset on empty field.** Warping the only session out fires
   `originator=change_character_location` →
   `Environment reset in map [103000800]; cleared [1] object(s)` (18:19:07.831Z);
   subsequent `GET` returns `{"data":[]}`.
3. **Replay on enter.** With state 0 tracked, a second character entering the map
   triggers the channel's `GET .../environment` on `EVENT_TOPIC_CHARACTER_STATUS`,
   which returned `[{"kind":"ENVIRONMENT","name":"gate","state":0}]` (18:21:36.285Z);
   the entering client sees the gate. This path only works because of the ingress
   route added in `e050686d4` — it is a cross-service call through the ingress.
4. **`DELETE` reset.** 204, tracking cleared, channel logs
   `Environment reset in map [103000800]; restoring [1] tracked object(s)`
   (18:23:01.998Z). See Defect 1 for what it restores *to*.

## Defect 1 — reset restores state 0, not the object's declared default

`handleStatusEventEnvironmentReset`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`)
re-announces a hardcoded `0` for every cleared object:

```go
// Explicitly zero every tracked object
_ = announceObjectState(l, ctx, wp, kind, o.Name, 0, s)
```

For this gate, state 0 is the *visible* state and the map declares `l2=1`, so a
reset leaves the gate **visible** instead of restoring the map's initial
appearance. Confirmed in-client: after `DELETE`, the gate remained visible.
For a PQ reset this is backwards — the stage would reset into its completed look.

`0` is only correct for `ObjectKindObstacle`, where the client's
`FieldObstacleAllReset` genuinely means "all off". It is not a safe default for
named environment objects, whose rest state is whatever `l2` the map declares.

**Blocker on the obvious fix:** the declared default is not available anywhere
server-side. `services/atlas-data/atlas.com/data/map/reader.go` does not parse
the map's `obj` node at all (its only match is `shipObj` at :319), so no service
knows an object's `l2`. Fixing this properly means extending the atlas-data map
reader to expose obj entries (at minimum `name` + `l2`), then having atlas-maps
carry the original state in `Cleared` so the channel restores that instead of 0.

That is a design decision, not a mechanical fix — it widens task-278 into
atlas-data. Flagging rather than implementing.

## Fix already landed on this branch

**No ingress route for `.../environment`** — `deploy/shared/routes.conf` had
`.../weather` and `.../jukebox` but no `.../environment`, so the endpoint fell
through the `^/api/worlds(/.*)?$` catch-all to atlas-world. Fixed in commit
`e050686d4` (+ regenerated `routes.conf.template.generated`);
`gen-routes.sh --check` and `gen-routes_test.sh` pass. Without it, behaviour 3
above cannot work in any deployed environment.

## Superseded / environment incident (first attempt, ~15:20–17:35Z)

Kept because the root cause is a live defect in pr-environment provisioning, not
in task-278.

A `diagnostics.tracePackets` PATCH on tenant `8fca08b3…` published a tenant-status
event carrying the row's server-owned `environment: "main"`. pr-1566 is env `6a89`
with an **empty `environments` table**, so from that moment every environment-tagged
message was dropped at the consumer gate (`libs/atlas-kafka/consumer/gate.go`),
first `reason=stale` then `reason=not_active`. Before the PATCH nothing had
published the mapping, so messages flowed untagged down the legacy path — which is
why one early POST worked and later ones did not. This broke **login**, not just
task-278 (`COMMAND_TOPIC_ACCOUNT_SESSION-6a89` session CREATE dropped).

`environment` is immutable through PATCH (tenants `processor.go:174`
`sanitized.Environment = persisted.Environment`), so the repair was DELETE + POST
with the id preserved: `Create` keeps `input.Id` and takes `Environment` from the
request context, which is empty absent an `ENVIRONMENT` header. Applied to the
tenant row and to the login service-config row `e7fb1d7e-47b8-46bd-97dc-867d93530856`
(which had crash-looped on `configuration: projection snapshot not yet published`).
`tracePackets` restored to `false`.

**Still outstanding:** `atlas-channel`'s service-config row
(`e7fb1d7e-47b8-46bd-97dc-867d93530000`) still carries `environment: 'main'`. It
is healthy only because it holds a snapshot fetched earlier; it will likely hit
the same fatal on its next restart. Other service rows were not audited. The
underlying provisioning defect — `main`-stamped config rows seeded into a `6a89`
namespace with an empty `environments` table — is task-232 territory and will
recur on any pr environment seeded the same way.

Backups: `/tmp/tenant-backup.json` (pre-incident), `/tmp/tfinal.json` (post-repair).
