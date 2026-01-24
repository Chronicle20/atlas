# Marriage Domain

## Responsibility

The Marriage domain manages character relationships including proposals, engagements, marriages, ceremonies, and divorces. It enforces eligibility requirements, cooldown periods, and state transitions across the marriage lifecycle.

## Core Models

### Marriage

Immutable domain object representing a relationship between two characters.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Marriage identifier |
| characterId1 | uint32 | First character (proposer) |
| characterId2 | uint32 | Second character (target) |
| status | MarriageStatus | Current marriage state |
| proposedAt | time.Time | Proposal timestamp |
| engagedAt | *time.Time | Engagement timestamp |
| marriedAt | *time.Time | Marriage timestamp |
| divorcedAt | *time.Time | Divorce timestamp |
| tenantId | uuid.UUID | Tenant identifier |
| createdAt | time.Time | Creation timestamp |
| updatedAt | time.Time | Last update timestamp |

### Proposal

Immutable domain object representing a marriage proposal.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Proposal identifier |
| proposerId | uint32 | Proposing character |
| targetId | uint32 | Target character |
| status | ProposalStatus | Current proposal state |
| proposedAt | time.Time | Proposal timestamp |
| respondedAt | *time.Time | Response timestamp |
| expiresAt | time.Time | Expiration timestamp |
| rejectionCount | uint32 | Number of rejections |
| cooldownUntil | *time.Time | Cooldown end timestamp |
| tenantId | uuid.UUID | Tenant identifier |
| createdAt | time.Time | Creation timestamp |
| updatedAt | time.Time | Last update timestamp |

### Ceremony

Immutable domain object representing a wedding ceremony.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Ceremony identifier |
| marriageId | uint32 | Associated marriage |
| characterId1 | uint32 | First partner |
| characterId2 | uint32 | Second partner |
| status | CeremonyStatus | Current ceremony state |
| scheduledAt | time.Time | Scheduled timestamp |
| startedAt | *time.Time | Start timestamp |
| completedAt | *time.Time | Completion timestamp |
| cancelledAt | *time.Time | Cancellation timestamp |
| postponedAt | *time.Time | Postponement timestamp |
| invitees | []uint32 | Invited character IDs |
| tenantId | uuid.UUID | Tenant identifier |
| createdAt | time.Time | Creation timestamp |
| updatedAt | time.Time | Last update timestamp |

## Invariants

### Eligibility

- Characters must be level 10 or higher to marry
- Characters cannot marry themselves
- Characters can only be in one relationship at a time
- Both characters must be in the same tenant

### Proposal Constraints

- Global cooldown: 4 hours between any proposals by the same character
- Per-target cooldown: Starts at 24 hours, doubles on each successive rejection
- Proposals expire after 24 hours without response
- Cannot propose if currently married or engaged
- Cannot receive proposals if already engaged

### Ceremony Constraints

- Maximum of 15 invitees per ceremony
- Partners cannot be invitees
- No duplicate invitees allowed
- Ceremony is postponed if partner is offline for 5+ minutes during active ceremony

## State Transitions

### MarriageStatus

| From | To | Trigger |
|------|-----|---------|
| proposed | engaged | Proposal accepted |
| proposed | expired | Proposal expired |
| engaged | married | Ceremony completed |
| married | divorced | Divorce initiated |

### ProposalStatus

| From | To | Trigger |
|------|-----|---------|
| pending | accepted | Target accepts |
| pending | rejected | Target declines |
| pending | expired | Time elapsed |
| pending | cancelled | Proposer cancels |

### CeremonyStatus

| From | To | Trigger |
|------|-----|---------|
| scheduled | active | Ceremony starts |
| scheduled | cancelled | Ceremony cancelled |
| active | completed | Ceremony completes |
| active | cancelled | Ceremony cancelled |
| active | postponed | Partner disconnected |
| postponed | scheduled | Ceremony rescheduled |
| postponed | active | Ceremony restarted |
| postponed | cancelled | Ceremony cancelled |

## Processors

### Processor

Handles marriage, proposal, and ceremony operations.

**Proposal Operations**
- `Propose` - Creates a new marriage proposal with eligibility checks
- `ProposeAndEmit` - Creates proposal and emits events
- `AcceptProposal` - Accepts a proposal and creates a marriage
- `AcceptProposalAndEmit` - Accepts proposal and emits events
- `DeclineProposal` - Declines a proposal and updates cooldown
- `DeclineProposalAndEmit` - Declines proposal and emits events
- `CancelProposal` - Cancels a pending proposal
- `CancelProposalAndEmit` - Cancels proposal and emits events
- `ExpireProposal` - Marks a proposal as expired
- `ExpireProposalAndEmit` - Expires proposal and emits events
- `ProcessExpiredProposals` - Batch processes all expired proposals

**Marriage Operations**
- `Divorce` - Initiates divorce proceedings
- `DivorceAndEmit` - Divorces and emits events
- `HandleCharacterDeletion` - Auto-divorces when character is deleted
- `HandleCharacterDeletionAndEmit` - Handles deletion with events
- `GetMarriageByCharacter` - Retrieves active marriage for character
- `GetMarriageHistory` - Retrieves marriage history for character

**Ceremony Operations**
- `ScheduleCeremony` - Schedules a ceremony for an engaged marriage
- `ScheduleCeremonyAndEmit` - Schedules ceremony and emits events
- `StartCeremony` - Transitions ceremony to active state
- `StartCeremonyAndEmit` - Starts ceremony and emits events
- `CompleteCeremony` - Transitions ceremony to completed state
- `CompleteCeremonyAndEmit` - Completes ceremony and emits events
- `CancelCeremony` - Cancels a ceremony
- `CancelCeremonyAndEmit` - Cancels ceremony and emits events
- `PostponeCeremony` - Postpones an active ceremony
- `PostponeCeremonyAndEmit` - Postpones ceremony and emits events
- `RescheduleCeremony` - Reschedules a ceremony to a new time
- `RescheduleCeremonyAndEmit` - Reschedules ceremony and emits events
- `AddInvitee` - Adds an invitee to a ceremony
- `AddInviteeAndEmit` - Adds invitee and emits events
- `RemoveInvitee` - Removes an invitee from a ceremony
- `RemoveInviteeAndEmit` - Removes invitee and emits events
- `AdvanceCeremonyState` - Advances ceremony to the next state
- `AdvanceCeremonyStateAndEmit` - Advances state and emits events
- `ProcessCeremonyTimeouts` - Batch processes ceremony timeouts

**Eligibility Operations**
- `CheckEligibility` - Checks if character meets level requirement
- `CheckProposalEligibility` - Comprehensive eligibility check for proposals
- `CheckGlobalCooldown` - Checks if proposer is in global cooldown
- `CheckPerTargetCooldown` - Checks per-target cooldown status

**Query Operations**
- `GetActiveProposal` - Retrieves active proposal between two characters
- `GetPendingProposalsByCharacter` - Retrieves pending proposals for character
- `GetProposalHistory` - Retrieves proposal history between characters
- `GetCeremonyById` - Retrieves ceremony by ID
- `GetCeremonyByMarriage` - Retrieves ceremony by marriage ID
- `GetUpcomingCeremonies` - Retrieves scheduled ceremonies
- `GetActiveCeremonies` - Retrieves active ceremonies
