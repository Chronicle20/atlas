package family

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// FamilyMember represents an immutable family member with private fields
type FamilyMember struct {
	id          uint32
	characterId uint32
	tenantId    uuid.UUID
	seniorId    *uint32
	juniorIds   []uint32
	rep         uint32
	dailyRep    uint32
	level       uint16
	world       byte
	createdAt   time.Time
	updatedAt   time.Time
}

// Accessor methods for FamilyMember
func (fm FamilyMember) Id() uint32 {
	return fm.id
}

func (fm FamilyMember) CharacterId() uint32 {
	return fm.characterId
}

func (fm FamilyMember) TenantId() uuid.UUID {
	return fm.tenantId
}

func (fm FamilyMember) SeniorId() *uint32 {
	return fm.seniorId
}

func (fm FamilyMember) JuniorIds() []uint32 {
	// Return a copy to maintain immutability
	result := make([]uint32, len(fm.juniorIds))
	copy(result, fm.juniorIds)
	return result
}

func (fm FamilyMember) Rep() uint32 {
	return fm.rep
}

func (fm FamilyMember) DailyRep() uint32 {
	return fm.dailyRep
}

func (fm FamilyMember) Level() uint16 {
	return fm.level
}

func (fm FamilyMember) World() byte {
	return fm.world
}


func (fm FamilyMember) CreatedAt() time.Time {
	return fm.createdAt
}

func (fm FamilyMember) UpdatedAt() time.Time {
	return fm.updatedAt
}

// Business logic methods

// HasSenior returns true if the member has a senior
func (fm FamilyMember) HasSenior() bool {
	return fm.seniorId != nil
}

// HasJuniors returns true if the member has any juniors
func (fm FamilyMember) HasJuniors() bool {
	return len(fm.juniorIds) > 0
}

// JuniorCount returns the number of juniors
func (fm FamilyMember) JuniorCount() int {
	return len(fm.juniorIds)
}

// CanAddJunior returns true if the member can add another junior
func (fm FamilyMember) CanAddJunior() bool {
	return len(fm.juniorIds) < 2
}

// HasJunior returns true if the specified character is a junior
func (fm FamilyMember) HasJunior(characterId uint32) bool {
	for _, juniorId := range fm.juniorIds {
		if juniorId == characterId {
			return true
		}
	}
	return false
}

// ValidateLevelDifference checks if the level difference is within acceptable range
func (fm FamilyMember) ValidateLevelDifference(otherLevel uint16) bool {
	diff := int(fm.level) - int(otherLevel)
	if diff < 0 {
		diff = -diff
	}
	return diff <= 20
}

// IsSameWorld returns true if the member is on the same world
func (fm FamilyMember) IsSameWorld(world byte) bool {
	return fm.world == world
}

// IsRepCapReached returns true if daily rep limit is reached
func (fm FamilyMember) IsRepCapReached() bool {
	return fm.dailyRep >= 5000
}

// CanReceiveRep returns true if the member can receive more rep today
func (fm FamilyMember) CanReceiveRep(amount uint32) bool {
	return fm.dailyRep+amount <= 5000
}

// Builder forward declaration - implementation in builder.go
type Builder struct {
	id          uint32
	characterId uint32
	tenantId    uuid.UUID
	seniorId    *uint32
	juniorIds   []uint32
	rep         uint32
	dailyRep    uint32
	level       uint16
	world       byte
	createdAt   time.Time
	updatedAt   time.Time
}

// Builder returns a new builder for modification
func (fm FamilyMember) Builder() *Builder {
	return &Builder{
		id:          fm.id,
		characterId: fm.characterId,
		tenantId:    fm.tenantId,
		seniorId:    fm.seniorId,
		juniorIds:   append([]uint32{}, fm.juniorIds...),
		rep:         fm.rep,
		dailyRep:    fm.dailyRep,
		level:       fm.level,
		world:       fm.world,
		createdAt:   fm.createdAt,
		updatedAt:   fm.updatedAt,
	}
}

// Validation errors
var (
	ErrInvalidCharacterId = errors.New("invalid character ID")
	ErrInvalidTenantId    = errors.New("invalid tenant ID")
	ErrInvalidLevel       = errors.New("invalid level")
	ErrTooManyJuniors     = errors.New("cannot have more than 2 juniors")
	ErrSelfReference      = errors.New("cannot reference self as senior or junior")
	ErrDuplicateJunior    = errors.New("duplicate junior ID")
	ErrInvalidDailyRep    = errors.New("daily rep cannot exceed 5000")
)

// Pure functions for business logic validation

// ValidateCharacterId validates a character ID
func ValidateCharacterId(characterId uint32) error {
	if characterId == 0 {
		return ErrInvalidCharacterId
	}
	return nil
}

// ValidateTenantId validates a tenant ID
func ValidateTenantId(tenantId uuid.UUID) error {
	if tenantId == uuid.Nil {
		return ErrInvalidTenantId
	}
	return nil
}

// ValidateLevel validates a character level
func ValidateLevel(level uint16) error {
	if level == 0 {
		return ErrInvalidLevel
	}
	return nil
}

// ValidateJuniorIds validates junior IDs list
func ValidateJuniorIds(characterId uint32, juniorIds []uint32) error {
	if len(juniorIds) > 2 {
		return ErrTooManyJuniors
	}
	
	// Check for self-reference
	for _, juniorId := range juniorIds {
		if juniorId == characterId {
			return ErrSelfReference
		}
	}
	
	// Check for duplicates
	seen := make(map[uint32]bool)
	for _, juniorId := range juniorIds {
		if seen[juniorId] {
			return ErrDuplicateJunior
		}
		seen[juniorId] = true
	}
	
	return nil
}

// ValidateSeniorId validates senior ID
func ValidateSeniorId(characterId uint32, seniorId *uint32) error {
	if seniorId != nil && *seniorId == characterId {
		return ErrSelfReference
	}
	return nil
}