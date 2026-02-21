# Marriage Kafka Integration

## Topics Consumed

### COMMAND_TOPIC_MARRIAGE

Marriage command topic for receiving marriage, proposal, and ceremony commands.

**Environment Variable:** `COMMAND_TOPIC_MARRIAGE`
**Consumer Group:** Marriage Service

| Command | Body Type | Description |
|---------|-----------|-------------|
| PROPOSE | ProposeBody | Initiate a marriage proposal |
| ACCEPT | AcceptBody | Accept a marriage proposal |
| DECLINE | DeclineBody | Decline a marriage proposal |
| CANCEL | CancelBody | Cancel a pending proposal |
| DIVORCE | DivorceBody | Initiate divorce proceedings |
| SCHEDULE_CEREMONY | ScheduleCeremonyBody | Schedule a wedding ceremony |
| START_CEREMONY | StartCeremonyBody | Start a scheduled ceremony |
| COMPLETE_CEREMONY | CompleteCeremonyBody | Complete an active ceremony |
| CANCEL_CEREMONY | CancelCeremonyBody | Cancel a ceremony |
| POSTPONE_CEREMONY | PostponeCeremonyBody | Postpone an active ceremony |
| RESCHEDULE_CEREMONY | RescheduleCeremonyBody | Reschedule a ceremony |
| ADD_INVITEE | AddInviteeBody | Add invitee to ceremony |
| REMOVE_INVITEE | RemoveInviteeBody | Remove invitee from ceremony |
| ADVANCE_CEREMONY_STATE | AdvanceCeremonyStateBody | Advance ceremony state |

### EVENT_TOPIC_CHARACTER_STATUS

Character status event topic for receiving character deletion events.

**Environment Variable:** `EVENT_TOPIC_CHARACTER_STATUS`
**Consumer Group:** Marriage Service

| Event | Body Type | Description |
|-------|-----------|-------------|
| DELETED | DeletedStatusEventBody | Character has been deleted |

## Topics Produced

### EVENT_TOPIC_MARRIAGE_STATUS

Marriage status event topic for publishing marriage, proposal, and ceremony events.

**Environment Variable:** `EVENT_TOPIC_MARRIAGE_STATUS`

| Event | Body Type | Description |
|-------|-----------|-------------|
| PROPOSAL_CREATED | ProposalCreatedBody | Proposal has been created |
| PROPOSAL_ACCEPTED | ProposalAcceptedBody | Proposal has been accepted |
| PROPOSAL_DECLINED | ProposalDeclinedBody | Proposal has been declined |
| PROPOSAL_EXPIRED | ProposalExpiredBody | Proposal has expired |
| PROPOSAL_CANCELLED | ProposalCancelledBody | Proposal has been cancelled |
| MARRIAGE_CREATED | MarriageCreatedBody | Marriage has been created |
| MARRIAGE_DIVORCED | MarriageDivorcedBody | Marriage has been divorced |
| MARRIAGE_DELETED | MarriageDeletedBody | Marriage deleted due to character deletion |
| CEREMONY_SCHEDULED | CeremonyScheduledBody | Ceremony has been scheduled |
| CEREMONY_STARTED | CeremonyStartedBody | Ceremony has started |
| CEREMONY_COMPLETED | CeremonyCompletedBody | Ceremony has completed |
| CEREMONY_POSTPONED | CeremonyPostponedBody | Ceremony has been postponed |
| CEREMONY_CANCELLED | CeremonyCancelledBody | Ceremony has been cancelled |
| CEREMONY_RESCHEDULED | CeremonyRescheduledBody | Ceremony has been rescheduled |
| INVITEE_ADDED | InviteeAddedBody | Invitee added to ceremony |
| INVITEE_REMOVED | InviteeRemovedBody | Invitee removed from ceremony |
| MARRIAGE_ERROR | MarriageErrorBody | Error occurred during operation |

## Message Types

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

### Command Bodies

**ProposeBody**
```go
type ProposeBody struct {
    TargetCharacterId uint32 `json:"targetCharacterId"`
}
```

**AcceptBody**
```go
type AcceptBody struct {
    ProposalId uint32 `json:"proposalId"`
}
```

**DeclineBody**
```go
type DeclineBody struct {
    ProposalId uint32 `json:"proposalId"`
}
```

**CancelBody**
```go
type CancelBody struct {
    ProposalId uint32 `json:"proposalId"`
}
```

**DivorceBody**
```go
type DivorceBody struct {
    MarriageId uint32 `json:"marriageId"`
}
```

**ScheduleCeremonyBody**
```go
type ScheduleCeremonyBody struct {
    MarriageId  uint32    `json:"marriageId"`
    ScheduledAt time.Time `json:"scheduledAt"`
    Invitees    []uint32  `json:"invitees"`
}
```

**StartCeremonyBody**
```go
type StartCeremonyBody struct {
    CeremonyId uint32 `json:"ceremonyId"`
}
```

**CompleteCeremonyBody**
```go
type CompleteCeremonyBody struct {
    CeremonyId uint32 `json:"ceremonyId"`
}
```

**CancelCeremonyBody**
```go
type CancelCeremonyBody struct {
    CeremonyId uint32 `json:"ceremonyId"`
}
```

**PostponeCeremonyBody**
```go
type PostponeCeremonyBody struct {
    CeremonyId uint32 `json:"ceremonyId"`
}
```

**RescheduleCeremonyBody**
```go
type RescheduleCeremonyBody struct {
    CeremonyId  uint32    `json:"ceremonyId"`
    ScheduledAt time.Time `json:"scheduledAt"`
}
```

**AddInviteeBody**
```go
type AddInviteeBody struct {
    CeremonyId  uint32 `json:"ceremonyId"`
    CharacterId uint32 `json:"characterId"`
}
```

**RemoveInviteeBody**
```go
type RemoveInviteeBody struct {
    CeremonyId  uint32 `json:"ceremonyId"`
    CharacterId uint32 `json:"characterId"`
}
```

**AdvanceCeremonyStateBody**
```go
type AdvanceCeremonyStateBody struct {
    CeremonyId uint32 `json:"ceremonyId"`
    NextState  string `json:"nextState"`
}
```

### Event Bodies

**ProposalCreatedBody**
```go
type ProposalCreatedBody struct {
    ProposalId        uint32    `json:"proposalId"`
    ProposerId        uint32    `json:"proposerId"`
    TargetCharacterId uint32    `json:"targetCharacterId"`
    ProposedAt        time.Time `json:"proposedAt"`
    ExpiresAt         time.Time `json:"expiresAt"`
}
```

**ProposalAcceptedBody**
```go
type ProposalAcceptedBody struct {
    ProposalId        uint32    `json:"proposalId"`
    ProposerId        uint32    `json:"proposerId"`
    TargetCharacterId uint32    `json:"targetCharacterId"`
    AcceptedAt        time.Time `json:"acceptedAt"`
}
```

**ProposalDeclinedBody**
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

**ProposalExpiredBody**
```go
type ProposalExpiredBody struct {
    ProposalId        uint32    `json:"proposalId"`
    ProposerId        uint32    `json:"proposerId"`
    TargetCharacterId uint32    `json:"targetCharacterId"`
    ExpiredAt         time.Time `json:"expiredAt"`
}
```

**ProposalCancelledBody**
```go
type ProposalCancelledBody struct {
    ProposalId        uint32    `json:"proposalId"`
    ProposerId        uint32    `json:"proposerId"`
    TargetCharacterId uint32    `json:"targetCharacterId"`
    CancelledAt       time.Time `json:"cancelledAt"`
}
```

**MarriageCreatedBody**
```go
type MarriageCreatedBody struct {
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    MarriedAt    time.Time `json:"marriedAt"`
}
```

**MarriageDivorcedBody**
```go
type MarriageDivorcedBody struct {
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    DivorcedAt   time.Time `json:"divorcedAt"`
    InitiatedBy  uint32    `json:"initiatedBy"`
}
```

**MarriageDeletedBody**
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

**CeremonyScheduledBody**
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

**CeremonyStartedBody**
```go
type CeremonyStartedBody struct {
    CeremonyId   uint32    `json:"ceremonyId"`
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    StartedAt    time.Time `json:"startedAt"`
}
```

**CeremonyCompletedBody**
```go
type CeremonyCompletedBody struct {
    CeremonyId   uint32    `json:"ceremonyId"`
    MarriageId   uint32    `json:"marriageId"`
    CharacterId1 uint32    `json:"characterId1"`
    CharacterId2 uint32    `json:"characterId2"`
    CompletedAt  time.Time `json:"completedAt"`
}
```

**CeremonyPostponedBody**
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

**CeremonyCancelledBody**
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

**CeremonyRescheduledBody**
```go
type CeremonyRescheduledBody struct {
    CeremonyId     uint32    `json:"ceremonyId"`
    MarriageId     uint32    `json:"marriageId"`
    CharacterId1   uint32    `json:"characterId1"`
    CharacterId2   uint32    `json:"characterId2"`
    RescheduledAt  time.Time `json:"rescheduledAt"`
    NewScheduledAt time.Time `json:"newScheduledAt"`
    RescheduledBy  uint32    `json:"rescheduledBy"`
}
```

**InviteeAddedBody**
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

**InviteeRemovedBody**
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

**MarriageErrorBody**
```go
type MarriageErrorBody struct {
    ErrorType   string    `json:"errorType"`
    ErrorCode   string    `json:"errorCode"`
    Message     string    `json:"message"`
    CharacterId uint32    `json:"characterId"`
    Context     string    `json:"context"`
    Timestamp   time.Time `json:"timestamp"`
}
```

## Transaction Semantics

- Commands are processed with persistent configuration
- Events are emitted using message buffering for transactional consistency
- Multiple related events are emitted together in a single buffer
- Messages are keyed by character ID for partition ordering
- Header parsers extract span context and tenant context
