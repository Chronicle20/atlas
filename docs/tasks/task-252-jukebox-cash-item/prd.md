# Jukebox Cash Item — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

Using a jukebox (song player) cash item currently does nothing. The client sends
`CWvsContext::SendConsumeCashItemUseRequest`, `CharacterCashItemUseHandleFunc`
classifies the item as cash slot type 20 (`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1319-1321`,
the `item.ClassificationSongPlayer` branch of `GetCashSlotItemType`), and then —
because no arm matches type 20 — falls through to the warn-and-drop at
`character_cash_item_use.go:952`. The item is not consumed and nothing is
broadcast. Cosmic's reference implementation (`UseCashItemHandler.java:392`)
broadcasts a music change to the map on the same item use.

Every piece of plumbing this feature needs already exists, built for the
analogous weather cash item (`CashSlotItemTypeFieldEffect`, enum 16):

- The clientbound codec `PlayJukebox` (`libs/atlas-packet/field/clientbound/play_jukebox.go`,
  `CField::OnPlayJukeBox`) is written, unit-tested, evidence-pinned for
  `gms_v95` and `jms_v185`, registered as a writer at
  `services/atlas-channel/atlas.com/channel/main.go:842`, and routed in all nine
  seed templates. Nothing in the repository invokes it — it is a writer-only
  scaffold.
- The BGM change mechanism exists as `FieldEffectBackgroundMusicBody(name)`
  (`libs/atlas-packet/field/field_effect_body.go:63`), already used for weather
  items, doors, and events.
- The command path — channel handler → saga (`DestroyAsset` + a map action) →
  `atlas-saga-orchestrator/map_command` → `atlas-maps` (registry state with
  expiry) → map status event → `atlas-channel` map consumer broadcast, plus
  on-enter replay — is fully built for weather and is the template this feature
  follows.

This task wires the type-20 arm end to end: consume the item, record the playing
song as durable per-field state in `atlas-maps`, broadcast `PlayJukebox` and the
BGM field effect to everyone in the map, replay both to players who enter mid-
song, and stop the song when it expires.

## 2. Goals

Primary goals:

- Using a classification-510 cash item consumes exactly one of that item and
  changes the background music for every character in the user's field.
- The song is durable field state: a character entering the field while a song
  is playing hears it and sees the jukebox state, exactly as weather replays
  today.
- The song stops on expiry, with the client returned to the map's own BGM.
- The BGM to play is resolved from the item's own WZ data (`info/path`), so any
  classification-510 item works without a code change.
- `PlayJukebox` stops being a writer-only scaffold.

Non-goals:

- Purchasing jukebox items in the cash shop (unrelated cash-shop flow).
- Per-item duration data. The WZ corpus carries none (see §6.1); duration is a
  server constant.
- Any change to non-cash BGM sources — event BGM
  (`kafka/consumer/event/consumer.go:82`) and door BGM
  (`kafka/consumer/map/consumer.go:789`) are untouched.
- Cross-channel or world-wide broadcast. Scope is one field (world + channel +
  map + instance), matching weather.
- A player-facing "stop my jukebox" command.

## 3. User Stories

- As a player, I want to use a jukebox cash item so that the music in my map
  changes for everyone present.
- As a player already in the map, I want to hear the new song as soon as another
  player uses their jukebox, without re-entering the map.
- As a player entering a map where a song is playing, I want to hear that song
  rather than the map's default music.
- As a player, I want the music to return to the map's default when the song
  ends, so that a stale song does not persist indefinitely.
- As a player, I want my jukebox item consumed exactly once per successful use,
  and not consumed at all when the use is rejected.

## 4. Functional Requirements

### 4.1 Item-use arm (atlas-channel)

- **FR-1.1** `CharacterCashItemUseHandleFunc` gains an arm for cash slot type 20
  (`CashSlotItemTypeSongPlayer`), added to the `CashSlotItemType` constant block
  in `character_cash_item_use.go` alongside the existing named types.
- **FR-1.2** The arm runs after the existing slot/template ownership check
  (`cashItemInSlotFunc`), so a request naming a slot that does not hold the
  claimed template is rejected before any state changes — unchanged behaviour,
  inherited from the shared preamble.
- **FR-1.3** The arm decodes the type-20 sub-body per §9 OQ-1. On versions where
  `update_time` trails the sub-body (`updateTimeFirst == false`, i.e. GMS v83
  and v84), the trailing value replaces the header-decoded `updateTime`, exactly
  as the morph-coupon arm does (`character_cash_item_use.go:944-948`).
- **FR-1.4** The arm resolves the item's BGM name from the cash item data
  service (§4.2). If the resolved name is empty, the arm logs and returns
  **without consuming the item and without broadcasting**.
- **FR-1.5** On success, the arm creates a saga of type `FieldEffectUse`
  (`InitiatedBy: "CASH_ITEM_USE"`) with two steps, in order:
  1. `DestroyAsset` — `CharacterId`, `TemplateId` = the used item, `Quantity: 1`,
     `RemoveAll: false`.
  2. `PlayJukebox` (new action, §4.3) — carrying `WorldId`, `ChannelId`,
     `MapId`, `Instance`, `ItemId`, `BgmPath`, `PlayerName`, `DurationMs`.
- **FR-1.6** The arm sends no `EnableActions`. The non-silent inventory operation
  emitted by the consume commit already clears the client's exclusive-request
  lock, matching the reasoning recorded for the field-effect and morph-coupon
  arms.
- **FR-1.7** The arm gates on the cash slot **type** (20), not on a hard-coded
  item id, so any classification-510 item routes here.

### 4.2 BGM resolution (atlas-data, atlas-channel)

- **FR-2.1** `atlas-data`'s cash reader parses `info/path` from the cash item WZ
  node into a new field on the cash `RestModel`. The existing `BgmPath` field is
  **not** reused: `reader.go:93-95` only populates it when
  `info/isBgmOrEffect == 1`, and the jukebox node carries neither
  `isBgmOrEffect` nor `bgmPath` (see §6.1).
- **FR-2.2** The value the client expects is the WZ path with the `Sound/` prefix
  and the `.img` suffix removed: `"Sound/Jukebox.img/Congratulation"` →
  `"Jukebox/Congratulation"`. This matches the string Cosmic sends
  (`musicChange("Jukebox/Congratulation")`). Where this normalisation lives
  (data service vs. channel) is a design-phase decision; it must be applied
  exactly once.
- **FR-2.3** An item whose node has no `info/path` yields an empty value, which
  FR-1.4 turns into a no-op rejection.
- **FR-2.4** `atlas-channel`'s cash data REST model
  (`services/atlas-channel/atlas.com/channel/data/cash/rest.go`) gains the
  corresponding field.

### 4.3 Saga action (libs/atlas-saga, atlas-saga-orchestrator)

- **FR-3.1** A new saga `Action` (e.g. `play_jukebox`) and its payload struct are
  added to `libs/atlas-saga` (`model.go`, `payloads.go`), with an `unmarshal.go`
  case mirroring `FieldEffectWeather` (`unmarshal.go:606`).
- **FR-3.2** `atlas-saga-orchestrator`'s handler gains a `handlePlayJukebox`
  method registered in the action switch (`handler.go:1003`), which validates the
  payload, builds the `field.Model`, and calls a new `map_command` processor
  method.
- **FR-3.3** `map_command.Processor` gains `PlayJukebox(transactionId, f,
  itemId, bgmPath, playerName, durationMs)`, producing a `PLAY_JUKEBOX` command
  onto `EnvCommandTopicMap`, mirroring `FieldEffectWeather`
  (`map_command/processor.go:33`).
- **FR-3.4** Step failure semantics follow the existing saga machinery: if
  `DestroyAsset` fails, the jukebox step does not run.

### 4.4 Field state and lifecycle (atlas-maps)

- **FR-4.1** `atlas-maps` consumes the `PLAY_JUKEBOX` command and records a
  jukebox entry in a per-field registry keyed by `{Tenant, Field}`, mirroring
  `map/weather/registry.go` and `processor.go`. The entry holds `ItemId`,
  `BgmPath`, `PlayerName`, and `ExpiresAt`.
- **FR-4.2** Duration is a server-side constant defined in `atlas-maps`
  (weather's precedent: `maxWeatherDuration` in
  `kafka/consumer/map/consumer.go:51`). A command carrying a longer duration is
  capped at the constant and the cap is logged, exactly as weather does.
- **FR-4.3** Starting a jukebox in a field that already has an active one
  **replaces** the entry; the new song takes effect immediately for everyone in
  the field, and the replaced entry's expiry no longer applies.
- **FR-4.4** On start, `atlas-maps` emits a `JUKEBOX_START` status event on
  `EnvEventTopicMapStatus` carrying `ItemId`, `BgmPath`, and `PlayerName`.
- **FR-4.5** On expiry, `atlas-maps` emits a `JUKEBOX_END` status event carrying
  the `ItemId` that ended, mirroring `WeatherEndEventProvider`
  (`map/weather/producer.go:31`).
- **FR-4.6** `atlas-maps` exposes the active entry over REST at
  `/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/jukebox`,
  returning `404` when no song is playing — mirroring the weather resource
  (`map/weather/resource.go`).

### 4.5 Broadcast and replay (atlas-channel)

- **FR-5.1** On `JUKEBOX_START`, the map consumer broadcasts to every session in
  the field (`_map.NewProcessor(...).ForSessionsInMap`):
  1. `PlayJukebox(itemId, playerName)` via `fieldcb.PlayJukeboxWriter`.
  2. `FieldEffect` with `FieldEffectBackgroundMusicBody(bgmPath)`.
- **FR-5.2** On `JUKEBOX_END`, the map consumer broadcasts `PlayJukebox` with a
  **negative** item id and no player name — the codec's documented stop signal
  (`play_jukebox.go`: the trailing name is written only when `itemId >= 0`) —
  followed by a `FieldEffect` BGM restoring the field's own music (design phase
  determines the restore source; the door/event BGM call sites are the
  precedent).
- **FR-5.3** On character map-enter, the consumer queries `atlas-maps` for an
  active jukebox and, if present, announces `PlayJukebox` + the BGM field effect
  to that session alone — the same `routine.Go` block shape used for weather at
  `kafka/consumer/map/consumer.go:346-361`.
- **FR-5.4** A new `atlas-channel` `jukebox` package provides the REST client for
  FR-4.6, mirroring `channel/weather/` (`processor.go`, `requests.go`,
  `rest.go`, `mock/`).

## 5. API Surface

New endpoint on **atlas-maps**:

```
GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/jukebox
```

JSON:API response, resource type `jukebox`:

```json
{
  "data": {
    "type": "jukebox",
    "id": "5100000",
    "attributes": {
      "itemId": 5100000,
      "bgmPath": "Jukebox/Congratulation",
      "playerName": "Chronicle"
    }
  }
}
```

- `200` — a song is playing in that field.
- `404` — no active song (no body), matching `handleGetWeatherInMap`.
- Tenant scoping comes from the request context, as with every other atlas-maps
  resource; the registry key is `{Tenant, Field}`.

Modified endpoint on **atlas-data**: the cash item resource gains one attribute
carrying the WZ `info/path` value (FR-2.1). Existing attributes are unchanged and
the field is omitted when absent.

No new or modified endpoints in atlas-channel or atlas-saga-orchestrator.

## 6. Data Model

### 6.1 Source data (read-only, WZ)

Verified against all three GMS 83.1 extractions present under `tmp/`
(`Item.wz/Cash/0510.img.xml`). The corpus contains exactly one
classification-510 item:

```xml
<imgdir name="05100000">
  <imgdir name="info">
    <int name="cash" value="1"/>
    <string name="path" value="Sound/Jukebox.img/Congratulation"/>
  </imgdir>
</imgdir>
```

The referenced sound (`Sound.wz/Jukebox.img.xml`) is a bare
`<sound name="Congratulation"/>` with no exported length. There is therefore
**no duration in the data** — hence FR-4.2's server constant — and no
`bgmPath`/`isBgmOrEffect`, hence FR-2.1's new field rather than reuse of
`BgmPath`.

### 6.2 In-memory state (atlas-maps)

A jukebox registry mirroring `map/weather/registry.go`: process-local, keyed by
`FieldKey{Tenant, Field}`, one entry per field.

| Field | Type | Notes |
|---|---|---|
| `ItemId` | `uint32` | The cash item used |
| `BgmPath` | `string` | Client-form BGM name (FR-2.2) |
| `PlayerName` | `string` | Who started it; carried in `PlayJukebox` |
| `ExpiresAt` | `time.Time` | Start + capped duration |

No persistence and no migration: this is ephemeral field state, exactly like
weather. A service restart clears active songs, which is acceptable for a
short-lived cosmetic effect.

### 6.3 Kafka messages

- Command `PLAY_JUKEBOX` on `EnvCommandTopicMap` — body: `ItemId`, `BgmPath`,
  `PlayerName`, `DurationMs`; envelope carries `TransactionId`, `WorldId`,
  `ChannelId`, `MapId`, `Instance`.
- Status events `JUKEBOX_START` / `JUKEBOX_END` on `EnvEventTopicMapStatus`,
  following `StatusEvent[T]` as `WEATHER_START`/`WEATHER_END` do. The message
  types are declared in each service's own `kafka/message/map` package, per the
  existing per-service duplication convention.

## 7. Service Impact

| Service | Change |
|---|---|
| **atlas-data** | Parse `info/path` for cash items; expose it on the cash REST model (FR-2.1). |
| **atlas-channel** | New type-20 arm in `character_cash_item_use.go` incl. `CashSlotItemTypeSongPlayer`; new `jukebox` REST-client package; `JUKEBOX_START`/`JUKEBOX_END` handlers and the on-enter replay block in `kafka/consumer/map/consumer.go`; new field on the cash data REST model. |
| **libs/atlas-saga** | New `play_jukebox` action, payload struct, and unmarshal case. |
| **atlas-saga-orchestrator** | `handlePlayJukebox` + action-switch registration; `map_command.Processor.PlayJukebox` and its command provider. |
| **atlas-maps** | `PLAY_JUKEBOX` command consumer; jukebox registry/processor with expiry; `JUKEBOX_START`/`JUKEBOX_END` producers; the `/jukebox` REST resource. |
| **libs/atlas-packet** | Only if OQ-1 resolves to a non-empty type-20 sub-body: a new `cash/serverbound/item_use_song_player.go`. The clientbound `PlayJukebox` codec needs no change. |
| **libs/atlas-constants** | None expected — `ClassificationSongPlayer` (510) already exists at `item/constants.go:91`. |
| **atlas-ui** | None. |

## 8. Non-Functional Requirements

- **Multi-tenancy**: every registry key, Kafka message, and REST lookup is
  tenant-scoped. A song started in tenant A must never be visible to tenant B.
  The registry key is `{Tenant, Field}`, matching weather.
- **Ownership**: the arm never trusts the wire for the item identity. The
  template is resolved from the character's own cash inventory slot via
  `cashItemInSlotFunc` and compared against the claimed id before any state
  change (inherited from the shared preamble).
- **Bounded effect**: duration is capped server-side (FR-4.2), so a crafted or
  buggy command cannot pin a field's BGM indefinitely.
- **Blast radius**: broadcast is limited to sessions in the one field. No
  world-wide or cross-channel fan-out.
- **Observability**: log at debug on start/replace/expire with tenant, field,
  item id, and duration; log at warn on rejection (unresolvable BGM, capped
  duration), matching the weather consumer's logging.
- **No wire change to verified versions**: `PlayJukebox` is already evidence-
  pinned for `gms_v95` and `jms_v185`. This task must not alter its encoding.
- **Failure isolation**: a missing or erroring `atlas-maps` jukebox lookup on
  map-enter must not block the rest of the enter sequence — the replay runs in
  its own `routine.Go` block and returns silently on error, as weather does.

## 9. Open Questions

- **OQ-1 (design phase, IDA)**: does the type-20 arm of
  `CWvsContext::SendConsumeCashItemUseRequest` encode a sub-body, or is it empty
  apart from the trailing `update_time` on GMS v83/v84? No
  `cash/serverbound/item_use_song_player.go` exists today. Must be derived from
  the GMS v95.1 IDB (and cross-checked on v83) before FR-1.3 is implemented; the
  morph-coupon arm (`character_cash_item_use.go:930-950`) is the empty-sub-body
  precedent to compare against.
- **OQ-2 (design phase)**: what exactly restores the map's own BGM on
  `JUKEBOX_END` (FR-5.2)? Candidates are re-sending the field's configured BGM
  from map data, or relying on the client to restore it after the stop signal.
  Must be confirmed against `CField::OnPlayJukeBox`'s negative-id path rather
  than assumed.
- **OQ-3**: what value should the duration constant take? Weather uses 20s;
  a song is plausibly longer. Needs a decision (design phase) — it is a single
  named constant either way.
- **OQ-4**: should `PlayJukebox` be broadcast to the whole field or only to
  other players (with the user's own client already reacting locally)? The
  weather precedent broadcasts to everyone including the user; confirm the
  client does not double-apply.

## 10. Acceptance Criteria

- [ ] Using item `05100000` in a map consumes exactly one of that item from the
      cash inventory and leaves the rest of the inventory untouched.
- [ ] Every character in that world/channel/map/instance receives `PlayJukebox`
      with the used item id and the user's character name, followed by a
      `FieldEffect` BGM packet carrying `Jukebox/Congratulation`.
- [ ] No character in a different map, channel, instance, or tenant receives
      either packet.
- [ ] A character entering the map while the song is active receives both
      packets for that session alone.
- [ ] When the duration elapses, every character in the field receives
      `PlayJukebox` with a negative item id and the map's own BGM is restored.
- [ ] Using a second jukebox while one is active replaces the song immediately
      for everyone in the field.
- [ ] A classification-510 item whose WZ node has no `info/path` is rejected:
      nothing is consumed and nothing is broadcast.
- [ ] A use request naming a slot that does not hold the claimed template is
      rejected with no consumption and no broadcast.
- [ ] `GET .../instances/{id}/jukebox` returns the active entry, and `404` when
      nothing is playing.
- [ ] `grep -rn "PlayJukeboxWriter" services/` shows at least one invoking call
      site (the writer is no longer scaffold-only).
- [ ] Unit tests cover: the type-20 arm's saga emission and its two rejection
      paths (handler, with the existing package-var test seams — no live Kafka);
      the WZ `info/path` parse and its normalisation to client form; the
      atlas-maps registry start/replace/expire transitions and the duration cap;
      the `JUKEBOX_START`/`JUKEBOX_END` consumer broadcasts.
- [ ] Existing `PlayJukebox` codec tests and its pinned evidence records for
      `gms_v95` and `jms_v185` are unchanged and still pass.
- [ ] Flagless `tools/verify.sh` exits 0.
