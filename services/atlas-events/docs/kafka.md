# Event Kafka Integration

All consumers use the `Events Service` consumer group and parse the span and tenant headers (`SpanHeaderParser`, `TenantHeaderParser`). All consumers start from `kafka.LastOffset`.

## Topics Consumed

### EVENT_TOPIC_CHARACTER_STATUS

Character status events. Only `LOGIN` is consumed.

**Message Types:**

#### LOGIN

Grants every active `ANNIVERSARY` occurrence's buff to the logging-in character (`anniversary.LoginProcessor.OnLogin`).

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "LOGIN",
  "body": {
    "channelId": 0,
    "mapId": 0,
    "instance": "uuid"
  }
}
```

### EVENT_TOPIC_MONSTER_STATUS

Monster status events. Only `CREATED`, `KILLED`, and `DESTROYED` are consumed; every other type on the topic is ignored. Consumed as a generic envelope with `json.RawMessage` body — only the envelope fields are used.

**Message Types:**

#### CREATED / KILLED / DESTROYED

Tracks a `CRIMSON_BALROG` occurrence's monster set (`crimsonbalrog.MonsterProcessor.OnMonsterStatus`), matching only monsters whose `spawnSourceType` is `EVENT` and whose `spawnSourceId` parses as a UUID belonging to an active `CRIMSON_BALROG` occurrence. `CREATED` records the monster spawned; `KILLED`/`DESTROYED` records it gone and, once every spawned monster for the occurrence is accounted for and none alive, completes the occurrence.

```json
{
  "uniqueId": 0,
  "monsterId": 0,
  "type": "CREATED",
  "spawnSourceType": "EVENT",
  "spawnSourceId": "uuid",
  "body": {}
}
```

### EVENT_TOPIC_TRANSPORT_STATUS

Transport (voyage) status events. `VOYAGE_DEPARTED` and `VOYAGE_ARRIVED` are consumed; both handlers are registered on the same topic and each type-guards independently.

**Message Types:**

#### VOYAGE_DEPARTED

Schedules a delayed `TRIGGER_EVALUATION` work row for every enabled `CRIMSON_BALROG` definition whose `applicableRouteIds` matches the departed route (`crimsonbalrog.TriggerProcessor.OnVoyageDeparted`).

```json
{
  "routeId": "uuid",
  "type": "VOYAGE_DEPARTED",
  "body": {
    "voyageId": "uuid",
    "worldId": 0,
    "channelId": 0,
    "stagingMapId": 0,
    "enRouteMapIds": [0],
    "destinationMapId": 0,
    "observationMapId": 0,
    "departedAt": "2024-01-01T00:00:00Z"
  }
}
```

#### VOYAGE_ARRIVED

Completes and cleans up any still-ACTIVE `CRIMSON_BALROG` occurrence scoped to the arriving voyage (`crimsonbalrog.ArrivalProcessor.OnVoyageArrived`).

```json
{
  "routeId": "uuid",
  "type": "VOYAGE_ARRIVED",
  "body": {
    "voyageId": "uuid",
    "worldId": 0,
    "channelId": 0,
    "stagingMapId": 0,
    "enRouteMapIds": [0],
    "destinationMapId": 0,
    "observationMapId": 0,
    "departedAt": "2024-01-01T00:00:00Z"
  }
}
```

## Topics Produced

### COMMAND_TOPIC_CHARACTER_BUFF

Character-buff commands, produced by `ANNIVERSARY`.

**Message Types:**

#### APPLY

Grants the logging-in character the exp/drop buff for an active ANNIVERSARY occurrence. Keyed by character id.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "APPLY",
  "body": {
    "fromId": 0,
    "sourceId": 0,
    "level": 0,
    "duration": 0,
    "changes": [
      {"type": "expBuffRate", "amount": 200},
      {"type": "itemUpByItem", "amount": 200}
    ],
    "accumulate": false,
    "noExpiry": true,
    "correlationId": "uuid"
  }
}
```

#### CANCEL_BY_CORRELATION

Sweeps every buff tenant-wide whose correlation id matches the completed occurrence's id. Keyed with character id 0 (tenant-wide, not per-character).

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "CANCEL_BY_CORRELATION",
  "body": {
    "correlationId": "uuid"
  }
}
```

### COMMAND_TOPIC_MONSTER

Monster field commands, produced by `CRIMSON_BALROG`. Keyed by map id.

**Message Types:**

#### SPAWN_FIELD

Spawns one monster at a configured position, tagged with the occurrence's provenance.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "SPAWN_FIELD",
  "body": {
    "monsterId": 0,
    "x": 0,
    "y": 0,
    "fh": 0,
    "team": 0,
    "spawnSourceType": "EVENT",
    "spawnSourceId": "uuid"
  }
}
```

#### DESTROY_BY_SOURCE

Despawns every monster in the field carrying the given provenance pair, on occurrence cleanup.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "DESTROY_BY_SOURCE",
  "body": {
    "spawnSourceType": "EVENT",
    "spawnSourceId": "uuid"
  }
}
```

### EVENT_TOPIC_EVENT_VISUAL

Event visual notifications, produced by `CRIMSON_BALROG`. Keyed by map id, so a SHOW and its later HIDE for the same map cannot be reordered. Carries only the visual name and gameplay bytes — this service never constructs a client packet.

**Message Types:**

#### SHOW

Announces the boat-attack visual for a map, on occurrence start.

```json
{
  "occurrenceId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "type": "SHOW",
  "body": {
    "visual": "string",
    "bgm": "string"
  }
}
```

#### HIDE

Retires the boat-attack visual for a map, on occurrence completion.

```json
{
  "occurrenceId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "type": "HIDE",
  "body": {
    "visual": "string"
  }
}
```

## Transaction Semantics

- All produced messages for a single handler call are accumulated in an in-process buffer (`kafka/message.Buffer`) and emitted together via `message.Emit`, grouped by topic.
- `COMMAND_TOPIC_CHARACTER_BUFF` messages for `APPLY` are keyed by character id; `CANCEL_BY_CORRELATION` is keyed with character id 0.
- `COMMAND_TOPIC_MONSTER` and `EVENT_TOPIC_EVENT_VISUAL` messages are keyed by map id, so a map's spawn/visual sequence stays ordered on one partition.
- Kafka redelivery of the same logical scheduling request (`VOYAGE_DEPARTED`, a definition-enable) is absorbed by the `scheduling` work row's dedupe key, not by consumer-side idempotency.
