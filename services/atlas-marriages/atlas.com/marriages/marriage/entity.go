package marriage

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity represents the GORM-compatible database representation of a marriage
type Entity struct {
	ID           uint32         `gorm:"primaryKey;autoIncrement"`
	CharacterId1 uint32         `gorm:"index;not null"`
	CharacterId2 uint32         `gorm:"index;not null"`
	Status       MarriageStatus `gorm:"index;not null"`
	ProposedAt   time.Time      `gorm:"not null"`
	EngagedAt    *time.Time     `gorm:"index"`
	MarriedAt    *time.Time     `gorm:"index"`
	DivorcedAt   *time.Time     `gorm:"index"`
	TenantId     uuid.UUID      `gorm:"type:uuid;index;not null"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
}

// TableName returns the table name for the marriage entity
func (Entity) TableName() string {
	return "marriages"
}

// Migration performs the database migration for the marriage entity and proposal entity
func Migration(db *gorm.DB) error {
	if err := db.AutoMigrate(&Entity{}); err != nil {
		return err
	}
	return db.AutoMigrate(&ProposalEntity{})
}

// Make transforms a marriage entity to a domain model
func Make(entity Entity) (Marriage, error) {
	return NewBuilder(entity.CharacterId1, entity.CharacterId2, entity.TenantId).
		SetId(entity.ID).
		SetStatus(entity.Status).
		SetProposedAt(entity.ProposedAt).
		SetEngagedAt(entity.EngagedAt).
		SetMarriedAt(entity.MarriedAt).
		SetDivorcedAt(entity.DivorcedAt).
		SetCreatedAt(entity.CreatedAt).
		SetUpdatedAt(entity.UpdatedAt).
		Build()
}

// ToEntity converts a marriage domain model to a database entity
func (m Marriage) ToEntity() Entity {
	return Entity{
		ID:           m.id,
		CharacterId1: m.characterId1,
		CharacterId2: m.characterId2,
		Status:       m.status,
		ProposedAt:   m.proposedAt,
		EngagedAt:    m.engagedAt,
		MarriedAt:    m.marriedAt,
		DivorcedAt:   m.divorcedAt,
		TenantId:     m.tenantId,
		CreatedAt:    m.createdAt,
		UpdatedAt:    m.updatedAt,
	}
}

// ProposalEntity represents the GORM-compatible database representation of a proposal
type ProposalEntity struct {
	ID             uint32         `gorm:"primaryKey;autoIncrement"`
	ProposerId     uint32         `gorm:"index;not null"`
	TargetId       uint32         `gorm:"index;not null"`
	Status         ProposalStatus `gorm:"index;not null"`
	ProposedAt     time.Time      `gorm:"not null"`
	RespondedAt    *time.Time     `gorm:"index"`
	ExpiresAt      time.Time      `gorm:"index;not null"`
	RejectionCount uint32         `gorm:"default:0"`
	CooldownUntil  *time.Time     `gorm:"index"`
	TenantId       uuid.UUID      `gorm:"type:uuid;index;not null"`
	CreatedAt      time.Time      `gorm:"not null"`
	UpdatedAt      time.Time      `gorm:"not null"`
}

// TableName returns the table name for the proposal entity
func (ProposalEntity) TableName() string {
	return "proposals"
}

// ProposalMigration performs the database migration for the proposal entity
func ProposalMigration(db *gorm.DB) error {
	return db.AutoMigrate(&ProposalEntity{})
}

// MakeProposal transforms a proposal entity to a domain model
func MakeProposal(entity ProposalEntity) (Proposal, error) {
	return NewProposalBuilder(entity.ProposerId, entity.TargetId, entity.TenantId).
		SetId(entity.ID).
		SetStatus(entity.Status).
		SetProposedAt(entity.ProposedAt).
		SetRespondedAt(entity.RespondedAt).
		SetExpiresAt(entity.ExpiresAt).
		SetRejectionCount(entity.RejectionCount).
		SetCooldownUntil(entity.CooldownUntil).
		SetCreatedAt(entity.CreatedAt).
		SetUpdatedAt(entity.UpdatedAt).
		Build()
}

// ToProposalEntity converts a proposal domain model to a database entity
func (p Proposal) ToProposalEntity() ProposalEntity {
	return ProposalEntity{
		ID:             p.id,
		ProposerId:     p.proposerId,
		TargetId:       p.targetId,
		Status:         p.status,
		ProposedAt:     p.proposedAt,
		RespondedAt:    p.respondedAt,
		ExpiresAt:      p.expiresAt,
		RejectionCount: p.rejectionCount,
		CooldownUntil:  p.cooldownUntil,
		TenantId:       p.tenantId,
		CreatedAt:      p.createdAt,
		UpdatedAt:      p.updatedAt,
	}
}