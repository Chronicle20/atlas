package reactor

import (
	"time"

	"atlas-data/searchindex"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SearchIndexEntity struct {
	TenantId  uuid.UUID `gorm:"type:uuid;primaryKey"`
	ReactorId uint32    `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (SearchIndexEntity) TableName() string {
	return "reactor_search_index"
}

func Migration(db *gorm.DB) error {
	return searchindex.Migrate(db, &SearchIndexEntity{}, searchindex.MigrationOptions{
		IndexStatements: []string{
			"CREATE INDEX IF NOT EXISTS idx_reactor_search_index_name_trgm ON reactor_search_index USING GIN (LOWER(name) gin_trgm_ops)",
		},
	})
}
