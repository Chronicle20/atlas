package seeder

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SeedState struct {
	TenantID        uuid.UUID      `gorm:"type:uuid;primaryKey"`
	GroupName       string         `gorm:"type:text;primaryKey"`
	CatalogRevision string         `gorm:"type:text;not null"`
	SeededAt        time.Time      `gorm:"not null"`
	ResultSummary   datatypes.JSON `gorm:"type:jsonb;not null"`
}

func (SeedState) TableName() string { return "seed_state" }

func UpsertSeedState(db *gorm.DB, s *SeedState) error {
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "group_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"catalog_revision",
			"seeded_at",
			"result_summary",
		}),
	}).Create(s).Error
}

func ReadSeedState(db *gorm.DB, tenantID uuid.UUID, groupName string) (*SeedState, error) {
	var out SeedState
	err := db.Where("tenant_id = ? AND group_name = ?", tenantID, groupName).First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}
