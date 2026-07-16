# Kafka

## Topics Consumed

### EVENT_TOPIC_CHARACTER_STATUS

Character status event topic.

| Message Type | Direction | Description |
|--------------|-----------|-------------|
| DELETED | Event | Character was deleted; triggers cascading delete of the character's cards and collection row |

#### Event Structure

```json
{
  "transactionId": "uuid",
  "worldId": "byte",
  "characterId": "uint32",
  "type": "DELETED",
  "body": {}
}
```

### COMMAND_TOPIC_MONSTER_BOOK

Monster-book command topic.

| Message Type | Direction | Description |
|--------------|-----------|-------------|
| CARD_PICKED_UP | Command | A character picked up (or re-picked-up) a monster card |
| SET_COVER | Command | Set (or clear) a character's monster-book cover card |

#### Command Structure

```json
{
  "tenantId": "uuid",
  "characterId": "uint32",
  "eventId": "uuid",
  "type": "CARD_PICKED_UP",
  "body": {
    "cardId": "uint32",
    "source": "string"
  }
}
```

```json
{
  "tenantId": "uuid",
  "characterId": "uint32",
  "eventId": "uuid",
  "type": "SET_COVER",
  "body": {
    "coverCardId": "uint32"
  }
}
```

## Topics Produced

### EVENT_TOPIC_MONSTER_BOOK_STATUS

Monster-book status event topic.

| Message Type | Direction | Description |
|--------------|-----------|-------------|
| CARD_ADDED | Event | A card was inserted or leveled up |
| COVER_CHANGED | Event | The collection's cover card changed |
| STATS_CHANGED | Event | Collection stats (book level, counts, EXP bonus) changed |

#### Event Structure

```json
{
  "tenantId": "uuid",
  "characterId": "uint32",
  "eventId": "uuid",
  "type": "CARD_ADDED",
  "body": {
    "cardId": "uint32",
    "newLevel": "uint8",
    "full": "bool"
  }
}
```

```json
{
  "tenantId": "uuid",
  "characterId": "uint32",
  "eventId": "uuid",
  "type": "COVER_CHANGED",
  "body": {
    "coverCardId": "uint32"
  }
}
```

```json
{
  "tenantId": "uuid",
  "characterId": "uint32",
  "eventId": "uuid",
  "type": "STATS_CHANGED",
  "body": {
    "bookLevel": "uint16",
    "normalCount": "uint16",
    "specialCount": "uint16",
    "totalUniqueCards": "uint16",
    "expBonusPercent": "uint16"
  }
}
```

### EVENT_TOPIC_CHARACTER_STATUS

Character status event topic. This service produces an EXPERIENCE_CHANGED event on this topic (in addition to consuming DELETED events from it, above) whenever collection stats change and expBonusPercent is nonzero.

| Message Type | Direction | Description |
|--------------|-----------|-------------|
| EXPERIENCE_CHANGED | Event | Monster-book EXP bonus distribution, reported as a character experience distribution |

#### Event Structure

```json
{
  "characterId": "uint32",
  "type": "EXPERIENCE_CHANGED",
  "body": {
    "distributions": [
      {
        "experienceType": "MONSTER_BOOK",
        "amount": "uint32",
        "attr1": "uint32"
      }
    ]
  }
}
```

## Message Types

### Character Messages

| Struct | Purpose |
|--------|---------|
| character.StatusEvent[E] | Generic character status envelope (consumed) |
| character.DeletedStatusEventBody | Body for DELETED status event (consumed) |
| experienceStatusEvent (unexported) | Character EXPERIENCE_CHANGED envelope subset (produced); mirrors the fields atlas-channel's consumer reads from character.StatusEvent[ExperienceChangedStatusEventBody] |
| experienceChangedBody (unexported) | Body for the produced EXPERIENCE_CHANGED event: a list of experienceDistribution |
| experienceDistribution (unexported) | One EXP distribution entry (experienceType, amount, attr1) |

### Monster-Book Messages

| Struct | Purpose |
|--------|---------|
| monsterbook.Command[B] | Generic monster-book command envelope (consumed) |
| monsterbook.CardPickedUpBody | Body for CARD_PICKED_UP command |
| monsterbook.SetCoverBody | Body for SET_COVER command |
| monsterbook.StatusEvent[B] | Generic monster-book status envelope (produced) |
| monsterbook.CardAddedBody | Body for CARD_ADDED status event |
| monsterbook.CoverChangedBody | Body for COVER_CHANGED status event |
| monsterbook.StatsChangedBody | Body for STATS_CHANGED status event |

## Transaction Semantics

- Outbound messages produced in response to a command are buffered via `message.Buffer` and enqueued to the transactional outbox (`atlas-outbox` library) within the same database transaction as the triggering write; a separate background drainer (`outboxlib.NewDrainer`, started in `main.go`) asynchronously publishes buffered messages to Kafka.
- `card.Add` is idempotent per (tenantId, characterId, cardId, eventId): a CARD_PICKED_UP command carrying an eventId already stored as the card's lastEventId is a no-op and does not buffer a CARD_ADDED event.
- `collection.SetCoverAndEmit` is idempotent per (tenantId, characterId, eventId): a SET_COVER command carrying an eventId already stored as the collection's lastCoverEventId does not update the row or buffer a COVER_CHANGED event.
- `collection.RecomputeAndEmit` only buffers STATS_CHANGED and EXPERIENCE_CHANGED events when the recomputed stats differ from the previously stored stats.
- CARD_PICKED_UP handling runs card.Add and, if the card was newly inserted, collection.RecomputeAndEmit within one database transaction and one message buffer, so CARD_ADDED, STATS_CHANGED, and EXPERIENCE_CHANGED (when applicable) are enqueued to the outbox atomically together.
- The character DELETED handler deletes the character's cards and collection row within one database transaction; it does not emit any Kafka message.
- Monster-book status events and the produced EXPERIENCE_CHANGED event are partitioned by characterId (`kafkaProducer.CreateKey(int(characterId))`).
