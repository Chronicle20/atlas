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

// Migration performs the database migration for the marriage entity
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
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