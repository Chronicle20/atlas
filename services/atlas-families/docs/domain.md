# Family Domain

## Responsibility

The Family domain manages hierarchical character relationships (senior-junior) and reputation tracking for family members.

## Core Models

### FamilyMember

An immutable domain model representing a family member with the following attributes:

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Internal database identifier |
| characterId | uint32 | Game character identifier |
| tenantId | uuid.UUID | Multi-tenant identifier |
| seniorId | *uint32 | Reference to senior's character ID (nil for root members) |
| juniorIds | []uint32 | List of junior character IDs (max 2) |
| rep | uint32 | Total accumulated reputation |
| dailyRep | uint32 | Daily reputation gained |
| level | uint16 | Character level |
| world | byte | Game world identifier |
| createdAt | time.Time | Creation timestamp |
| updatedAt | time.Time | Last modification timestamp |

### Builder

A fluent builder for constructing FamilyMember instances with validation.

### BatchResetResult

Represents the result of a batch daily reputation reset operation.

| Field | Type | Description |
|-------|------|-------------|
| AffectedCount | int64 | Number of members affected by reset |
| ResetTime | time.Time | Timestamp of reset execution |

## Invariants

- characterId must be non-zero
- tenantId must be non-nil UUID
- level must be greater than zero
- juniorIds cannot contain more than 2 entries
- seniorId cannot equal characterId (no self-reference)
- juniorIds cannot contain characterId (no self-reference)
- juniorIds cannot contain duplicates
- dailyRep cannot exceed 5000

## Processors

### Processor Interface

Defines core business logic operations:

| Method | Description |
|--------|-------------|
| AddJunior | Adds a junior to a senior's family |
| RemoveMember | Removes a member from the family with cascade operations |
| BreakLink | Breaks the family link for a character |
| AwardRep | Awards reputation to a character |
| DeductRep | Deducts reputation from a character |
| ResetDailyRep | Resets daily reputation for all members |
| GetFamilyTree | Retrieves the complete family tree for a character |
| GetByCharacterId | Retrieves a family member by character ID |

### AndEmit Variants

Each processor method has an `AndEmit` variant that combines business logic execution with Kafka event emission:

- AddJuniorAndEmit
- RemoveMemberAndEmit
- BreakLinkAndEmit
- AwardRepAndEmit
- DeductRepAndEmit

### Administrator Functions

| Function | Description |
|----------|-------------|
| CreateMember | Creates a new family member with validation |
| SaveMember | Saves a family member to the database |
| DeleteMember | Deletes a family member from the database |
| BatchResetDailyRep | Resets daily reputation for all members |

### Provider Functions

| Function | Description |
|----------|-------------|
| GetByCharacterIdProvider | Returns provider for finding member by character ID |
| GetByIdProvider | Returns provider for finding member by ID |
| GetBySeniorIdProvider | Returns provider for finding all juniors of a senior |
| GetFamilyTreeProvider | Returns provider for getting complete family tree |
| ExistsProvider | Returns provider for checking if member exists |

### Scheduler

| Component | Description |
|-----------|-------------|
| ReputationResetJob | Handles daily reputation reset scheduling |
