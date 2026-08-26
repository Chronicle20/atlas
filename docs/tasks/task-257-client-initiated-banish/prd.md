# Client-Initiated Mob Banish — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

MapleStory mobs can eject a player from a map ("banish") through two distinct paths. The
first is a mob **skill** cast — mob skill 129 — which the server drives; Atlas already
implements it in `atlas-monsters` (`monster/processor.go:1224` dispatches to
`executeBanish` at `monster/processor.go:1248`, which reads the WZ `ban` node exposed by
`atlas-data` and emits a portal `WARP` command). The second is **client-initiated**: when a
banish-flagged mob's touch/attack lands, the client itself sends `MOB_BANISH_PLAYER`
(`CUserLocal::SendBanMapByMobRequest`, a single `Encode4(dwMobTemplateID)`) and expects the
server to perform the map change. Atlas decodes and logs that packet but does nothing
(`services/atlas-channel/atlas.com/channel/socket/handler/mob_banish_player.go:19`), so every
mob whose banish rides on its attack rather than on skill 129 silently fails to banish.

This task gives the client-initiated path real behavior, routed through `atlas-monsters` so
both paths converge on one banish implementation rather than growing a second one in
`atlas-channel`. Because the client chooses the mob template id in the request, the server
must not take it on faith: an unvalidated handler is a free warp to any map that appears in
any mob's `ban` node. The command is therefore validated in `atlas-monsters`, which owns
live monster state and can confirm that a monster of that template is actually alive in the
requesting character's field.

The task also closes two fidelity gaps that the existing skill path has: the WZ banish
**portal name** (`ban/banMap/0/portal`) is parsed by `atlas-data` but dropped on the way to
the warp, and the WZ banish **message** (`ban/banMsg`) is never shown to the player. Both
are fixed once, in the shared implementation, so the skill path inherits them.

## 2. Goals

Primary goals:

- Make `MOB_BANISH_PLAYER` warp the requesting character to the mob's WZ-configured banish
  map, replacing the decode-and-log stub.
- Validate the client-supplied mob template id against live field state before honoring it,
  failing closed on any mismatch.
- Honor the WZ banish portal name so the character lands on the configured portal rather
  than a random spawn point.
- Display the WZ `banMsg` to the banished character as pink text.
- Converge the client-initiated and skill-129 banish paths on a single implementation in
  `atlas-monsters`, so portal and message handling apply to both.

Non-goals:

- Rate limiting or abuse throttling of repeated `MOB_BANISH_PLAYER` requests (explicitly out
  of scope; the field/template validation is the only guard in this task).
- Any change to the `MobBanishPlayer` packet codec — it exists, is IDA-derived, and is
  verified (`libs/atlas-packet/character/serverbound/mob_banish_player.go`).
- Server-side detection of "banish on touch" (deciding *when* a banish should fire). The
  client makes that determination and sends the request; the server validates and executes.
- Reading or acting on the WZ `ban/banType` field (see §9).
- Changing monster AI, attack resolution, or damage handling.

## 3. User Stories

- As a player standing in a banish-flagged mob's map, I want to be warped out when that
  mob's attack lands, so the map behaves the way the content intends.
- As a player being banished, I want to land at the map's intended arrival portal, so I do
  not appear at an arbitrary spawn point far from where the content expects me.
- As a player being banished, I want to see the mob's banish message, so I understand why I
  was ejected.
- As a server operator, I do not want a crafted `MOB_BANISH_PLAYER` packet to teleport a
  client to an arbitrary map, so the request must be validated against live field state.
- As a maintainer, I want one banish implementation, so a fix to portal or message handling
  applies to both the skill-cast and attack-triggered paths.

## 4. Functional Requirements

### 4.1 Channel handler

- `MobBanishPlayerHandleFunc` MUST, on decode, emit a `BANISH` command on
  `COMMAND_TOPIC_MONSTER` carrying the session's field (`s.Field()`), the session's
  character id, and the decoded `mobTemplateId`. It MUST NOT resolve monster data, decide
  the target map, or emit a warp itself.
- The handler MUST retain the existing debug log line of the decoded packet.
- The handler MUST NOT return an error path to the client; a request that fails validation
  downstream is silently ignored (matching the client, which does not await a response).

### 4.2 Banish command validation (`atlas-monsters`)

On receiving a `BANISH` command, the processor MUST, in order, and MUST abort (log at debug
or warn, take no action) at the first failure:

1. Resolve live monsters in the command's field via the existing field-scoped lookup
   (`Processor.GetInField`). If the lookup errors, abort.
2. Require at least one live monster in that field whose template id (`Model.MonsterId()`)
   equals the requested `monsterTemplateId`. If none, abort — this is the trust boundary.
3. Fetch monster information for the template. If the fetch errors, abort.
4. Require `Banish().MapId != 0`. If zero (no `ban` node, or an empty one), abort.

Only after all four checks pass does the banish execute.

### 4.3 Banish execution (shared)

A single execution helper in `atlas-monsters` MUST perform the following, and MUST be
called by both the new `BANISH` command handler and the existing `executeBanish`
(skill-129) path:

- Emit a `WARP` command on `COMMAND_TOPIC_PORTAL` for the character, targeting
  `Banish().MapId`, carrying `Banish().PortalName` when it is non-empty.
- When `Banish().Message` is non-empty, emit a `SEND_MESSAGE` command on
  `COMMAND_TOPIC_SYSTEM_MESSAGE` for that character with `messageType: "PINK_TEXT"` and the
  banish message as the body text.
- Order matters: emit the message only after the warp command send returns without error.
  If the warp emit fails, log the error and skip the message, so a player is never told they
  were banished when they were not.

The skill-129 path retains its existing target selection (`getDiseaseTargets`) and calls the
shared helper once per target.

### 4.4 Portal-name resolution (`atlas-portals`)

- The `WARP` command body gains an optional `targetPortalName` string.
- `handleWarpCommand` MUST resolve, in this precedence order: `UseTargetPosition` (existing,
  highest), then `TargetPortalId != 0` (existing), then `targetPortalName != ""` (new), then
  the existing random-spawn `Warp`.
- Name resolution MUST use the existing `Processor.GetInMapByName(targetMapId, name)`. If
  the name does not resolve, the handler MUST fall back to the existing random-spawn `Warp`
  and log a warning — it MUST NOT drop the warp, because failing to banish is worse than
  banishing to a default spawn.
- `"sp"` is a real portal name in map data, not a sentinel; it is resolved by name like any
  other. No special-casing.

### 4.5 Monster data projection (`atlas-channel` / `atlas-monsters`)

- `atlas-monsters`'s `monster/information` model already carries `Banish{Message, MapId,
  PortalName}` — no change needed there.
- `atlas-channel`'s `data/monster` projection (`Model`, `RestModel`) does NOT need banish
  fields under this design, because the channel no longer makes the banish decision. It MUST
  be left unchanged.

## 5. API Surface

No REST endpoints are added, removed, or changed. `atlas-data`'s
`GET /data/monsters/{id}` already serves `banish{message, map_id, portal_name}`
(`services/atlas-data/atlas.com/data/monster/rest.go:85`) and `atlas-monsters` already
consumes it.

Kafka surface changes:

| Topic | Direction | Change |
|---|---|---|
| `COMMAND_TOPIC_MONSTER` | `atlas-channel` → `atlas-monsters` | New command type `BANISH` with body `{characterId, monsterTemplateId}` |
| `COMMAND_TOPIC_PORTAL` | `atlas-monsters` → `atlas-portals` | `WARP` body gains optional `targetPortalName` |
| `COMMAND_TOPIC_SYSTEM_MESSAGE` | `atlas-monsters` → `atlas-channel` | New producer (existing `SEND_MESSAGE` / `PINK_TEXT` contract, no schema change) |

### 5.1 `BANISH` command

Envelope is the existing `command[E]` in
`services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go:36`
(`worldId`, `channelId`, `mapId`, `instance`, `monsterId`, `type`, `body`).

- `type`: `"BANISH"`
- `monsterId` (envelope): `0` — the client supplies a *template* id, not a unique id; the
  unique monster is resolved during validation. The field is left zero rather than
  overloaded.
- Body:

```json
{
  "characterId": 123456,
  "monsterTemplateId": 9500324
}
```

- Partition key: character id (matching the warp/portal command convention in
  `atlas-channel/.../portal/producer.go`), so a character's banish requests stay ordered
  relative to each other.

### 5.2 `WARP` command body addition

```json
{
  "characterId": 123456,
  "targetMapId": 926120410,
  "targetPortalId": 0,
  "targetPortalName": "st00",
  "useTargetPosition": false,
  "targetX": 0,
  "targetY": 0
}
```

`targetPortalName` is optional and omitted (empty) by every existing producer, so the field
is additive and backward compatible. The field must be added to `atlas-portals`'s
`warpBody` (`portal/kafka.go:41`) and to `atlas-monsters`'s local `warpBody`
(`monster/disease.go:96`). `atlas-channel`'s local `WarpBody` copy
(`kafka/message/portal/kafka.go:40`) is NOT required to change under this design; leave it
alone unless a channel producer needs it.

## 6. Data Model

No database entities, tables, or migrations. All banish data is read-only WZ content served
by `atlas-data`.

WZ shape (grounded in `Mob.wz`, verified against both the v83 and the 1172 data sets):

```
<imgdir name="ban">
  <int name="banType" value="1"/>              <!-- present in 10 of 26 nodes; always 1 -->
  <string name="banMsg" value="..."/>           <!-- optional -->
  <imgdir name="banMap">
    <imgdir name="0">
      <int name="field" value="926120410"/>
      <string name="portal" value="st00"/>      <!-- optional -->
    </imgdir>
  </imgdir>
</imgdir>
```

Findings from the WZ survey that constrain this design:

- **No `999999999` sentinel exists.** Every `banMap/0/field` in both data sets is a real map
  id. "Banish to previous map" is not a case this task must handle. A missing or zero map id
  is simply a no-op.
- **`banMap` never has an index beyond `0`** in either data set, so `atlas-data`'s
  `banMap/0`-only read (`monster/reader.go:120`) is complete.
- **Non-`sp` portals are common**: of 19 portal entries, 11 are `sp`, and 8 are real named
  portals (`out00` ×3, `st00` ×3, `in02`, `top00`). Dropping the portal name is therefore a
  user-visible defect, not a theoretical one.
- **Three nodes have the WZ typo `potal` instead of `portal`** (e.g. `9500194.img.xml`).
  `getBanish` reads `banMap/0/portal` with a `"sp"` default, so these silently become `sp`.
  That is acceptable behavior (a spawn-point landing) and this task does NOT add typo
  tolerance; it is recorded here so the behavior is intentional rather than accidental.

## 7. Service Impact

**`atlas-channel`**

- `socket/handler/mob_banish_player.go`: replace the deferred-behavior comment with a
  `BANISH` command emit; the handler needs the writer/producer plumbing the other
  command-emitting handlers use.
- New producer for the `BANISH` command in the channel's monster command message package.
- `data/monster` unchanged.

**`atlas-monsters`**

- `kafka/consumer/monster/kafka.go`: add `CommandTypeBanish = "BANISH"` and a
  `banishCommandBody`. Note the existing caution in that file (every handler on this shared
  topic unmarshals every message) — keep the body's field types wide enough that a large
  template id cannot overflow a sibling body's narrow field, and confirm no sibling body has
  a narrow field colliding on the `monsterTemplateId` JSON key.
- `kafka/consumer/monster/consumer.go`: register `handleBanishCommand`.
- `monster/processor.go`: add the validated `Banish(f, characterId, monsterTemplateId)`
  entry point; refactor `executeBanish` (line 1248) to call the shared execution helper.
- `monster/disease.go`: extend `warpBody`/`warpCommandProvider` with the portal name.
- New local `kafka/message/system_message` package (mirroring the existing local copies in
  `atlas-party-quests` and `atlas-saga-orchestrator`) plus a producer for the `SEND_MESSAGE`
  / `PINK_TEXT` command.

**`atlas-portals`**

- `portal/kafka.go`: add `TargetPortalName` to `warpBody`.
- `portal/consumer.go`: resolve the name in `handleWarpCommand` with the documented
  precedence and warn-and-fall-back behavior.

**`atlas-data`** — unchanged (already serves the banish payload).

## 8. Non-Functional Requirements

- **Security / trust**: the mob template id is client-controlled input. The field+template
  liveness check in §4.2 is the security boundary and MUST fail closed. No banish may be
  executed on the strength of the packet alone.
- **Multi-tenancy**: all new Kafka commands carry the tenant through the existing header
  parsers; the `atlas-channel` system-message consumer already enforces tenant and
  world/channel matching before announcing, and the new producer inherits that.
- **Observability**: each abort branch in §4.2 logs with the character id, the requested
  template id, and the field, so a failed banish is diagnosable from logs alone without
  reproducing it. A successful banish logs the resolved map and portal.
- **Performance**: validation costs one field-scoped monster lookup plus one cached monster
  information fetch per request, on a packet that fires at most once per banish event. No
  new hot path.
- **Backward compatibility**: `targetPortalName` is additive; existing `WARP` producers that
  omit it retain byte-for-byte identical behavior.

## 9. Open Questions

- **`ban/banType`**: present on 10 of 26 nodes, always `1`, and currently unread by
  `atlas-data`. Its semantics are unverified against the client. Deliberately out of scope
  for this task; if a later finding shows it selects a banish variant, it becomes its own
  task rather than a guess here.
- **Message ordering**: the pink-text message and the map change are two independent Kafka
  commands to two different services, so the client may render the message either just
  before or just after the field transition. Cosmic sends both in the same tick and does not
  guarantee order either. Accepted as-is unless live testing shows the message is lost
  across the transition — in which case the fix is to sequence the message *before* the warp
  emit, which §4.3 already does.
- **`atlas-channel`'s local `WarpBody` copy**: left unchanged. If a future channel producer
  needs portal-name warps, it will need the same additive field.

## 10. Acceptance Criteria

- [ ] `MOB_BANISH_PLAYER` no longer contains a deferred-behavior stub; the handler emits a
      `BANISH` command with the session field, character id, and decoded template id.
- [ ] `atlas-monsters` handles `BANISH` and warps the character to the WZ banish map when,
      and only when, a live monster of that template exists in the character's field and the
      template has a non-zero banish map.
- [ ] A `BANISH` request naming a template with no live instance in the character's field
      produces no warp, and logs the rejection with character id, template id, and field.
- [ ] A `BANISH` request naming a template with no `ban` node (or a zero banish map)
      produces no warp.
- [ ] When the WZ banish node specifies a portal name, the emitted `WARP` carries
      `targetPortalName`, and `atlas-portals` lands the character on the portal resolved by
      that name.
- [ ] When the portal name does not resolve in the target map, `atlas-portals` warps the
      character via the existing random-spawn path and logs a warning; the warp is not
      dropped.
- [ ] When the WZ banish node has a non-empty `banMsg`, the banished character receives it
      as `PINK_TEXT` via `COMMAND_TOPIC_SYSTEM_MESSAGE`; when it is empty, no message is
      sent.
- [ ] The skill-129 path (`executeBanish`) routes through the same shared execution helper
      and therefore also honors the portal name and emits the banish message.
- [ ] Unit tests cover: each of the four validation aborts; the portal-name-present and
      portal-name-absent warp bodies; the message-present and message-absent branches; and
      `atlas-portals`'s name-resolution precedence including the fallback-on-miss.
- [ ] No existing `WARP` producer's emitted body changes when `targetPortalName` is unset.
- [ ] `tools/verify.sh` exits 0 (flagless).
