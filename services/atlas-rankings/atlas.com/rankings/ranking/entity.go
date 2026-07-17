package ranking

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{}, &CycleEntity{})
}

// Entity is one ranked character. 0 is never stored as a rank — unranked
// characters simply have no row.
type Entity struct {
	TenantId        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_rankings_tenant_character;index:idx_rankings_tenant_world"`
	Id              uuid.UUID `gorm:"type:uuid;primaryKey"`
	CharacterId     uint32    `gorm:"not null;uniqueIndex:idx_rankings_tenant_character"`
	WorldId         world.Id  `gorm:"not null;index:idx_rankings_tenant_world"`
	JobCategory     uint16    `gorm:"not null"`
	OverallRank     uint32    `gorm:"not null"`
	OverallRankMove int32     `gorm:"not null"`
	JobRank         uint32    `gorm:"not null"`
	JobRankMove     int32     `gorm:"not null"`
	ComputedAt      time.Time `gorm:"not null"`
}

func (e *Entity) BeforeCreate(_ *gorm.DB) error {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return nil
}

func (e Entity) TableName() string {
	return "character_rankings"
}

// CycleEntity tracks recompute cadence and observability per tenant. It
// exists (rather than MAX(computed_at)) so a tenant with zero eligible
// characters still records cycle progress and does not busy-loop.
type CycleEntity struct {
	TenantId         uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Id               uuid.UUID `gorm:"type:uuid;primaryKey"`
	LastStartedAt    time.Time `gorm:"not null"`
	LastCompletedAt  *time.Time
	CharactersRanked uint32
	DurationMs       uint32
}

func (e *CycleEntity) BeforeCreate(_ *gorm.DB) error {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return nil
}

func (e CycleEntity) TableName() string {
	return "ranking_cycles"
}

func Make(e Entity) (Model, error) {
	return NewBuilder().
		SetCharacterId(e.CharacterId).
		SetWorldId(e.WorldId).
		SetJobCategory(e.JobCategory).
		SetOverallRank(e.OverallRank).
		SetOverallRankMove(e.OverallRankMove).
		SetJobRank(e.JobRank).
		SetJobRankMove(e.JobRankMove).
		SetComputedAt(e.ComputedAt).
		Build(), nil
}

func MakeCycle(e CycleEntity) (CycleModel, error) {
	return CycleModel{
		lastStartedAt:    e.LastStartedAt,
		lastCompletedAt:  e.LastCompletedAt,
		charactersRanked: e.CharactersRanked,
		durationMs:       e.DurationMs,
	}, nil
}
