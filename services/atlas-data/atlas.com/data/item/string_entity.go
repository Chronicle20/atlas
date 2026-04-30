package item

import (
	"time"

	"atlas-data/searchindex"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StringSearchIndexEntity struct {
	TenantId    uuid.UUID      `gorm:"type:uuid;primaryKey"`
	ItemId      uint32         `gorm:"primaryKey"`
	Name        string         `gorm:"not null"`
	Compartment inventory.Type `gorm:"not null;default:0"`
	Subcategory string         `gorm:"not null;default:''"`
	JobMask     *uint8
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (StringSearchIndexEntity) TableName() string {
	return "item_string_search_index"
}

func StringMigration(db *gorm.DB) error {
	return searchindex.Migrate(db, &StringSearchIndexEntity{}, searchindex.MigrationOptions{
		IndexStatements: []string{
			"CREATE INDEX IF NOT EXISTS idx_item_string_search_index_name_trgm ON item_string_search_index USING GIN (LOWER(name) gin_trgm_ops)",
			"ALTER TABLE item_string_search_index ADD COLUMN IF NOT EXISTS compartment SMALLINT NOT NULL DEFAULT 0",
			"ALTER TABLE item_string_search_index ADD COLUMN IF NOT EXISTS subcategory TEXT NOT NULL DEFAULT ''",
			"ALTER TABLE item_string_search_index ADD COLUMN IF NOT EXISTS job_mask SMALLINT NULL",
			"CREATE INDEX IF NOT EXISTS idx_item_string_search_index_compartment ON item_string_search_index (tenant_id, compartment)",
			"CREATE INDEX IF NOT EXISTS idx_item_string_search_index_subcategory ON item_string_search_index (tenant_id, compartment, subcategory)",
		},
	})
}
