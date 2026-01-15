# Invite Domain

## Responsibility

Manages the lifecycle of invitations between characters. An invite represents a pending request from an originator character to a target character, associated with a reference entity and categorized by type.

## Core Models

### Model

Represents an invitation.

| Field | Type | Description |
|-------|------|-------------|
| tenant | tenant.Model | Tenant context |
| id | uint32 | Unique invite identifier |
| inviteType | string | Category of invite |
| referenceId | uint32 | Reference entity identifier |
| originatorId | uint32 | Character who sent the invite |
| targetId | uint32 | Character who receives the invite |
| worldId | byte | World identifier |
| age | time.Time | Creation timestamp |

### Invite Types

- BUDDY
- FAMILY
- FAMILY_SUMMON
- MESSENGER
- TRADE
- PARTY
- GUILD
- ALLIANCE

## Invariants

- An invite is uniquely identified by tenant, target, invite type, and reference.
- Creating an invite with a duplicate reference for the same target and type returns the existing invite.
- Invites expire after 180 seconds.

## Processors

### Processor

Provides invite lifecycle operations.

| Method | Description |
|--------|-------------|
| GetByCharacterId | Retrieves all invites for a character |
| ByCharacterIdProvider | Returns a provider for character invites |
| Create | Creates an invite and buffers a created event |
| CreateAndEmit | Creates an invite and emits the created event |
| Accept | Accepts an invite by reference and buffers an accepted event |
| AcceptAndEmit | Accepts an invite and emits the accepted event |
| Reject | Rejects an invite by originator and buffers a rejected event |
| RejectAndEmit | Rejects an invite and emits the rejected event |

### Registry

In-memory storage for invites, organized by tenant, target character, and invite type.

| Method | Description |
|--------|-------------|
| Create | Creates and stores a new invite |
| GetByOriginator | Retrieves an invite by target, type, and originator |
| GetByReference | Retrieves an invite by target, type, and reference |
| GetForCharacter | Retrieves all invites for a character |
| Delete | Removes an invite |
| GetExpired | Returns all invites older than the specified duration |

### Timeout

Background task that periodically scans for expired invites, deletes them, and emits rejection events.

| Configuration | Value |
|---------------|-------|
| Interval | 5 seconds |
| Timeout | 180 seconds |
