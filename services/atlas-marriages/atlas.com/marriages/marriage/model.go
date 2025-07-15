package marriage

import (
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