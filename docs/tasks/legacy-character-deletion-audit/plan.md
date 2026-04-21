# Character Deletion Event Handling Audit Plan

**Last Updated: 2026-02-03**

## Executive Summary

This plan addresses a critical data integrity issue where multiple Atlas services track character-related data but fail to properly clean up that data when a character is deleted. When the `atlas-character` service emits a `DELETED` status event, only 9 out of ~20 services that track character data properly handle this event. This leaves orphaned database records and potential memory leaks across the system.

## Current State Analysis

### Services That CORRECTLY Handle Character Deletion (9 services)

| Service | Cleanup Method | Notes |
|---------|---------------|-------|
| atlas-buddies | `DeleteAndEmit()` | Removes buddy entries, notifies all buddies |
| atlas-cashshop | `DeleteAllAndEmit()` | Clears wishlist items |
| atlas-inventory | `DeleteAndEmit()` | Cascades through compartments and assets |
| atlas-keys | `Delete()` | Removes keybind configurations |
| atlas-marriages | `HandleCharacterDeletionAndEmit()` | Comprehensive relationship cleanup |
| atlas-notes | `DeleteAllAndEmit()` | Removes all notes for character |
| atlas-parties | `LeaveAndEmit()` | Removes from party membership |
| atlas-pets | `DeleteForCharacterAndEmit()` | Deletes all pets owned by character |
| atlas-skills | `ClearAll()` + `Delete()` | Removes skills and macros |

### Services That NEED Character Deletion Handling

#### Critical - Persistent Database with character_id FK (4 services)

| Service | Database Entity | Impact |
|---------|-----------------|--------|
| **atlas-quest** | `quest_statuses`, `quest_progress` | Quest records orphaned |
| **atlas-guilds** | `members`, `characters` | Guild member records orphaned, other members not notified |
| **atlas-families** | `family_members` | Referential integrity broken (senior_id/junior_ids), family not notified |
| **atlas-fame** | Fame records | Historical fame data orphaned |

#### NOT Needed - In-Memory Services (2 services)

| Service | Reason |
|---------|--------|
| **atlas-rates** | Character must logout before deletion; LOGOUT event already clears registry |
| **atlas-messengers** | Character must logout before deletion; LOGOUT event already clears state |

### Services That Do NOT Need Deletion Handling

These services either:
- Store ephemeral session data that is cleared on logout (already handled)
- Store data keyed by account_id, not character_id
- Store static configuration data (NPC shops, etc.)

| Service | Reason |
|---------|--------|
| atlas-channel | Ephemeral session state, cleared on logout |
| atlas-chalkboards | In-memory, cleared on logout |
| atlas-consumables | In-memory location tracking, cleared on logout |
| atlas-maps | In-memory presence tracking, cleared on logout |
| atlas-chairs | In-memory, cleared on logout |
| atlas-portals | In-memory blocked state, cleared on logout |
| atlas-transports | In-memory route state, cleared on logout |
| atlas-npc-conversations | Conversation state, cleared on logout |
| atlas-npc-shops | NPC configuration data, not character-specific |
| atlas-storage | Keyed by account_id, not character_id |
| atlas-buffs | In-memory, non-persistent |
| atlas-saga-orchestrator | In-memory, non-persistent |

## Proposed Future State

All services that persist character-related data to a database or maintain character-keyed registries will:

1. Subscribe to character status events via Kafka
2. Handle the `DELETED` event type
3. Clean up all character-related data within a database transaction
4. Emit appropriate status events for audit trail

## Implementation Phases

### Phase 1: Critical Database Cleanup (High Priority)

Services with persistent database records and foreign key relationships.

#### 1.1 atlas-quest

**Current State:**
- Has `deleteByCharacterIdWithProgress()` function already implemented
- No character deletion event handler exists

**Implementation:**
- Add handler in `kafka/consumer/character/consumer.go`
- Call existing `deleteByCharacterIdWithProgress()` on DELETED event
- Emit quest deletion events for audit

**Effort:** Small (S)

#### 1.2 atlas-guilds

**Current State:**
- Has `LeaveAndEmit(guildId, characterId, force, transactionId)` processor method that:
  - Removes the member record via `RemoveMember()`
  - Emits `MEMBER_LEFT` status event with `force` flag
- No character deletion event handler to trigger this

**Implementation:**
- Add character consumer in `kafka/consumer/character/consumer.go`
- On DELETED event, lookup character's guild via `GetByMemberId(characterId)`
- If in a guild, call existing `LeaveAndEmit(guildId, characterId, true, transactionId)`
- Handle "not in guild" case gracefully

**Effort:** Small (S)

#### 1.3 atlas-families

**Current State:**
- Complex family tree structure with senior/junior relationships
- `RemoveMemberAndEmit()` processor method already exists and handles:
  - Removing character from senior's `junior_ids` list
  - Clearing `senior_id` for all juniors
  - Deleting member record
  - **Emitting `LinkBrokenEvent` to notify affected family members**
- No character deletion event handler to trigger this

**Implementation:**
- Add character consumer
- On DELETED event, call existing `RemoveMemberAndEmit(transactionId, characterId, "CHARACTER_DELETED")`
- Handle `ErrMemberNotFound` gracefully (character may not be in a family)

**Effort:** Small (S)

#### 1.4 atlas-fame

**Current State:**
- Stores fame history with character_id
- Need to verify current handling

**Implementation:**
- Audit current state
- Add deletion handler if missing
- Clean up fame records for deleted character

**Effort:** Small (S)

## Risk Assessment and Mitigation

### Risk 1: Incomplete Cascade Deletion
**Risk:** Deleting records without proper cascade may leave orphaned child records
**Mitigation:** Review entity relationships, use database transactions, test with integration tests

### Risk 2: Event Ordering Issues
**Risk:** Deletion events may arrive before related events are processed
**Mitigation:** Character deletion should be the final event; services should handle idempotently

### Risk 3: Service Discovery
**Risk:** New services may be added that track character data without deletion handlers
**Mitigation:** Document the pattern, add to service creation checklist, consider automated auditing

### Risk 4: Social Notification Failures
**Risk:** Guild/family members may not receive notifications if events fail to emit
**Mitigation:** Use message buffer pattern with transactional event emission; verify events in integration tests

## Success Metrics

1. **Zero Orphaned Records:** After character deletion, no database records remain with deleted character_id
2. **Social Notifications Sent:** Guild members receive `MEMBER_LEFT`, family members receive `LinkBrokenEvent`
3. **Test Coverage:** Integration tests verify deletion cascades and event emissions for each service

## Required Resources and Dependencies

### Dependencies
- Understanding of existing deletion patterns (documented above)
- Access to all service codebases
- Test environment for integration testing

### Technical Requirements
- Kafka consumer registration pattern (established)
- GORM transaction handling (established)
- Event emission patterns (established)

## Scope Summary

| Phase | Services | Notes |
|-------|----------|-------|
| Phase 1 | 4 services | atlas-quest, atlas-guilds, atlas-families, atlas-fame |
| Phase 2 | Verification & Docs | Testing and documentation |

**Note:** atlas-rates and atlas-messengers were excluded because characters must be logged out before deletion, and LOGOUT events already clean up in-memory state.

## Implementation Pattern Reference

All implementations should follow the established pattern from working services:

```go
func handleStatusEventDeleted(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) message.Handler[character.StatusEvent] {
    return func(e character.StatusEvent) {
        if e.Type != character.StatusEventTypeDeleted {
            return
        }
        // Service-specific cleanup with transaction and event emission
        err := processor.NewProcessor(l, ctx, db).DeleteAndEmit(e.CharacterId, e.WorldId)
        if err != nil {
            l.WithError(err).Errorf("Unable to process character [%d] deletion.", e.CharacterId)
        }
    }
}
```

Key elements:
1. Type check for DELETED event
2. Use processor with transaction support
3. Emit events within transaction
4. Error logging with character context
