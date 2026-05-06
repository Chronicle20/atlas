# Forced Return on Exit — Design Document

Version: v1
Status: Draft
Created: 2026-05-04
PRD: `prd.md`

---

## 1. Architecture in one sentence

Move durable character-location ownership from atlas-character to atlas-maps; atlas-maps becomes the consumer of `CHANGE_MAP` and `CHANGE_CHANNEL_REQUEST` and the sole emitter of `MAP_CHANGED` and `CHANNEL_CHANGED`; the forced-return rule is one in-process branch inside atlas-maps' write path. Per-feature exit handlers (timer, transports, party-quests) keep their domain bookkeeping and drop their warp emits.

## 2. Decisions (with rationale)

The PRD left six open questions. Each was resolved in design dialog; the PRD-numbered question is referenced in parentheses.

| # | Decision | Rationale |
|---|----------|-----------|
| D1 (Q1) | Resolver lives **inside atlas-maps**, not in atlas-character or a shared lib. Exposed in-process to atlas-maps' own consumers and via REST (`GET /characters/{id}/location`) to other services. | atlas-maps already consumes the lifecycle events that need resolution (LOGOUT for the timer), already loads map info from atlas-data, and already maintains a character→map registry. Co-locating the rule with the data and the existing event coupling avoids a new service-to-service dependency from atlas-character. |
| D2 (Q2) | Timer's `forcedReturnMapId` always equals WZ `forcedReturn` (registration sources from `md.ForcedReturnMapId()` at `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go:96`). **Timer's `CHANGE_MAP` emit is retired.** | Same field, same source — the timer's emit is now redundant with the unified resolver's write. |
| D3 (Q3) | Transports keeps `CANCELLED` emit and instance release. **Drops `WarpToRouteStartMapOnLogoutAndEmit`** (`route.StartMapId()` → origin) in favor of the resolver's WZ `forcedReturn` (terminus). **Behavioral change**: Cosmic-parity terminus wins. | Cosmic parity reasoning in PRD §1; transit map `200090000`'s WZ `forcedReturn` is `200000100` (Orbis dock = terminus). The previous origin-dock behavior was Atlas-specific. |
| D4 (Q4) | `_map.EmptyMapId = Id(999999999)` already exists at `libs/atlas-constants/map/constants.go:2267`. Add an `IsSentinel()` method on `_map.Id` for ergonomics; do not add a new constant. | Reuse existing constant per CLAUDE.md DOM-21. |
| D5 (Q5) | PQ `def.Exit()` retirement: **out of scope**. Disconnect path stops using it; non-disconnect leaves (NPC, expulsion, completion) keep it. Documented as TODO §10. | Cosmic parity for non-disconnect leaves may diverge per PQ; not a blocker for this task. |
| D6 (Q6) | atlas-transports `HandleLogin` transit-map detection branch is **redundant after this change** and gets removed. | Once a player on a transit map is persisted at the WZ forced-return target, the "logged in on a transit map" case can no longer arise. |

Additional design-phase decisions (not in PRD's open list):

| # | Decision | Rationale |
|---|----------|-----------|
| D7 | **PRD §4.1 Rule 1 (HP ≤ 0 → returnMap) is dropped.** atlas-maps does not need HP. | `atlas-channel/respawn` already honors `returnMap` on click-respawn. Dead-DC case results in one extra respawn click on relog — acceptable. atlas-maps stays free of session-state inputs. |
| D8 | **Persistence migrates from atlas-character to atlas-maps entirely.** Drop `map_id` and `instance` columns from `atlas-character.characters`. Add `character_locations` table on atlas-maps. | Logical conclusion of "atlas-maps owns the resolver." Map state belongs to the map service. Migration is trivial (2 existing rows). |
| D9 | **Channel-change uses request/response pattern**: `CHANGE_CHANNEL_REQUEST` → atlas-maps validates+resolves → `CHANNEL_CHANGED`. atlas-channel's old trigger is replaced (not run in parallel). | Future hook point for additional channel-change validators (PQ membership, anti-MP-up-on-change, etc.) — they subscribe to REQUEST in parallel with atlas-maps. |
| D10 | **atlas-maps in-memory registry moves to Redis.** | Aligns with the active `legacy-redis-registry-migration` initiative. atlas-maps already uses Redis for spawn cache (per project memory `reference_atlas_maps_spawn_cache.md`); no new ops dependency. Multi-pod-safe. |
| D11 | **Status events keep their `MapId`/`Instance` fields populated** for backward compatibility with the 43 consumers. atlas-character queries atlas-maps before emitting LOGIN/LOGOUT to fill them. | Avoids breaking 43 consumer schemas in this task. Tagged TODO §10 for follow-up cleanup (split into LOCATION_* + lifecycle-only events). |
| D12 | **Atlas-channel fails closed on atlas-maps unreachable at session bootstrap.** Kicks the player back to character-select with an error rather than spawning at a fallback map. | A character at the wrong map is worse than a clear error. Consistent with the "no silent fallback" principle. |

## 3. Components

### 3.1 atlas-maps (gains location ownership)

**New table** `character_locations`:

```
character_locations
  tenant_id    UUID    PK part 1
  character_id BIGINT  PK part 2     (uint32 in app)
  world_id     SMALLINT  NOT NULL    (byte)
  channel_id   SMALLINT  NOT NULL    (byte)
  map_id       INTEGER   NOT NULL    (uint32)
  instance     UUID      NOT NULL DEFAULT '00000000-...'
  updated_at   TIMESTAMP NOT NULL
```

GORM entity at `services/atlas-maps/atlas.com/maps/character/location/entity.go`. Migration in atlas-maps' migration set.

**New processor** `character/location/processor.go` with the standard atlas-maps Interface+Impl pattern:

```go
type Processor interface {
    GetByIdProvider(characterId uint32) func() (Model, error)
    Set(characterId uint32, field field.Model) error
    Resolve(currentField field.Model) (field.Model, ResolutionReason, error)
}
```

`Resolve` is the rule: load map info via existing `info.NewProcessor(p.l, p.ctx).GetById(currentField.MapId())`; if `info.ForcedReturnMapId().IsSentinel()` return `(currentField, ReasonStayPut)`; else return `(field with MapId=ForcedReturnMapId, Instance=uuid.Nil, ReasonForcedReturn)`.

**Spawn portal selection on resolver-driven relocation**: a relocation produced by `Resolve` lands the player at the destination map's *default spawn portal* (portal id `0`). This is intentionally narrow scope:

- **Portal walks** (player walks through a portal): the WZ portal-pair link drives target-portal selection. Authored by atlas-channel's portal handler in the existing `CHANGE_MAP` command. Unchanged in this task — the resolver does not run on this path; the player picked the portal explicitly.
- **NPC warps / GM commands / scripted teleports**: caller-specified portal id (or `0` for default). Unchanged.
- **Resolver-driven relocation** (this task — disconnect or channel-change forced-return): no source portal exists because the player wasn't walking through anything. We use the destination's default portal. We do *not* introduce a "remember the source map and pick a corresponding portal on the target" mapping — that's the new spawn-portal selection logic the PRD §2 non-goals exclude.

In code terms: the `field.Model` returned by `Resolve` carries `MapId` and `Instance`; the consumer that emits the resulting `CHANGE_MAP`-equivalent or `CHANNEL_CHANGED` event sets `TargetPortalId = 0` for resolver-driven cases. Existing portal-id behavior on portal walks and scripted warps is untouched.

**New REST**: `GET /characters/{id}/location` → JSON:API resource with `world_id, channel_id, map_id, instance`. Tenant-scoped via existing handler middleware. Used by atlas-channel session bootstrap and atlas-character status-event emit (backward compat per D11).

**Consumer changes** in `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go`:

- `handleStatusEventLogout`: extend to call `location.Resolve` + `Set`. Drop entry from Redis presence. (Existing timer cancel via `ForceReturnIfTracked` stays; the `CHANGE_MAP` emit inside `ForceReturnIfTracked` is removed per D2.)
- `handleStatusEventChannelChanged`: removed. CHANNEL_CHANGED is now emitted by atlas-maps itself (see CHANGE_CHANNEL_REQUEST consumer).
- `handleStatusEventMapChanged`: removed. MAP_CHANGED is now emitted by atlas-maps itself.
- `handleStatusEventLogin`: extend to hydrate Redis presence registry from atlas-maps' own location row.

**New consumer** for `COMMAND_TOPIC_CHARACTER` (or whatever topic carries `CHANGE_MAP` today): atlas-maps takes over from atlas-character. Reads old `character_locations` row, writes new, emits `MAP_CHANGED` status event to `EVENT_TOPIC_CHARACTER_STATUS`. Updates Redis presence (move from old map set to new).

**New consumer** for `CHANGE_CHANNEL_REQUEST` (new topic — see §3.5): looks up current location, calls `Resolve`, writes `character_locations` (channel_id=target, map_id=resolved), emits `CHANNEL_CHANGED` to `EVENT_TOPIC_CHARACTER_STATUS`.

**Registry migration** (D10): the existing `getCharacterRegistry` singleton (and similar per-map character-list structures) move to Redis-backed access:

- `atlas:maps:online:{tenantId}:{characterId}` — hash `{world_id, channel_id, map_id, instance}` for point lookups.
- `atlas:maps:presence:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` — set of `characterId` for "who is on this map".

Populated on LOGIN/MAP_CHANGED/CHANNEL_CHANGED; cleaned on LOGOUT. No TTL (session-scoped). Cold start: lazy hydration as events flow.

The timer's per-character forcedReturnMapId tracking can be retired entirely — once `Resolve` is the single rule source, the timer doesn't need to remember it. Only the `(characterId, expiry)` pair remains.

### 3.2 atlas-character (subtractive)

**Schema migration**: drop `map_id` and `instance` columns from `characters` table.

**Model** (`services/atlas-character/atlas.com/character/character/model.go`):
- Remove `MapId()` getter (line 99).
- Remove `Instance()` getter (line 103).
- Remove `SetMapId` / `SetInstance` builder methods (lines 401, 406).
- Remove `mapId` / `instance` fields and `CloneModel` copies of them.

**Entity** (`services/atlas-character/atlas.com/character/character/entity.go`): drop columns and their GORM tags.

**REST** (`services/atlas-character/atlas.com/character/character/rest.go`): per D11 backward compat, the `RestModel.MapId` and `RestModel.Instance` fields stay in the JSON shape *for now*. The `Transform()` function (lines 70–114) populates them via an in-flight call to atlas-maps' `GET /characters/{id}/location`. Tagged TODO §10.1 for removal.

**Processor** (`services/atlas-character/atlas.com/character/character/processor.go`):
- `Logout` (line 402): stop reading `c.MapId()` / `c.Instance()` from the model. Query atlas-maps for current location to populate the LOGOUT status event payload (D11). Emit LOGOUT as today.
- `ChangeChannel` (line 416): **removed entirely**. CHANNEL_CHANGED is now emitted by atlas-maps in response to CHANGE_CHANNEL_REQUEST.
- `ChangeMapAndEmit` (line 426): **removed entirely**. CHANGE_MAP command is now consumed by atlas-maps; atlas-character has no map-write responsibility.
- `dynamicUpdate` paths that touched map_id/instance: removed.
- `announceMapChangedWithBuffer` (line 455): removed (only callsite was `ChangeMap`).

**Consumer**: the `CHANGE_MAP` command consumer in atlas-character is removed (responsibility migrates to atlas-maps).

### 3.3 atlas-channel (modified at three sites)

**Session bootstrap** (wherever the player's spawn map is read after character-select):
- Pivot from `c.MapId()` / `c.Instance()` on the atlas-character REST model to `GET /characters/{id}/location` on atlas-maps.
- On atlas-maps unreachable: fail closed (D12). Drop the session, emit a clear error to the client, return to character-select.

**Channel-change handler** (`services/atlas-channel/atlas.com/channel/socket/handler/channel_change.go`):
- Keep the HP > 0 gate at line 30.
- Replace today's trigger to atlas-character with a `CHANGE_CHANNEL_REQUEST` emit on a new topic. Payload: `(characterId, oldChannelId, targetChannelId, tenant context)`.
- Future validator services subscribe to REQUEST in parallel with atlas-maps.

**Respawn processor** (`services/atlas-channel/atlas.com/channel/respawn/processor.go:74-89`): the cash-shop "Wheel of Fortune" branch reads `currentMapId` — pivot to read from atlas-maps' location instead of atlas-character's REST model. Same pivot principle as session bootstrap.

**Live session state** (`session.Model.MapId()` etc.): unchanged. Live position is a session/runtime concern owned by atlas-channel; durable location is atlas-maps'. The two are separate. Movement commands (`COMMAND_TOPIC_CHARACTER_MOVEMENT`) continue to author `(worldId, channelId, mapId, instance)` from atlas-channel's session.

### 3.4 atlas-login (modified at character-select)

**Character-list packet writer** (`services/atlas-login/atlas.com/login/socket/writer/character_list.go:41`): currently `uint32(c.MapId()), c.SpawnPoint()`. Pivot:
- For each character in the list, in-process fetch from atlas-maps' `GET /characters/{id}/location`. (Single-call loop, N is small per D11 confirmation.)
- `SpawnPoint` stays on atlas-character (it's not location, it's the "saved spawn portal id").

### 3.5 New Kafka topic — CHANGE_CHANNEL_REQUEST

Topic name: `COMMAND_TOPIC_CHARACTER_CHANNEL_CHANGE_REQUEST` (env: `EnvCommandTopicCharacterChannelChangeRequest`). Conventional naming aligned with existing command topics.

Message shape:

```go
type ChangeChannelRequestCommand struct {
    Tenant       tenant.Model `json:"tenant"`
    CharacterId  uint32       `json:"characterId"`
    OldChannelId channel.Id   `json:"oldChannelId"`
    TargetChannelId channel.Id `json:"targetChannelId"`
}
```

Producer: atlas-channel's `channel_change.go` handler.
Consumer: atlas-maps (validates HP gate already done, resolves location, writes, emits `CHANNEL_CHANGED`).
Future consumers: any service that needs to veto a channel-change. Not part of this task.

### 3.6 atlas-transports (subtractive)

`services/atlas-transports/atlas.com/transports/instance/processor.go`:
- `HandleLogout` (lines 243-275): no code change in this task. The impact survey at design phase confirmed atlas-transports does not currently emit a warp on logout (the PRD's mention of `WarpToRouteStartMapOnLogoutAndEmit` was outdated wording — the warp lives in `HandleLogin`, not `HandleLogout`). Instance release + `CANCELLED` emit + registry cleanup behavior is preserved.
- `HandleLogin` (lines 283-299): **remove the transit-map detection branch entirely.** With atlas-maps persisting WZ forced-return on disconnect, the post-DC login can never land on a transit map. Removing the branch eliminates dead code and the divergent `route.StartMapId()` warp.

### 3.7 atlas-party-quests (subtractive on disconnect)

`services/atlas-party-quests/atlas.com/party-quests/instance/processor.go:917`:
- `Leave(reason string)`: when `reason == "disconnect"`, skip the `def.Exit()` warp emit at line 953. Other reasons (NPC, expulsion, completion) keep the existing warp.
- Concretely: wrap the `mb.Put(character2.EnvCommandTopic, warpCharacterProvider(...))` call at line 953 in `if reason != "disconnect" { ... }`.
- `CHARACTER_LEFT` event emit (line 962) and registry cleanup are unconditional — not affected.

### 3.8 atlas-maps timer (subtractive)

`services/atlas-maps/atlas.com/maps/map/timer/processor.go::ForceReturnIfTracked`:
- Drop the `CHANGE_MAP` emit. The unified location processor inside the same service has already written the resolved location.
- Keep timer cancellation, log/span emission, registry cleanup.
- The `forcedReturnMapId` field on the timer entry (used only for the now-deleted emit) can be removed from the entry struct.

### 3.9 libs/atlas-constants/map (additive)

`libs/atlas-constants/map/model.go` (or constants.go): add method on `_map.Id`:

```go
func (id Id) IsSentinel() bool {
    return id == EmptyMapId
}
```

No new constant; reuse existing `EmptyMapId = Id(999999999)` at constants.go:2267.

## 4. Data flow

### 4.1 Logout

```
1. atlas-channel detects socket close
2. atlas-channel → existing trigger to atlas-character.Logout
3. atlas-character.Logout:
     - GET atlas-maps /characters/{id}/location  (D11 backward compat)
     - emit LOGOUT status event to EVENT_TOPIC_CHARACTER_STATUS
       payload includes (worldId, channelId, mapId, instance) from atlas-maps
4. atlas-maps consumes LOGOUT:
     - location.Resolve(currentField) → resolved field
     - location.Set(characterId, resolved field)
     - Redis: DEL atlas:maps:online:{t}:{cid}
     - Redis: SREM atlas:maps:presence:{t}:{world}:{ch}:{map}:{inst} cid
     - timer.ForceReturnIfTracked: cancel timer (no CHANGE_MAP emit)
5. atlas-transports consumes LOGOUT: instance release + CANCELLED emit
6. atlas-party-quests consumes LOGOUT: CHARACTER_LEFT emit + registry cleanup
   (no warp — D7/§3.7)
```

### 4.2 Channel-change

```
1. Player sends CHANGE_CHANNEL packet to atlas-channel
2. atlas-channel handler:
     - HP > 0 gate (existing)
     - emit CHANGE_CHANNEL_REQUEST(characterId, oldChannelId, targetChannelId)
3. atlas-maps consumes REQUEST:
     - load location row → currentField
     - Resolve(currentField) → resolved field on target channel
     - location.Set with (target channel, resolved map, uuid.Nil instance)
     - Redis: HSET atlas:maps:online:{t}:{cid} = resolved
     - Redis: SMOVE presence keys (old → new map+inst)
     - emit CHANNEL_CHANGED(oldField=current, newField=resolved)
4. atlas-channel new-channel handoff consumer: react as today
   (consumes the CHANNEL_CHANGED that atlas-maps emitted)
```

### 4.3 Map-change (portal walk)

```
1. atlas-channel (portal handler) emits CHANGE_MAP command
   (existing producer; payload unchanged)
2. atlas-maps consumes CHANGE_MAP (new — was atlas-character):
     - load location → oldField
     - location.Set(characterId, newField from command)
     - Redis: HSET online:{t}:{cid} = newField
     - Redis: SREM presence(old), SADD presence(new)
     - emit MAP_CHANGED(oldField, newField)
3. All existing MAP_CHANGED consumers react as today.
```

### 4.4 Login

```
1. Client picks character → atlas-channel session bootstrap:
     - GET atlas-character /characters/{id}  (model without map fields)
     - GET atlas-maps     /characters/{id}/location  (location)
     - on atlas-maps unreachable: fail closed (D12)
     - seed session with location
2. atlas-character.Login emits LOGIN status:
     - GET atlas-maps /characters/{id}/location  (D11 backward compat)
     - emit LOGIN with map fields populated
3. atlas-maps consumes LOGIN:
     - hydrate Redis: HSET online:{t}:{cid}, SADD presence({world,ch,map,inst})
4. Other LOGIN consumers react as today.
```

## 5. Migration

**Schema changes**

1. atlas-maps adds `character_locations` table (new migration).
2. atlas-character drops `map_id` and `instance` columns from `characters` (new migration).

**Backfill**: only 2 existing characters per user confirmation. One-shot script:

```sql
-- Run per tenant
INSERT INTO atlas_maps.character_locations
  (tenant_id, character_id, world_id, channel_id, map_id, instance, updated_at)
SELECT
  $tenant_id,
  c.id,
  0,    -- world_id placeholder; characters not online at deploy
  0,    -- channel_id placeholder
  c.map_id,
  c.instance,
  NOW()
FROM atlas_character.characters c;
```

The `world_id`/`channel_id` placeholders are inert at deploy — both characters are offline; values get corrected at next login. If we want stricter accuracy, run backfill during a maintenance window when no one is online.

**Deploy ordering**:

1. Deploy atlas-maps with new table + processor + REST + consumers (atlas-character is unaware; consumers are no-ops because no events route to the new code paths yet).
2. Backfill `character_locations`.
3. Deploy atlas-character with `Logout` modified to query atlas-maps; `ChangeChannel` and `ChangeMap` removed; columns dropped.
4. Deploy atlas-channel with `CHANGE_CHANNEL_REQUEST` emit, session-bootstrap pivot, respawn pivot.
5. Deploy atlas-login with character-list pivot.
6. Deploy atlas-transports and atlas-party-quests subtractive changes.

Steps 3+ can be a single deploy if migrations land cleanly. Atlas-maps in step 1 is the only deploy that strictly precedes the others.

**Rollback**: revert in reverse order. Backfill can be reversed by reading `character_locations` back into `atlas-character.characters`. The 2-row scale makes any rollback path trivial.

## 6. Error handling

| Failure | Behavior |
|---------|----------|
| atlas-maps unreachable on session bootstrap | atlas-channel kicks the client back to character-select with a clear error; no silent fallback (D12). |
| atlas-maps unreachable when atlas-character is populating LOGIN/LOGOUT for backward compat | atlas-character emits with zero/empty `MapId`/`Instance` and logs a warning. atlas-maps' own LOGOUT/LOGIN consumers do NOT depend on the populated payload — they read the canonical row from `character_locations`. The populated fields are purely advisory to other consumers. |
| `Resolve` cannot load map info (atlas-data unreachable) | atlas-maps logs warning and falls through to "stay put" — write `currentField` unchanged. Better than relocating to a wrong target. |
| Redis unreachable | Live-presence reads/writes fail; durable Postgres path still works. Timer/broadcast features that rely on presence degrade until Redis recovers. Consistent with rest of codebase's Redis-reliance posture. |
| Race: player disconnects mid-channel-change (after REQUEST, before CHANNEL_CHANGED) | The LOGOUT consumer in atlas-maps idempotently overwrites `character_locations` to the resolved-on-disconnect target. CHANGE_CHANNEL_REQUEST consumer's write may land second and clobber — that's correct (whichever is logically last wins, and disconnect overrides channel-change since the player is offline). |

**Tenant scoping**: every `location` processor read/write uses `tenant.MustFromContext(ctx)`. REST handler uses existing tenant-header middleware. Backfill enumerates per-tenant.

## 7. Observability

- atlas-maps' `location` processor logs `{characterId, currentMapId, resolvedMapId, reason}` on every non-trivial resolution (`reason != ReasonStayPut`).
- OTel spans on resolution: `forced.return.map.id`, `resolution.reason`, `tenant.id`.
- Existing log/span emission in `MapTimer.Disconnect`, atlas-transports `HandleLogout`, atlas-party-quests `Leave` is preserved.
- New metric: `atlas_maps_location_resolutions_total{reason="forced_return"|"stay_put"}` — observability into how often forced-return fires in production.

## 8. Testing

**Unit**

- `location.Resolve` table-driven cases:
  - Sentinel `forcedReturn` → stay put.
  - Non-sentinel `forcedReturn` → relocate, instance=Nil.
  - Map info load error → stay put + warning logged.
  - `_map.Id.IsSentinel()` truth table.
- `location.Processor` Get/Set:
  - Tenant scoping (read for tenant A returns nothing for tenant B's character).
  - Concurrent Set: last write wins (idempotent).
- atlas-maps consumer changes: assert correct topic emit shapes for MAP_CHANGED / CHANNEL_CHANGED.
- Redis presence: SADD/SREM pair on map transitions.

**Integration** (golden-path scenarios from PRD acceptance criteria)

| # | Scenario | Expected |
|---|----------|----------|
| I1 | Disconnect on KPQ room (`103000800`, forcedReturn=`103000890`) | Login lands at `103000890` instance=Nil. No PQ membership. |
| I2 | Disconnect on transit map `200090000` | Login lands at `200000100` (Orbis dock). HandleLogin no-op. |
| I3 | Disconnect on time-limited map | Login lands at the WZ forcedReturn target. Timer cancelled. |
| I4 | Disconnect on Henesys Hunting Ground 1 (`100020000`, sentinel forcedReturn) | Login lands at `100020000`, same instance as before logout. |
| I5 | Channel-change on KPQ room | Lands on new channel at `103000890` instance=Nil. |
| I6 | Channel-change on regular map | Lands on new channel at same map, same instance. |
| I7 | Concurrent disconnect during channel-change | character_locations final value matches disconnect's resolution. |
| I8 | atlas-maps unreachable during session bootstrap | atlas-channel returns error to client; player at character-select. |

**Migration**

- Backfill script run on a fixture DB with sample rows; verify `character_locations` populated and atlas-character columns dropped successfully on a second run.

**Test surface awareness**: the impact survey identified ~117 test files referencing `MapId()` / `Instance()` on character mocks. A significant portion of plan-phase work is updating those tests to either drop the field expectations or mock `atlas-maps/character/location` instead.

## 9. Out of scope (deferred)

- Multi-character / party-aware return logic.
- Removing PQ JSON `def.Exit()` field for non-disconnect leaves (see §10.5).
- Cash-shop "preserve current map on cash-shop close" semantics.
- New spawn-portal selection logic *for resolver-driven relocations*: the destination map's default spawn portal (id `0`) is used. We do not maintain a "source map → target portal" mapping that would let a force-returned player land at a specific portal on the destination. Existing portal-determination for portal walks (WZ portal-pair links) and scripted warps (caller-specified portal id) is untouched. See §3.1 for the three-case breakdown.
- Auto-respawn-on-login for dead-DC parity (D7).

## 10. Cleanup TODOs (follow-up tasks)

These are not blockers for task-055 but should be tracked as follow-up work.

### 10.1 atlas-character map awareness removal
- atlas-character emitting LOGIN/LOGOUT with `MapId`/`Instance` populated via in-flight atlas-maps lookup is a backward-compat shim (D11). Long-term:
  - Introduce `LOCATION_*` events from atlas-maps.
  - Migrate the location-aware consumers to subscribe to `LOCATION_*`.
  - Strip `MapId`/`Instance` from `EVENT_TOPIC_CHARACTER_STATUS` LOGIN/LOGOUT payloads.
  - Remove the atlas-maps lookup from atlas-character's emit path.
- atlas-character `RestModel.MapId` and `RestModel.Instance` are populated via the same shim. Long-term:
  - Remove the fields from `RestModel`.
  - atlas-login (the only direct REST consumer) already pivots to atlas-maps in this task; this cleanup just removes the dead REST fields.

### 10.2 atlas-character drop-command field context
- `services/atlas-character/atlas.com/character/character/kafka/message/drop/kafka.go:17-23` carries `MapId` and `Instance` on drop command bodies. Source today is atlas-character's character model. Once atlas-character has no map awareness, this needs to be sourced from atlas-channel's session at command-author time. Verify drop command authoring sites and ensure they read from session, not from the character model.

### 10.3 atlas-party-quests `def.Exit()` retirement (PRD Q5)
- Disconnect path stops using `def.Exit()` in this task.
- Non-disconnect leaves (NPC, expulsion, completion) still use it.
- Investigate whether WZ `forcedReturn` is the correct map for non-disconnect leaves on every PQ; if so, retire the JSON field. If some PQs diverge, document why.

### 10.4 Movement command source-of-truth review
- `COMMAND_TOPIC_CHARACTER_MOVEMENT` carries `(worldId, channelId, mapId, instance)` from atlas-channel's session. Once atlas-maps has the authoritative durable location and Redis-backed live-presence, evaluate whether atlas-channel still needs to send those fields or whether atlas-maps can derive them from `atlas:maps:online:{t}:{cid}`. Reduces payload size and removes another spot where field info is duplicated.

### 10.5 Future channel-change validators
- The `CHANGE_CHANNEL_REQUEST` topic introduced in this task is designed as an extensibility point. Concrete future validators to scope:
  - PQ membership block (cannot change channel while in a party quest).
  - Anti-MP-up-cheese (cannot change channel within N seconds of a stat-changing event).
  - Map-time-limit interaction (channel-change should not extend a time-limited map's grace).

### 10.6 atlas-transports `route.StartMapId()` review
- Per D3, `WarpToRouteStartMapOnLogoutAndEmit` is dropped in favor of WZ-driven terminus. Verify in plan phase whether `route.StartMapId()` is used anywhere else (login crash recovery is being removed per D6); if not, the `StartMapId` concept on the route model can be retired entirely from atlas-transports.

## 11. Acceptance criteria mapping (from PRD §10)

| PRD criterion | Verified by |
|---------------|-------------|
| Disconnect on `forcedReturn = X` (non-sentinel) → relog at X, instance=Nil | I1, I2, I3 |
| Disconnect on `forcedReturn = sentinel` → relog at same map+instance | I4 |
| Disconnect at HP=0 with `returnMap = Y` (sentinel `forcedReturn`) | **PRD-§10 parity delta from D7**: Rule 1 dropped. On relog the player is at HP=0 on the same map; the existing click-respawn flow at `atlas-channel/respawn` warps to Y on click. One extra click vs. Cosmic auto-revive — accepted in design dialog. |
| Disconnect at HP=0 with both `forcedReturn = X` and `returnMap = Y` non-sentinel (PRD rule 1 wins over rule 2) | **PRD-§10 parity delta from D7**: rule 2 path fires → relog at X. Click-respawn at X uses X's own `returnMap` (typically X itself for safe-zone lobbies, so the player is unaffected). For maps where X's `returnMap` ≠ Y, Cosmic would have gone direct to Y; we go to X. Practical impact: most PQ rooms have X == Y or X is itself a safe lobby with `returnMap = X`. Documented for QA awareness. |
| Channel-change on forcedReturn map → resolved map on new channel | I5 |
| KPQ disconnect → KPQ lobby without PQ consumer | I1 |
| Transit-map disconnect → WZ-defined post-transit dock | I2 |
| PQ Leave on disconnect drops warp emit | Unit assert on `Leave` branching by reason; I1 covers behavior |
| Timer ForceReturnIfTracked retires CHANGE_MAP emit | Unit assert on emit absence; I3 covers behavior |
| All affected services build cleanly + existing tests pass | CI |
| New unit tests cover three rules + sentinel + tenant fetch | §8 unit list |
| Integration: KPQ disconnect → relog at lobby | I1 |
