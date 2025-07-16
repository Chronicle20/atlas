package marriage

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// MarriageStatus represents the current state of a marriage
type MarriageStatus uint8

const (
	StatusProposed MarriageStatus = iota
	StatusEngaged
	StatusMarried
	StatusDivorced
	StatusExpired
)

// String returns the string representation of MarriageStatus
func (s MarriageStatus) String() string {
	switch s {
	case StatusProposed:
		return "proposed"
	case StatusEngaged:
		return "engaged"
	case StatusMarried:
		return "married"
	case StatusDivorced:
		return "divorced"
	case StatusExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// Marriage represents an immutable marriage domain object
type Marriage struct {
	id           uint32
	characterId1 uint32
	characterId2 uint32
	status       MarriageStatus
	proposedAt   time.Time
	engagedAt    *time.Time
	marriedAt    *time.Time
	divorcedAt   *time.Time
	tenantId     uuid.UUID
	createdAt    time.Time
	updatedAt    time.Time
}

// Id returns the marriage ID
func (m Marriage) Id() uint32 {
	return m.id
}

// CharacterId1 returns the first character ID
func (m Marriage) CharacterId1() uint32 {
	return m.characterId1
}

// CharacterId2 returns the second character ID
func (m Marriage) CharacterId2() uint32 {
	return m.characterId2
}

// Status returns the marriage status
func (m Marriage) Status() MarriageStatus {
	return m.status
}

// ProposedAt returns the proposal timestamp
func (m Marriage) ProposedAt() time.Time {
	return m.proposedAt
}

// EngagedAt returns the engagement timestamp
func (m Marriage) EngagedAt() *time.Time {
	return m.engagedAt
}

// MarriedAt returns the marriage timestamp
func (m Marriage) MarriedAt() *time.Time {
	return m.marriedAt
}

// DivorcedAt returns the divorce timestamp
func (m Marriage) DivorcedAt() *time.Time {
	return m.divorcedAt
}

// TenantId returns the tenant ID
func (m Marriage) TenantId() uuid.UUID {
	return m.tenantId
}

// CreatedAt returns the creation timestamp
func (m Marriage) CreatedAt() time.Time {
	return m.createdAt
}

// UpdatedAt returns the last update timestamp
func (m Marriage) UpdatedAt() time.Time {
	return m.updatedAt
}

// IsProposer returns true if the given character ID is the proposer
func (m Marriage) IsProposer(characterId uint32) bool {
	return m.characterId1 == characterId
}

// IsPartner returns true if the given character ID is one of the partners
func (m Marriage) IsPartner(characterId uint32) bool {
	return m.characterId1 == characterId || m.characterId2 == characterId
}

// GetPartner returns the partner's character ID given one partner's ID
func (m Marriage) GetPartner(characterId uint32) (uint32, bool) {
	if m.characterId1 == characterId {
		return m.characterId2, true
	}
	if m.characterId2 == characterId {
		return m.characterId1, true
	}
	return 0, false
}

// IsActive returns true if the marriage is currently active (married status)
func (m Marriage) IsActive() bool {
	return m.status == StatusMarried
}

// IsExpired returns true if the proposal has expired
func (m Marriage) IsExpired() bool {
	return m.status == StatusExpired
}

// IsDivorced returns true if the marriage has been divorced
func (m Marriage) IsDivorced() bool {
	return m.status == StatusDivorced
}

// CanAccept returns true if the marriage proposal can be accepted
func (m Marriage) CanAccept() bool {
	return m.status == StatusProposed
}

// CanMarry returns true if the marriage can proceed to married status
func (m Marriage) CanMarry() bool {
	return m.status == StatusEngaged
}

// CanDivorce returns true if the marriage can be divorced
func (m Marriage) CanDivorce() bool {
	return m.status == StatusMarried
}

// Builder returns a new builder for modifying the marriage
func (m Marriage) Builder() *Builder {
	return &Builder{
		id:           m.id,
		characterId1: m.characterId1,
		characterId2: m.characterId2,
		status:       m.status,
		proposedAt:   m.proposedAt,
		engagedAt:    m.engagedAt,
		marriedAt:    m.marriedAt,
		divorcedAt:   m.divorcedAt,
		tenantId:     m.tenantId,
		createdAt:    m.createdAt,
		updatedAt:    m.updatedAt,
	}
}

// Accept creates a new marriage with engaged status
func (m Marriage) Accept() (Marriage, error) {
	now := time.Now()
	return m.Builder().
		SetStatus(StatusEngaged).
		SetEngagedAt(&now).
		SetUpdatedAt(now).
		Build()
}

// Marry creates a new marriage with married status
func (m Marriage) Marry() (Marriage, error) {
	now := time.Now()
	return m.Builder().
		SetStatus(StatusMarried).
		SetMarriedAt(&now).
		SetUpdatedAt(now).
		Build()
}

// Divorce creates a new marriage with divorced status
func (m Marriage) Divorce() (Marriage, error) {
	now := time.Now()
	return m.Builder().
		SetStatus(StatusDivorced).
		SetDivorcedAt(&now).
		SetUpdatedAt(now).
		Build()
}

// Expire creates a new marriage with expired status
func (m Marriage) Expire() (Marriage, error) {
	now := time.Now()
	return m.Builder().
		SetStatus(StatusExpired).
		SetUpdatedAt(now).
		Build()
}

// ProposalStatus represents the current state of a proposal
type ProposalStatus uint8

const (
	ProposalStatusPending ProposalStatus = iota
	ProposalStatusAccepted
	ProposalStatusRejected
	ProposalStatusExpired
	ProposalStatusCancelled
)

// String returns the string representation of ProposalStatus
func (s ProposalStatus) String() string {
	switch s {
	case ProposalStatusPending:
		return "pending"
	case ProposalStatusAccepted:
		return "accepted"
	case ProposalStatusRejected:
		return "rejected"
	case ProposalStatusExpired:
		return "expired"
	case ProposalStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Proposal represents an immutable proposal domain object
type Proposal struct {
	id               uint32
	proposerId       uint32
	targetId         uint32
	status           ProposalStatus
	proposedAt       time.Time
	respondedAt      *time.Time
	expiresAt        time.Time
	rejectionCount   uint32
	cooldownUntil    *time.Time
	tenantId         uuid.UUID
	createdAt        time.Time
	updatedAt        time.Time
}

// Constants for proposal rules
const (
	ProposalExpiryDuration    = 24 * time.Hour  // 24 hours
	GlobalCooldownDuration    = 4 * time.Hour   // 4 hours between any proposals
	InitialPerTargetCooldown  = 24 * time.Hour  // 24 hours initial cooldown
)

// Id returns the proposal ID
func (p Proposal) Id() uint32 {
	return p.id
}

// ProposerId returns the proposer character ID
func (p Proposal) ProposerId() uint32 {
	return p.proposerId
}

// TargetId returns the target character ID
func (p Proposal) TargetId() uint32 {
	return p.targetId
}

// Status returns the proposal status
func (p Proposal) Status() ProposalStatus {
	return p.status
}

// ProposedAt returns the proposal timestamp
func (p Proposal) ProposedAt() time.Time {
	return p.proposedAt
}

// RespondedAt returns the response timestamp
func (p Proposal) RespondedAt() *time.Time {
	return p.respondedAt
}

// ExpiresAt returns the expiry timestamp
func (p Proposal) ExpiresAt() time.Time {
	return p.expiresAt
}

// RejectionCount returns the number of times this target has rejected proposals from this proposer
func (p Proposal) RejectionCount() uint32 {
	return p.rejectionCount
}

// CooldownUntil returns the timestamp until when the proposer cannot propose to this target again
func (p Proposal) CooldownUntil() *time.Time {
	return p.cooldownUntil
}

// TenantId returns the tenant ID
func (p Proposal) TenantId() uuid.UUID {
	return p.tenantId
}

// CreatedAt returns the creation timestamp
func (p Proposal) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns the last update timestamp
func (p Proposal) UpdatedAt() time.Time {
	return p.updatedAt
}

// IsExpired returns true if the proposal has expired
func (p Proposal) IsExpired() bool {
	return time.Now().After(p.expiresAt) || p.status == ProposalStatusExpired
}

// IsPending returns true if the proposal is still pending
func (p Proposal) IsPending() bool {
	return p.status == ProposalStatusPending && !p.IsExpired()
}

// IsAccepted returns true if the proposal has been accepted
func (p Proposal) IsAccepted() bool {
	return p.status == ProposalStatusAccepted
}

// IsRejected returns true if the proposal has been rejected
func (p Proposal) IsRejected() bool {
	return p.status == ProposalStatusRejected
}

// IsCancelled returns true if the proposal has been cancelled
func (p Proposal) IsCancelled() bool {
	return p.status == ProposalStatusCancelled
}

// CanRespond returns true if the proposal can still be responded to
func (p Proposal) CanRespond() bool {
	return p.status == ProposalStatusPending && !p.IsExpired()
}

// CanCancel returns true if the proposal can be cancelled by the proposer
func (p Proposal) CanCancel() bool {
	return p.status == ProposalStatusPending && !p.IsExpired()
}

// CalculateNextCooldown calculates the next cooldown duration based on rejection count
func (p Proposal) CalculateNextCooldown() time.Duration {
	if p.rejectionCount == 0 {
		return InitialPerTargetCooldown
	}
	
	// Exponential backoff: 24h, 48h, 96h, 192h, etc.
	multiplier := uint32(1)
	for i := uint32(0); i < p.rejectionCount; i++ {
		multiplier *= 2
	}
	
	return time.Duration(multiplier) * InitialPerTargetCooldown
}

// Accept creates a new proposal with accepted status
func (p Proposal) Accept() (Proposal, error) {
	if !p.CanRespond() {
		return Proposal{}, errors.New("proposal cannot be accepted")
	}
	
	now := time.Now()
	return p.Builder().
		SetStatus(ProposalStatusAccepted).
		SetRespondedAt(&now).
		SetUpdatedAt(now).
		Build()
}

// Reject creates a new proposal with rejected status and updates cooldown
func (p Proposal) Reject() (Proposal, error) {
	if !p.CanRespond() {
		return Proposal{}, errors.New("proposal cannot be rejected")
	}
	
	now := time.Now()
	nextCooldown := p.CalculateNextCooldown()
	cooldownUntil := now.Add(nextCooldown)
	
	return p.Builder().
		SetStatus(ProposalStatusRejected).
		SetRespondedAt(&now).
		SetRejectionCount(p.rejectionCount + 1).
		SetCooldownUntil(&cooldownUntil).
		SetUpdatedAt(now).
		Build()
}

// Cancel creates a new proposal with cancelled status
func (p Proposal) Cancel() (Proposal, error) {
	if !p.CanCancel() {
		return Proposal{}, errors.New("proposal cannot be cancelled")
	}
	
	now := time.Now()
	return p.Builder().
		SetStatus(ProposalStatusCancelled).
		SetUpdatedAt(now).
		Build()
}

// Expire creates a new proposal with expired status
func (p Proposal) Expire() (Proposal, error) {
	if p.status != ProposalStatusPending {
		return Proposal{}, errors.New("only pending proposals can expire")
	}
	
	now := time.Now()
	return p.Builder().
		SetStatus(ProposalStatusExpired).
		SetUpdatedAt(now).
		Build()
}

// Builder returns a new builder for modifying the proposal
func (p Proposal) Builder() *ProposalBuilder {
	return &ProposalBuilder{
		id:             p.id,
		proposerId:     p.proposerId,
		targetId:       p.targetId,
		status:         p.status,
		proposedAt:     p.proposedAt,
		respondedAt:    p.respondedAt,
		expiresAt:      p.expiresAt,
		rejectionCount: p.rejectionCount,
		cooldownUntil:  p.cooldownUntil,
		tenantId:       p.tenantId,
		createdAt:      p.createdAt,
		updatedAt:      p.updatedAt,
	}
}

// CeremonyStatus represents the current state of a ceremony
type CeremonyStatus uint8

const (
	CeremonyStatusScheduled CeremonyStatus = iota
	CeremonyStatusActive
	CeremonyStatusCompleted
	CeremonyStatusCancelled
	CeremonyStatusPostponed
)

// String returns the string representation of CeremonyStatus
func (s CeremonyStatus) String() string {
	switch s {
	case CeremonyStatusScheduled:
		return "scheduled"
	case CeremonyStatusActive:
		return "active"
	case CeremonyStatusCompleted:
		return "completed"
	case CeremonyStatusCancelled:
		return "cancelled"
	case CeremonyStatusPostponed:
		return "postponed"
	default:
		return "unknown"
	}
}

// Ceremony represents an immutable ceremony domain object
type Ceremony struct {
	id           uint32
	marriageId   uint32
	characterId1 uint32
	characterId2 uint32
	status       CeremonyStatus
	scheduledAt  time.Time
	startedAt    *time.Time
	completedAt  *time.Time
	cancelledAt  *time.Time
	postponedAt  *time.Time
	invitees     []uint32
	tenantId     uuid.UUID
	createdAt    time.Time
	updatedAt    time.Time
}

// Constants for ceremony rules
const (
	MaxInvitees              = 15             // Maximum number of invitees
	DisconnectionTimeout     = 5 * time.Minute // Timeout for disconnection before postponement
	CeremonyDurationUnlimited = true          // Ceremony duration is unlimited by default
)

// Id returns the ceremony ID
func (c Ceremony) Id() uint32 {
	return c.id
}

// MarriageId returns the associated marriage ID
func (c Ceremony) MarriageId() uint32 {
	return c.marriageId
}

// CharacterId1 returns the first character ID
func (c Ceremony) CharacterId1() uint32 {
	return c.characterId1
}

// CharacterId2 returns the second character ID
func (c Ceremony) CharacterId2() uint32 {
	return c.characterId2
}

// Status returns the ceremony status
func (c Ceremony) Status() CeremonyStatus {
	return c.status
}

// ScheduledAt returns the scheduled timestamp
func (c Ceremony) ScheduledAt() time.Time {
	return c.scheduledAt
}

// StartedAt returns the started timestamp
func (c Ceremony) StartedAt() *time.Time {
	return c.startedAt
}

// CompletedAt returns the completed timestamp
func (c Ceremony) CompletedAt() *time.Time {
	return c.completedAt
}

// CancelledAt returns the cancelled timestamp
func (c Ceremony) CancelledAt() *time.Time {
	return c.cancelledAt
}

// PostponedAt returns the postponed timestamp
func (c Ceremony) PostponedAt() *time.Time {
	return c.postponedAt
}

// Invitees returns the list of invitee character IDs
func (c Ceremony) Invitees() []uint32 {
	// Return a copy to maintain immutability
	invitees := make([]uint32, len(c.invitees))
	copy(invitees, c.invitees)
	return invitees
}

// TenantId returns the tenant ID
func (c Ceremony) TenantId() uuid.UUID {
	return c.tenantId
}

// CreatedAt returns the creation timestamp
func (c Ceremony) CreatedAt() time.Time {
	return c.createdAt
}

// UpdatedAt returns the last update timestamp
func (c Ceremony) UpdatedAt() time.Time {
	return c.updatedAt
}

// IsScheduled returns true if the ceremony is scheduled
func (c Ceremony) IsScheduled() bool {
	return c.status == CeremonyStatusScheduled
}

// IsActive returns true if the ceremony is currently active
func (c Ceremony) IsActive() bool {
	return c.status == CeremonyStatusActive
}

// IsCompleted returns true if the ceremony has been completed
func (c Ceremony) IsCompleted() bool {
	return c.status == CeremonyStatusCompleted
}

// IsCancelled returns true if the ceremony has been cancelled
func (c Ceremony) IsCancelled() bool {
	return c.status == CeremonyStatusCancelled
}

// IsPostponed returns true if the ceremony has been postponed
func (c Ceremony) IsPostponed() bool {
	return c.status == CeremonyStatusPostponed
}

// IsFinished returns true if the ceremony has reached a terminal state
func (c Ceremony) IsFinished() bool {
	return c.status == CeremonyStatusCompleted || c.status == CeremonyStatusCancelled
}

// CanStart returns true if the ceremony can be started
func (c Ceremony) CanStart() bool {
	return c.status == CeremonyStatusScheduled || c.status == CeremonyStatusPostponed
}

// CanComplete returns true if the ceremony can be completed
func (c Ceremony) CanComplete() bool {
	return c.status == CeremonyStatusActive
}

// CanCancel returns true if the ceremony can be cancelled
func (c Ceremony) CanCancel() bool {
	return c.status == CeremonyStatusScheduled || c.status == CeremonyStatusActive || c.status == CeremonyStatusPostponed
}

// CanPostpone returns true if the ceremony can be postponed
func (c Ceremony) CanPostpone() bool {
	return c.status == CeremonyStatusActive
}

// CanReschedule returns true if the ceremony can be rescheduled
func (c Ceremony) CanReschedule() bool {
	return c.status == CeremonyStatusScheduled || c.status == CeremonyStatusPostponed
}

// IsPartner returns true if the given character ID is one of the partners
func (c Ceremony) IsPartner(characterId uint32) bool {
	return c.characterId1 == characterId || c.characterId2 == characterId
}

// IsInvited returns true if the given character ID is invited to the ceremony
func (c Ceremony) IsInvited(characterId uint32) bool {
	for _, invitee := range c.invitees {
		if invitee == characterId {
			return true
		}
	}
	return false
}

// InviteeCount returns the number of invitees
func (c Ceremony) InviteeCount() int {
	return len(c.invitees)
}

// CanAddInvitee returns true if a new invitee can be added
func (c Ceremony) CanAddInvitee(characterId uint32) bool {
	if c.InviteeCount() >= MaxInvitees {
		return false
	}
	if c.IsPartner(characterId) {
		return false
	}
	if c.IsInvited(characterId) {
		return false
	}
	return c.status == CeremonyStatusScheduled || c.status == CeremonyStatusPostponed
}

// CanRemoveInvitee returns true if an invitee can be removed
func (c Ceremony) CanRemoveInvitee(characterId uint32) bool {
	if !c.IsInvited(characterId) {
		return false
	}
	return c.status == CeremonyStatusScheduled || c.status == CeremonyStatusPostponed
}

// Start creates a new ceremony with active status
func (c Ceremony) Start() (Ceremony, error) {
	if !c.CanStart() {
		return Ceremony{}, errors.New("ceremony cannot be started")
	}
	
	now := time.Now()
	return c.Builder().
		SetStatus(CeremonyStatusActive).
		SetStartedAt(&now).
		SetUpdatedAt(now).
		Build()
}

// Complete creates a new ceremony with completed status
func (c Ceremony) Complete() (Ceremony, error) {
	if !c.CanComplete() {
		return Ceremony{}, errors.New("ceremony cannot be completed")
	}
	
	now := time.Now()
	return c.Builder().
		SetStatus(CeremonyStatusCompleted).
		SetCompletedAt(&now).
		SetUpdatedAt(now).
		Build()
}

// Cancel creates a new ceremony with cancelled status
func (c Ceremony) Cancel() (Ceremony, error) {
	if !c.CanCancel() {
		return Ceremony{}, errors.New("ceremony cannot be cancelled")
	}
	
	now := time.Now()
	return c.Builder().
		SetStatus(CeremonyStatusCancelled).
		SetCancelledAt(&now).
		SetUpdatedAt(now).
		Build()
}

// Postpone creates a new ceremony with postponed status
func (c Ceremony) Postpone() (Ceremony, error) {
	if !c.CanPostpone() {
		return Ceremony{}, errors.New("ceremony cannot be postponed")
	}
	
	now := time.Now()
	return c.Builder().
		SetStatus(CeremonyStatusPostponed).
		SetPostponedAt(&now).
		SetUpdatedAt(now).
		Build()
}

// Reschedule creates a new ceremony with updated schedule
func (c Ceremony) Reschedule(newScheduledAt time.Time) (Ceremony, error) {
	if !c.CanReschedule() {
		return Ceremony{}, errors.New("ceremony cannot be rescheduled")
	}
	
	now := time.Now()
	return c.Builder().
		SetStatus(CeremonyStatusScheduled).
		SetScheduledAt(newScheduledAt).
		SetStartedAt(nil).
		SetPostponedAt(nil).
		SetUpdatedAt(now).
		Build()
}

// AddInvitee creates a new ceremony with an additional invitee
func (c Ceremony) AddInvitee(characterId uint32) (Ceremony, error) {
	if !c.CanAddInvitee(characterId) {
		return Ceremony{}, errors.New("invitee cannot be added")
	}
	
	newInvitees := make([]uint32, len(c.invitees)+1)
	copy(newInvitees, c.invitees)
	newInvitees[len(c.invitees)] = characterId
	
	now := time.Now()
	return c.Builder().
		SetInvitees(newInvitees).
		SetUpdatedAt(now).
		Build()
}

// RemoveInvitee creates a new ceremony with an invitee removed
func (c Ceremony) RemoveInvitee(characterId uint32) (Ceremony, error) {
	if !c.CanRemoveInvitee(characterId) {
		return Ceremony{}, errors.New("invitee cannot be removed")
	}
	
	newInvitees := make([]uint32, 0, len(c.invitees)-1)
	for _, invitee := range c.invitees {
		if invitee != characterId {
			newInvitees = append(newInvitees, invitee)
		}
	}
	
	now := time.Now()
	return c.Builder().
		SetInvitees(newInvitees).
		SetUpdatedAt(now).
		Build()
}

// Builder returns a new builder for modifying the ceremony
func (c Ceremony) Builder() *CeremonyBuilder {
	// Copy invitees to maintain immutability
	invitees := make([]uint32, len(c.invitees))
	copy(invitees, c.invitees)
	
	return &CeremonyBuilder{
		id:           c.id,
		marriageId:   c.marriageId,
		characterId1: c.characterId1,
		characterId2: c.characterId2,
		status:       c.status,
		scheduledAt:  c.scheduledAt,
		startedAt:    c.startedAt,
		completedAt:  c.completedAt,
		cancelledAt:  c.cancelledAt,
		postponedAt:  c.postponedAt,
		invitees:     invitees,
		tenantId:     c.tenantId,
		createdAt:    c.createdAt,
		updatedAt:    c.updatedAt,
	}
}

// ToEntity converts the ceremony domain model to an entity for persistence
func (c Ceremony) ToEntity() CeremonyEntity {
	// Call the existing ToCeremonyEntity method which handles proper serialization
	entity, _ := c.ToCeremonyEntity()
	return entity
}