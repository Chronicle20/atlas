package item

import (
	"time"

	"atlas-data/searchindex"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StringSearchIndexEntity struct {
	TenantId  uuid.UUID `gorm:"type:uuid;primaryKey"`
	ItemId    uint32    `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (StringSearchIndexEntity) TableName() string {
	return "item_string_search_index"
}

func StringMigration(db *gorm.DB) error {
	return searchindex.Migrate(db, &StringSearchIndexEntity{}, searchindex.MigrationOptions{
		IndexStatements: []string{
			"CREATE INDEX IF NOT EXISTS idx_item_string_search_index_name_trgm ON item_string_search_index USING GIN (LOWER(name) gin_trgm_ops)",
		},
	})
}
