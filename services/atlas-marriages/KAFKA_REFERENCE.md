# Kafka Event/Command Reference - Marriage Service

This document provides a comprehensive reference for all Kafka messages (commands and events) used by the Marriage Service. The service follows the Atlas microservice messaging patterns with type-safe generic messages and proper error handling.

## Table of Contents

1. [Overview](#overview)
2. [Message Structure](#message-structure)
3. [Topics Configuration](#topics-configuration)
4. [Commands](#commands)
5. [Events](#events)
6. [Error Handling](#error-handling)
7. [Examples](#examples)
8. [Integration Guide](#integration-guide)

## Overview

The Marriage Service uses Kafka messaging for:
- **Commands**: External services sending requests to the Marriage Service
- **Events**: Marriage Service notifying other services of state changes
- **Error Events**: Structured error reporting for failed operations

All messages are partitioned by character ID and include tenant context via headers.

## Message Structure

All marriage messages follow a generic structure with typed bodies:

### Command Structure
```go
type Command[E any] struct {
    CharacterId uint32 `json:"characterId"`
    Type        string `json:"type"`
    Body        E      `json:"body"`
}
```

### Event Structure
```go
type Event[E any] struct {
    CharacterId uint32 `json:"characterId"`
    Type        string `json:"type"`
    Body        E      `json:"body"`
}
```

## Topics Configuration

| Topic | Environment Variable | Purpose |
|-------|---------------------|---------|
| Command Topic | `COMMAND_TOPIC_MARRIAGE` | Receives commands from external services |
| Event Topic | `EVENT_TOPIC_MARRIAGE_STATUS` | Emits events to external services |
| Character Events | `EVENT_TOPIC_CHARACTER_STATUS` | Consumes character lifecycle events |

## Commands

Commands are sent **TO** the Marriage Service to trigger operations.

### Proposal Commands

#### PROPOSE
**Type**: `PROPOSE`  
**Purpose**: Initiate a marriage proposal between two characters.

**Body Structure**:
```go
type ProposeBody struct {
    TargetCharacterId uint32 `json:"targetCharacterId"`
}
```

**Validation**:
- Proposer must be level 10+
- Target must be level 10+
- Both characters must be in same tenant
- Proposer cannot be married or engaged
- Target cannot be engaged
- Must respect cooldown periods

---

#### ACCEPT
**Type**: `ACCEPT`  
**Purpose**: Accept a pending marriage proposal.

**Body Structure**:
```go
type AcceptBody struct {
    ProposalId uint32 `json:"proposalId"`
}
```

**Validation**:
- Proposal must exist and be active
- Character must be the target of the proposal
- Proposal must not be expired

---

#### DECLINE
**Type**: `DECLINE`  
**Purpose**: Decline a pending marriage proposal.

**Body Structure**:
```go
type DeclineBody struct {
    ProposalId uint32 `json:"proposalId"`
}
```

**Effects**:
- Increments rejection count
- Applies exponential cooldown
- Emits `PROPOSAL_DECLINED` event

---

#### CANCEL
**Type**: `CANCEL`  
**Purpose**: Cancel a pending proposal (by proposer).

**Body Structure**:
```go
type CancelBody struct {
    ProposalId uint32 `json:"proposalId"`
}
```

**Validation**:
- Character must be the proposer
- Proposal must be active

### Marriage Commands

#### DIVORCE
**Type**: `DIVORCE`  
**Purpose**: Initiate unilateral divorce.

**Body Structure**:
```go
type DivorceBody struct {
    MarriageId uint32 `json:"marriageId"`
}
```

**Validation**:
- Character must be married
- Marriage must be active

### Ceremony Commands

#### SCHEDULE_CEREMONY
**Type**: `SCHEDULE_CEREMONY`  
**Purpose**: Schedule a wedding ceremony for an engaged couple.

**Body Structure**:
```go
type ScheduleCeremonyBody struct {
    MarriageId  uint32    `json:"marriageId"`
    ScheduledAt time.Time `json:"scheduledAt"`
    Invitees    []uint32  `json:"invitees"`
}
```

**Validation**:
- Marriage must be in engaged state
- Maximum 15 invitees
- Scheduled time must be in future

---

#### START_CEREMONY
**Type**: `START_CEREMONY`  
**Purpose**: Begin a scheduled ceremony.

**Body Structure**:
```go
type StartCeremonyBody struct {
    CeremonyId uint32 `json:"ceremonyId"`
}
```

---

#### COMPLETE_CEREMONY
**Type**: `COMPLETE_CEREMONY`  
**Purpose**: Mark ceremony as completed, finalizing the marriage.

**Body Structure**:
```go
type CompleteCeremonyBody struct {
    CeremonyId uint32 `json:"ceremonyId"`
}
```

---

#### CANCEL_CEREMONY
**Type**: `CANCEL_CEREMONY`  
**Purpose**: Cancel a scheduled ceremony.

**Body Structure**:
```go
type CancelCeremonyBody struct {
    CeremonyId uint32 `json:"ceremonyId"`
}
```

---

#### POSTPONE_CEREMONY
**Type**: `POSTPONE_CEREMONY`  
**Purpose**: Temporarily postpone an active ceremony.

**Body Structure**:
```go
type PostponeCeremonyBody struct {
    CeremonyId uint32 `json:"ceremonyId"`
}
```

---

#### RESCHEDULE_CEREMONY
**Type**: `RESCHEDULE_CEREMONY`  
**Purpose**: Change the scheduled time of a ceremony.

**Body Structure**:
```go
type RescheduleCeremonyBody struct {
    CeremonyId  uint32    `json:"ceremonyId"`
    ScheduledAt time.Time `json:"scheduledAt"`
}
```

---

#### ADD_INVITEE
**Type**: `ADD_INVITEE`  
**Purpose**: Add an invitee to a ceremony.

**Body Structure**:
```go
type AddInviteeBody struct {
    CeremonyId  uint32 `json:"ceremonyId"`
    CharacterId uint32 `json:"characterId"`
}
```

**Validation**:
- Maximum 15 invitees total
- Character not already invited

---

#### REMOVE_INVITEE
**Type**: `REMOVE_INVITEE`  
**Purpose**: Remove an invitee from a ceremony.

**Body Structure**:
```go
type RemoveInviteeBody struct {
    CeremonyId  uint32 `json:"ceremonyId"`
    CharacterId uint32 `json:"characterId"`
}
```

---

#### ADVANCE_CEREMONY_STATE
**Type**: `ADVANCE_CEREMONY_STATE`  
**Purpose**: Advance ceremony through its state machine.

**Body Structure**:
```go
type AdvanceCeremonyStateBody struct {
    CeremonyId uint32 `json:"ceremonyId"`
    NextState  string `json:"nextState"`
}
```

**Valid States**: `SCHEDULED`, `ACTIVE`, `COMPLETED`, `CANCELLED`, `POSTPONED`

## Events

Events are emitted **BY** the Marriage Service to notify external services.

### Proposal Events

#### PROPOSAL_CREATED
**Type**: `PROPOSAL_CREATED`  
**Emitted**: When a new proposal is created.

**Body Structure**:
```go
type ProposalCreatedBody struct {
    ProposalId        uint32    `json:"proposalId"`
    ProposerId        uint32    `json:"proposerId"`
    TargetCharacterId uint32    `json:"targetCharacterId"`
    ProposedAt        time.Time `json:"proposedAt"`
    ExpiresAt         time.Time `json:"expiresAt"`
}
```

---

#### PROPOSAL_ACCEPTED
**Type**: `PROPOSAL_ACCEPTED`  
**Emitted**: When a proposal is accepted, creating an engagement.

**Body Structure**:
```go
type ProposalAcceptedBody struct {
    ProposalId        uint32    `json:"proposalId"`
    ProposerId        uint32    `json:"proposerId"`
    TargetCharacterId uint32    `json:"targetCharacterId"`
    AcceptedAt        time.Time `json:"acceptedAt"`
}
```

---

#### PROPOSAL_DECLINED
**Type**: `PROPOSAL_DECLINED`  
**Emitted**: When a proposal is declined.

**Body Structure**:
```go
type ProposalDeclinedBody struct {
    ProposalId        uint32    `json:"proposalId"`
    ProposerId        uint32    `json:"proposerId"`
    TargetCharacterId uint32    `json:"targetCharacterId"`
    DeclinedAt        time.Time `json:"declinedAt"`
    RejectionCount    uint32    `json:"rejectionCount"`
    CooldownUntil     time.Time `json:"cooldownUntil"`
}
```

---

#### PROPOSAL_EXPIRED
**Type**: `PROPOSAL_EXPIRED`  
**Emitted**: When a proposal expires after 24 hours.

**Body Structure**:
```go
type ProposalExpiredBody struct {
    ProposalId        uint32    `json:"proposalId"`
    ProposerId        uint32    `json:"proposerId"`
    TargetCharacterId uint32    `json:"targetCharacterId"`
    ExpiredAt         time.Time `json:"expiredAt"`
}
```

---

#### PROPOSAL_CANCELLED
**Type**: `PROPOSAL_CANCELLED`  
**Emitted**: When a proposal is cancelled by the proposer.

**Body Structure**:
```go
type ProposalCancelledBody struct {
    ProposalId        uint32    `json:"proposalId"`
    ProposerId        uint32    `json:"proposerId"`
    TargetCharacterId uint32    `json:"targetCharacterId"`
    CancelledAt       time.Time `json:"cancelledAt"`
}
```

### Marriage Events

#### MARRIAGE_CREATED
**Type**: `MARRIAGE_CREATED`  
**Emitted**: When a ceremony is completed and marriage is finalized.

**Body Structure**:
```go
type MarriageCreatedBody struct {
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    MarriedAt    time.Time `json:"marriedAt"`
}
```

---

#### MARRIAGE_DIVORCED
**Type**: `MARRIAGE_DIVORCED`  
**Emitted**: When a marriage is dissolved through divorce.

**Body Structure**:
```go
type MarriageDivorcedBody struct {
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    DivorcedAt   time.Time `json:"divorcedAt"`
    InitiatedBy  uint32    `json:"initiatedBy"`
}
```

---

#### MARRIAGE_DELETED
**Type**: `MARRIAGE_DELETED`  
**Emitted**: When a marriage is deleted due to character deletion.

**Body Structure**:
```go
type MarriageDeletedBody struct {
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    DeletedAt    time.Time `json:"deletedAt"`
    DeletedBy    uint32    `json:"deletedBy"`
    Reason       string    `json:"reason"`
}
```

### Ceremony Events

#### CEREMONY_SCHEDULED
**Type**: `CEREMONY_SCHEDULED`  
**Emitted**: When a ceremony is scheduled.

**Body Structure**:
```go
type CeremonyScheduledBody struct {
    CeremonyId   uint32    `json:"ceremonyId"`
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    ScheduledAt  time.Time `json:"scheduledAt"`
    Invitees     []uint32  `json:"invitees"`
}
```

---

#### CEREMONY_STARTED
**Type**: `CEREMONY_STARTED`  
**Emitted**: When a ceremony begins.

**Body Structure**:
```go
type CeremonyStartedBody struct {
    CeremonyId   uint32    `json:"ceremonyId"`
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    StartedAt    time.Time `json:"startedAt"`
}
```

---

#### CEREMONY_COMPLETED
**Type**: `CEREMONY_COMPLETED`  
**Emitted**: When a ceremony is completed successfully.

**Body Structure**:
```go
type CeremonyCompletedBody struct {
    CeremonyId   uint32    `json:"ceremonyId"`
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    CompletedAt  time.Time `json:"completedAt"`
}
```

---

#### CEREMONY_POSTPONED
**Type**: `CEREMONY_POSTPONED`  
**Emitted**: When a ceremony is postponed (e.g., due to disconnection).

**Body Structure**:
```go
type CeremonyPostponedBody struct {
    CeremonyId   uint32    `json:"ceremonyId"`
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    PostponedAt  time.Time `json:"postponedAt"`
    Reason       string    `json:"reason"`
}
```

---

#### CEREMONY_CANCELLED
**Type**: `CEREMONY_CANCELLED`  
**Emitted**: When a ceremony is cancelled.

**Body Structure**:
```go
type CeremonyCancelledBody struct {
    CeremonyId   uint32    `json:"ceremonyId"`
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    CancelledAt  time.Time `json:"cancelledAt"`
    CancelledBy  uint32    `json:"cancelledBy"`
    Reason       string    `json:"reason"`
}
```

---

#### CEREMONY_RESCHEDULED
**Type**: `CEREMONY_RESCHEDULED`  
**Emitted**: When a ceremony is rescheduled.

**Body Structure**:
```go
type CeremonyRescheduledBody struct {
    CeremonyId      uint32    `json:"ceremonyId"`
    MarriageId      uint32    `json:"marriageId"`
    CharacterId1    uint32    `json:"characterId1"`
    CharacterId2    uint32    `json:"characterId2"`
    RescheduledAt   time.Time `json:"rescheduledAt"`
    NewScheduledAt  time.Time `json:"newScheduledAt"`
    RescheduledBy   uint32    `json:"rescheduledBy"`
}
```

---

#### INVITEE_ADDED
**Type**: `INVITEE_ADDED`  
**Emitted**: When an invitee is added to a ceremony.

**Body Structure**:
```go
type InviteeAddedBody struct {
    CeremonyId   uint32    `json:"ceremonyId"`
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    InviteeId    uint32    `json:"inviteeId"`
    AddedAt      time.Time `json:"addedAt"`
    AddedBy      uint32    `json:"addedBy"`
}
```

---

#### INVITEE_REMOVED
**Type**: `INVITEE_REMOVED`  
**Emitted**: When an invitee is removed from a ceremony.

**Body Structure**:
```go
type InviteeRemovedBody struct {
    CeremonyId   uint32    `json:"ceremonyId"`
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    InviteeId    uint32    `json:"inviteeId"`
    RemovedAt    time.Time `json:"removedAt"`
    RemovedBy    uint32    `json:"removedBy"`
}
```

## Error Handling

### Error Event Structure
**Type**: `MARRIAGE_ERROR`  
**Emitted**: When an operation fails.

**Body Structure**:
```go
type MarriageErrorBody struct {
    ErrorType   string                 `json:"errorType"`
    ErrorCode   string                 `json:"errorCode"`
    Message     string                 `json:"message"`
    CharacterId uint32                 `json:"characterId"`
    Context     map[string]interface{} `json:"context"`
    Timestamp   time.Time              `json:"timestamp"`
}
```

### Error Types

| Error Type | Description |
|------------|-------------|
| `PROPOSAL_ERROR` | Errors related to proposal operations |
| `MARRIAGE_ERROR` | Errors related to marriage operations |
| `CEREMONY_ERROR` | Errors related to ceremony operations |
| `VALIDATION_ERROR` | Input validation failures |
| `COOLDOWN_ERROR` | Cooldown period violations |
| `ELIGIBILITY_ERROR` | Character eligibility issues |
| `NOT_FOUND_ERROR` | Resource not found |
| `ALREADY_EXISTS_ERROR` | Duplicate resource creation |
| `STATE_TRANSITION_ERROR` | Invalid state transitions |
| `INVITEE_LIMIT_ERROR` | Invitee limit violations |
| `DISCONNECTION_TIMEOUT_ERROR` | Ceremony timeout due to disconnection |

### Error Codes

| Error Code | Description |
|------------|-------------|
| `ALREADY_MARRIED` | Character is already married |
| `ALREADY_ENGAGED` | Character is already engaged |
| `INSUFFICIENT_LEVEL` | Character level below minimum (10) |
| `SELF_PROPOSAL` | Cannot propose to self |
| `GLOBAL_COOLDOWN` | Global proposal cooldown active |
| `TARGET_COOLDOWN` | Target-specific cooldown active |
| `PROPOSAL_EXPIRED` | Proposal has expired |
| `PROPOSAL_NOT_FOUND` | Proposal does not exist |
| `MARRIAGE_NOT_FOUND` | Marriage does not exist |
| `CEREMONY_NOT_FOUND` | Ceremony does not exist |
| `INVALID_STATE` | Invalid state for operation |
| `INVITEE_LIMIT_EXCEEDED` | More than 15 invitees |
| `INVITEE_ALREADY_INVITED` | Character already invited |
| `INVITEE_NOT_FOUND` | Invitee not found in ceremony |
| `PARTNER_DISCONNECTED` | Partner disconnected during ceremony |
| `CEREMONY_TIMEOUT` | Ceremony timed out |
| `CONCURRENT_PROPOSAL` | Concurrent proposal attempt |
| `TENANT_MISMATCH` | Characters in different tenants |

## Examples

### Sending a Proposal Command
```json
{
  "characterId": 12345,
  "type": "PROPOSE",
  "body": {
    "targetCharacterId": 67890
  }
}
```

### Proposal Created Event
```json
{
  "characterId": 12345,
  "type": "PROPOSAL_CREATED",
  "body": {
    "proposalId": 1001,
    "proposerId": 12345,
    "targetCharacterId": 67890,
    "proposedAt": "2023-10-15T14:30:00Z",
    "expiresAt": "2023-10-16T14:30:00Z"
  }
}
```

### Error Event Example
```json
{
  "characterId": 12345,
  "type": "MARRIAGE_ERROR",
  "body": {
    "errorType": "ELIGIBILITY_ERROR",
    "errorCode": "ALREADY_MARRIED",
    "message": "Character is already married",
    "characterId": 12345,
    "context": {
      "marriageId": 500,
      "partnerId": 54321
    },
    "timestamp": "2023-10-15T14:30:00Z"
  }
}
```

## Integration Guide

### For External Services

1. **Consuming Marriage Events**: Subscribe to `EVENT_TOPIC_MARRIAGE_STATUS` to receive notifications of marriage state changes.

2. **Sending Commands**: Publish to `COMMAND_TOPIC_MARRIAGE` to initiate marriage operations.

3. **Error Handling**: Always handle `MARRIAGE_ERROR` events to provide user feedback.

4. **Headers**: Include proper tenant and span headers for tracing and multi-tenancy.

### Message Partitioning

All messages are partitioned by character ID to ensure:
- Ordered processing for each character
- Scalable load distribution
- Consistent state management

### Tenant Context

Tenant information is included in message headers, never in the message body:
- `TENANT_ID`: UUID format tenant identifier
- `REGION`: Game region identifier
- `MAJOR_VERSION`: Major version number
- `MINOR_VERSION`: Minor version number

### Reliability

- All events are emitted transactionally
- Failed messages are retried with exponential backoff
- Dead letter queues handle unprocessable messages
- Idempotent operations prevent duplicate processing

### Monitoring

Key metrics to monitor:
- Message throughput per topic
- Error event frequency
- Processing latency
- Consumer lag
- Cooldown violations

## Related Documentation

- [REST API Reference](API_REFERENCE.md)
- [Service Architecture](README.md)
- [Deployment Guide](DEPLOYMENT.md)
- [Atlas Kafka Patterns](https://docs.atlas.com/kafka)