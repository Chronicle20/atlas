# Character Deletion Audit - Context Document

**Last Updated: 2026-02-03**

## Key Files Reference

### Character Status Event Definitions

| File | Purpose |
|------|---------|
| `services/atlas-character/atlas.com/character/kafka/producer/status/producer.go` | Emits character status events including DELETED |
| `services/atlas-character/atlas.com/character/event.go` | Defines `StatusEvent` and `StatusEventType` constants |

### Working Implementation Examples

These services correctly handle character deletion. Use as reference patterns:

| Service | Consumer File | Handler Function |
|---------|--------------|------------------|
| atlas-buddies | `atlas.com/buddies/kafka/consumer/character/consumer.go` | `handleStatusEventDeleted` |
| atlas-inventory | `atlas.com/inventory/kafka/consumer/character/consumer.go` | `handleStatusEventDeleted` |
| atlas-pets | `atlas.com/pets/kafka/consumer/character/consumer.go` | `handleCharacterDeleted` |
| atlas-marriages | `atlas.com/marriages/kafka/consumer/character/consumer.go` | `handleStatusEventDeleted` |
| atlas-skills | `atlas.com/skills/kafka/consumer/character/consumer.go` | `handleStatusEventDeleted` |
| atlas-keys | `atlas.com/keys/kafka/consumer/character/consumer.go` | `handleStatusEventDeleted` |
| atlas-notes | `atlas.com/notes/kafka/consumer/character/consumer.go` | `handleStatusEventDeleted` |
| atlas-cashshop | `atlas.com/cashshop/kafka/consumer/character/consumer.go` | `handleStatusEventDeleted` |
| atlas-parties | `atlas.com/parties/kafka/consumer/character/consumer.go` | `handleStatusEventDeleted` |

### Services Needing Implementation

| Service | Consumer Location | Existing Cleanup Method | Event to Emit |
|---------|------------------|------------------------|---------------|
| atlas-quest | `atlas.com/quest/kafka/consumer/character/consumer.go` | `deleteByCharacterIdWithProgress()` in `quest/administrator.go` | Audit events |
| atlas-guilds | NEEDS NEW CONSUMER | `LeaveAndEmit(guildId, characterId, true, transactionId)` in `guild/processor.go` | `MEMBER_LEFT` (already handled) |
| atlas-families | NEEDS NEW CONSUMER | `RemoveMemberAndEmit(transactionId, characterId, reason)` in `family/processor.go` | `LinkBrokenEvent` (already handled) |
| atlas-fame | `atlas.com/fame/kafka/consumer/character/consumer.go` | NEEDS VERIFICATION | Fame deletion events |

**Note:** atlas-rates and atlas-messengers do NOT need implementation because characters must be logged out before deletion, and both services already handle LOGOUT events to clean up in-memory state.

## Key Decisions

### Decision 1: Ephemeral Services Excluded
Services storing only ephemeral session data (atlas-channel, atlas-maps, atlas-chairs, etc.) do not need deletion handlers because:
- Data is cleared on logout (which happens before deletion)
- Data is non-persistent (lost on service restart)
- No database orphan risk

### Decision 2: Account-scoped Services Excluded
`atlas-storage` stores data keyed by `account_id`, not `character_id`. Character deletion does not affect storage data.

### Decision 3: atlas-families Requires Special Handling
The family system has complex referential relationships:
- `senior_id` points to parent character
- `junior_ids` array contains children character IDs
Simple deletion of the character's own record is insufficient. Must update related records.

### Decision 4: In-Memory Services Do NOT Need Deletion Handlers
atlas-rates and atlas-messengers were originally considered but excluded because:
- A character MUST be logged out before it can be deleted
- Both services already handle LOGOUT events to clean up in-memory state
- By the time the DELETED event arrives, the in-memory entries are already gone
- No memory leak risk exists

### Decision 5: Social Notification is Critical
For guilds and families, it's not just about data cleanup - other online players need to be notified:
- **Guilds:** Emit `MEMBER_LEFT` event with `Force: true` so guild members' clients update their roster
- **Families:** `RemoveMemberAndEmit` already emits `LinkBrokenEvent` to notify senior/juniors

## Dependencies

### Kafka Topics
- `CHARACTER_STATUS` - Character status events including DELETED
- Consumer group per service (e.g., `atlas-quest-character-status-consumer`)

### Database Entities Affected

**atlas-quest:**
- `quest_statuses` (character_id indexed)
- `quest_progress` (via quest_status_id FK)

**atlas-guilds:**
- `members` (character_id primary key)
- `characters` (character_id primary key)

**atlas-families:**
- `family_members` (character_id unique)
  - `senior_id` (nullable, references another character)
  - `junior_ids` (JSON array of character IDs)

**atlas-fame:**
- Fame records with character_id

## Testing Strategy

### Unit Tests
- Test deletion handler processes DELETED event type correctly
- Test deletion handler ignores non-DELETED event types
- Test error handling when character data not found

### Integration Tests
- Create character with service data
- Trigger character deletion
- Verify no orphaned records remain
- Verify events emitted correctly

### atlas-families Specific Tests
- Character with no family membership
- Character as junior only (has senior)
- Character as senior only (has juniors)
- Character as both senior and junior
- Family dissolution when last member deleted

## Event Types Reference

```go
const (
    StatusEventTypeDeleted        StatusEventType = "DELETED"
    StatusEventTypeCreated        StatusEventType = "CREATED"
    StatusEventTypeLogin          StatusEventType = "LOGIN"
    StatusEventTypeLogout         StatusEventType = "LOGOUT"
    StatusEventTypeMapChanged     StatusEventType = "MAP_CHANGED"
    StatusEventTypeChannelChanged StatusEventType = "CHANNEL_CHANGED"
    // ... other types
)
```

## Consumer Registration Pattern

```go
func InitConsumers(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) func() {
    cg := kafka.NewConsumerGroupId(consumerGroupId)

    return consumer.Create(l)(ctx)(cg)(character.StatusEventTopic())(
        consumer.SetGroupId(cg),
        consumer.SetHandler(message.PersistentConfig(handleStatusEventDeleted(l, ctx, db))),
    )
}
```
