package marriage

import (
	"time"
)

// Topic environment variable names
const (
	// Command topics
	EnvCommandTopic = "COMMAND_TOPIC_MARRIAGE"

	// Event topics
	EnvEventTopicStatus = "EVENT_TOPIC_MARRIAGE_STATUS"
)

// Command Types
const (
	// Proposal commands
	CommandMarriagePropose = "PROPOSE"
	CommandMarriageAccept  = "ACCEPT"
	CommandMarriageDecline = "DECLINE"
	CommandMarriageCancel  = "CANCEL"

	// Marriage commands
	CommandMarriageDivorce = "DIVORCE"

	// Ceremony commands
	CommandCeremonySchedule         = "SCHEDULE_CEREMONY"
	CommandCeremonyStart            = "START_CEREMONY"
	CommandCeremonyComplete         = "COMPLETE_CEREMONY"
	CommandCeremonyCancel           = "CANCEL_CEREMONY"
	CommandCeremonyPostpone         = "POSTPONE_CEREMONY"
	CommandCeremonyReschedule       = "RESCHEDULE_CEREMONY"
	CommandCeremonyAddInvitee       = "ADD_INVITEE"
	CommandCeremonyRemoveInvitee    = "REMOVE_INVITEE"
	CommandCeremonyAdvanceState     = "ADVANCE_CEREMONY_STATE"
)

// Event Types
const (
	// Proposal events
	EventProposalCreated = "PROPOSAL_CREATED"
	EventProposalAccepted = "PROPOSAL_ACCEPTED"
	EventProposalDeclined = "PROPOSAL_DECLINED"
	EventProposalExpired  = "PROPOSAL_EXPIRED"
	EventProposalCancelled = "PROPOSAL_CANCELLED"

	// Marriage events
	EventMarriageCreated = "MARRIAGE_CREATED"
	EventMarriageDivorced = "MARRIAGE_DIVORCED"
	EventMarriageDeleted = "MARRIAGE_DELETED"

	// Ceremony events
	EventCeremonyScheduled = "CEREMONY_SCHEDULED"
	EventCeremonyStarted   = "CEREMONY_STARTED"
	EventCeremonyCompleted = "CEREMONY_COMPLETED"
	EventCeremonyPostponed = "CEREMONY_POSTPONED"
	EventCeremonyCancelled = "CEREMONY_CANCELLED"
	EventCeremonyRescheduled = "CEREMONY_RESCHEDULED"
	EventInviteeAdded      = "INVITEE_ADDED"
	EventInviteeRemoved    = "INVITEE_REMOVED"

	// Error events
	EventMarriageError = "MARRIAGE_ERROR"
)

// Generic command structure
type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// Generic event structure
type Event[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// Command Bodies

// ProposeBody represents the body of a marriage proposal command
type ProposeBody struct {
	TargetCharacterId uint32 `json:"targetCharacterId"`
}

// AcceptBody represents the body of a proposal acceptance command
type AcceptBody struct {
	ProposalId uint32 `json:"proposalId"`
}

// DeclineBody represents the body of a proposal decline command
type DeclineBody struct {
	ProposalId uint32 `json:"proposalId"`
}

// CancelBody represents the body of a proposal cancellation command
type CancelBody struct {
	ProposalId uint32 `json:"proposalId"`
}

// DivorceBody represents the body of a divorce command
type DivorceBody struct {
	MarriageId uint32 `json:"marriageId"`
}


// ScheduleCeremonyBody represents the body of a ceremony scheduling command
type ScheduleCeremonyBody struct {
	MarriageId  uint32    `json:"marriageId"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Invitees    []uint32  `json:"invitees"`
}

// StartCeremonyBody represents the body of a ceremony start command
type StartCeremonyBody struct {
	CeremonyId uint32 `json:"ceremonyId"`
}

// CompleteCeremonyBody represents the body of a ceremony completion command
type CompleteCeremonyBody struct {
	CeremonyId uint32 `json:"ceremonyId"`
}

// CancelCeremonyBody represents the body of a ceremony cancellation command
type CancelCeremonyBody struct {
	CeremonyId uint32 `json:"ceremonyId"`
}

// PostponeCeremonyBody represents the body of a ceremony postponement command
type PostponeCeremonyBody struct {
	CeremonyId uint32 `json:"ceremonyId"`
}

// RescheduleCeremonyBody represents the body of a ceremony rescheduling command
type RescheduleCeremonyBody struct {
	CeremonyId  uint32    `json:"ceremonyId"`
	ScheduledAt time.Time `json:"scheduledAt"`
}

// AddInviteeBody represents the body of an add invitee command
type AddInviteeBody struct {
	CeremonyId  uint32 `json:"ceremonyId"`
	CharacterId uint32 `json:"characterId"`
}

// RemoveInviteeBody represents the body of a remove invitee command
type RemoveInviteeBody struct {
	CeremonyId  uint32 `json:"ceremonyId"`
	CharacterId uint32 `json:"characterId"`
}

// AdvanceCeremonyStateBody represents the body of a ceremony state advancement command
type AdvanceCeremonyStateBody struct {
	CeremonyId uint32 `json:"ceremonyId"`
	NextState  string `json:"nextState"`
}

// Event Bodies

// ProposalCreatedBody represents the body of a proposal created event
type ProposalCreatedBody struct {
	ProposalId        uint32    `json:"proposalId"`
	ProposerId        uint32    `json:"proposerId"`
	TargetCharacterId uint32    `json:"targetCharacterId"`
	ProposedAt        time.Time `json:"proposedAt"`
	ExpiresAt         time.Time `json:"expiresAt"`
}

// ProposalAcceptedBody represents the body of a proposal accepted event
type ProposalAcceptedBody struct {
	ProposalId        uint32    `json:"proposalId"`
	ProposerId        uint32    `json:"proposerId"`
	TargetCharacterId uint32    `json:"targetCharacterId"`
	AcceptedAt        time.Time `json:"acceptedAt"`
}

// ProposalDeclinedBody represents the body of a proposal declined event
type ProposalDeclinedBody struct {
	ProposalId        uint32    `json:"proposalId"`
	ProposerId        uint32    `json:"proposerId"`
	TargetCharacterId uint32    `json:"targetCharacterId"`
	DeclinedAt        time.Time `json:"declinedAt"`
	RejectionCount    uint32    `json:"rejectionCount"`
	CooldownUntil     time.Time `json:"cooldownUntil"`
}

// ProposalExpiredBody represents the body of a proposal expired event
type ProposalExpiredBody struct {
	ProposalId        uint32    `json:"proposalId"`
	ProposerId        uint32    `json:"proposerId"`
	TargetCharacterId uint32    `json:"targetCharacterId"`
	ExpiredAt         time.Time `json:"expiredAt"`
}

// ProposalCancelledBody represents the body of a proposal cancelled event
type ProposalCancelledBody struct {
	ProposalId        uint32    `json:"proposalId"`
	ProposerId        uint32    `json:"proposerId"`
	TargetCharacterId uint32    `json:"targetCharacterId"`
	CancelledAt       time.Time `json:"cancelledAt"`
}

// MarriageCreatedBody represents the body of a marriage created event
type MarriageCreatedBody struct {
	MarriageId   uint32    `json:"marriageId"`
	CharacterId1 uint32    `json:"characterId1"`
	CharacterId2 uint32    `json:"characterId2"`
	MarriedAt    time.Time `json:"marriedAt"`
}

// MarriageDivorcedBody represents the body of a marriage divorced event
type MarriageDivorcedBody struct {
	MarriageId     uint32    `json:"marriageId"`
	CharacterId1   uint32    `json:"characterId1"`
	CharacterId2   uint32    `json:"characterId2"`
	DivorcedAt     time.Time `json:"divorcedAt"`
	InitiatedBy    uint32    `json:"initiatedBy"`
}

// MarriageDeletedBody represents the body of a marriage deleted event
type MarriageDeletedBody struct {
	MarriageId     uint32    `json:"marriageId"`
	CharacterId1   uint32    `json:"characterId1"`
	CharacterId2   uint32    `json:"characterId2"`
	DeletedAt      time.Time `json:"deletedAt"`
	DeletedBy      uint32    `json:"deletedBy"`
	Reason         string    `json:"reason"`
}

// CeremonyScheduledBody represents the body of a ceremony scheduled event
type CeremonyScheduledBody struct {
	CeremonyId   uint32    `json:"ceremonyId"`
	MarriageId   uint32    `json:"marriageId"`
	CharacterId1 uint32    `json:"characterId1"`
	CharacterId2 uint32    `json:"characterId2"`
	ScheduledAt  time.Time `json:"scheduledAt"`
	Invitees     []uint32  `json:"invitees"`
}

// CeremonyStartedBody represents the body of a ceremony started event
type CeremonyStartedBody struct {
	CeremonyId   uint32    `json:"ceremonyId"`
	MarriageId   uint32    `json:"marriageId"`
	CharacterId1 uint32    `json:"characterId1"`
	CharacterId2 uint32    `json:"characterId2"`
	StartedAt    time.Time `json:"startedAt"`
}

// CeremonyCompletedBody represents the body of a ceremony completed event
type CeremonyCompletedBody struct {
	CeremonyId   uint32    `json:"ceremonyId"`
	MarriageId   uint32    `json:"marriageId"`
	CharacterId1 uint32    `json:"characterId1"`
	CharacterId2 uint32    `json:"characterId2"`
	CompletedAt  time.Time `json:"completedAt"`
}

// CeremonyPostponedBody represents the body of a ceremony postponed event
type CeremonyPostponedBody struct {
	CeremonyId   uint32    `json:"ceremonyId"`
	MarriageId   uint32    `json:"marriageId"`
	CharacterId1 uint32    `json:"characterId1"`
	CharacterId2 uint32    `json:"characterId2"`
	PostponedAt  time.Time `json:"postponedAt"`
	Reason       string    `json:"reason"`
}

// CeremonyCancelledBody represents the body of a ceremony cancelled event
type CeremonyCancelledBody struct {
	CeremonyId   uint32    `json:"ceremonyId"`
	MarriageId   uint32    `json:"marriageId"`
	CharacterId1 uint32    `json:"characterId1"`
	CharacterId2 uint32    `json:"characterId2"`
	CancelledAt  time.Time `json:"cancelledAt"`
	CancelledBy  uint32    `json:"cancelledBy"`
	Reason       string    `json:"reason"`
}

// CeremonyRescheduledBody represents the body of a ceremony rescheduled event
type CeremonyRescheduledBody struct {
	CeremonyId      uint32    `json:"ceremonyId"`
	MarriageId      uint32    `json:"marriageId"`
	CharacterId1    uint32    `json:"characterId1"`
	CharacterId2    uint32    `json:"characterId2"`
	RescheduledAt   time.Time `json:"rescheduledAt"`
	NewScheduledAt  time.Time `json:"newScheduledAt"`
	RescheduledBy   uint32    `json:"rescheduledBy"`
}

// InviteeAddedBody represents the body of an invitee added event
type InviteeAddedBody struct {
	CeremonyId   uint32    `json:"ceremonyId"`
	MarriageId   uint32    `json:"marriageId"`
	CharacterId1 uint32    `json:"characterId1"`
	CharacterId2 uint32    `json:"characterId2"`
	InviteeId    uint32    `json:"inviteeId"`
	AddedAt      time.Time `json:"addedAt"`
	AddedBy      uint32    `json:"addedBy"`
}

// InviteeRemovedBody represents the body of an invitee removed event
type InviteeRemovedBody struct {
	CeremonyId   uint32    `json:"ceremonyId"`
	MarriageId   uint32    `json:"marriageId"`
	CharacterId1 uint32    `json:"characterId1"`
	CharacterId2 uint32    `json:"characterId2"`
	InviteeId    uint32    `json:"inviteeId"`
	RemovedAt    time.Time `json:"removedAt"`
	RemovedBy    uint32    `json:"removedBy"`
}

// MarriageErrorBody represents the body of a marriage error event
type MarriageErrorBody struct {
	ErrorType   string    `json:"errorType"`
	ErrorCode   string    `json:"errorCode"`
	Message     string    `json:"message"`
	CharacterId uint32    `json:"characterId"`
	Context     string    `json:"context"`
	Timestamp   time.Time `json:"timestamp"`
}

// Error types for MarriageErrorBody
const (
	ErrorTypeProposal            = "PROPOSAL_ERROR"
	ErrorTypeMarriage            = "MARRIAGE_ERROR"
	ErrorTypeCeremony            = "CEREMONY_ERROR"
	ErrorTypeValidation          = "VALIDATION_ERROR"
	ErrorTypeCooldown            = "COOLDOWN_ERROR"
	ErrorTypeEligibility         = "ELIGIBILITY_ERROR"
	ErrorTypeNotFound            = "NOT_FOUND_ERROR"
	ErrorTypeAlreadyExists       = "ALREADY_EXISTS_ERROR"
	ErrorTypeStateTransition     = "STATE_TRANSITION_ERROR"
	ErrorTypeInviteeLimit        = "INVITEE_LIMIT_ERROR"
	ErrorTypeDisconnectionTimeout = "DISCONNECTION_TIMEOUT_ERROR"
)

// Error codes for specific error scenarios
const (
	ErrorCodeAlreadyMarried           = "ALREADY_MARRIED"
	ErrorCodeAlreadyEngaged           = "ALREADY_ENGAGED"
	ErrorCodeInsufficientLevel        = "INSUFFICIENT_LEVEL"
	ErrorCodeSelfProposal             = "SELF_PROPOSAL"
	ErrorCodeGlobalCooldown           = "GLOBAL_COOLDOWN"
	ErrorCodeTargetCooldown           = "TARGET_COOLDOWN"
	ErrorCodeProposalExpired          = "PROPOSAL_EXPIRED"
	ErrorCodeProposalNotFound         = "PROPOSAL_NOT_FOUND"
	ErrorCodeMarriageNotFound         = "MARRIAGE_NOT_FOUND"
	ErrorCodeCeremonyNotFound         = "CEREMONY_NOT_FOUND"
	ErrorCodeInvalidState             = "INVALID_STATE"
	ErrorCodeInviteeLimitExceeded     = "INVITEE_LIMIT_EXCEEDED"
	ErrorCodeInviteeAlreadyInvited    = "INVITEE_ALREADY_INVITED"
	ErrorCodeInviteeNotFound          = "INVITEE_NOT_FOUND"
	ErrorCodePartnerDisconnected      = "PARTNER_DISCONNECTED"
	ErrorCodeCeremonyTimeout          = "CEREMONY_TIMEOUT"
	ErrorCodeConcurrentProposal       = "CONCURRENT_PROPOSAL"
	ErrorCodeTenantMismatch           = "TENANT_MISMATCH"
)