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

### Builder

Constructs invite Model instances with validation. Defaults age to current time.

Required fields: tenant, id, inviteType, originatorId, targetId.

## Invariants

- An invite is uniquely identified by tenant, target, invite type, and reference.
- Creating an invite with a duplicate reference for the same target and type returns the existing invite.
- Invites expire after 180 seconds.
- Invite IDs start at 1000000000.

## Processors

### Processor

Provides invite lifecycle operations.

| Method | Description |
|--------|-------------|
| GetByCharacterId | Retrieves all invites targeting a character |
| ByCharacterIdProvider | Returns a provider for character invites |
| Create | Creates an invite and buffers a created event |
| CreateAndEmit | Creates an invite and emits the created event |
| Accept | Accepts an invite by reference and buffers an accepted event |
| AcceptAndEmit | Accepts an invite and emits the accepted event |
| Reject | Rejects an invite by originator and buffers a rejected event |
| RejectAndEmit | Rejects an invite and emits the rejected event |
| DeleteByCharacterIdAndEmit | Removes all invites for a character and emits rejection events |

### Registry

Redis-backed storage for invites, organized by tenant with indexes on target, originator, and target-type composite key.

| Method | Description |
|--------|-------------|
| Create | Creates and stores a new invite with deduplication by reference |
| GetByOriginator | Retrieves an invite by target, type, and originator |
| GetByReference | Retrieves an invite by target, type, and reference |
| GetForCharacter | Retrieves all invites targeting a character |
| Delete | Removes an invite by target, type, and originator |
| DeleteForCharacter | Removes all invites targeting or originated by a character |
| GetExpired | Returns all invites older than the specified duration |
| GetActiveTenants | Returns all tenants that have created invites |

### Timeout

Background task that periodically scans all active tenants for expired invites, deletes them, and emits rejection events.

| Configuration | Value |
|---------------|-------|
| Interval | 5 seconds |
| Timeout | 180 seconds |
