package npc

import (
	"time"

	"atlas-data/searchindex"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SearchIndexEntity struct {
	TenantId  uuid.UUID `gorm:"type:uuid;primaryKey"`
	NpcId     uint32    `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	Storebank bool      `gorm:"not null;default:false"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (SearchIndexEntity) TableName() string {
	return "npc_search_index"
}

func Migration(db *gorm.DB) error {
	return searchindex.Migrate(db, &SearchIndexEntity{}, searchindex.MigrationOptions{
		IndexStatements: []string{
			"CREATE INDEX IF NOT EXISTS idx_npc_search_index_name_trgm ON npc_search_index USING GIN (LOWER(name) gin_trgm_ops)",
			"CREATE INDEX IF NOT EXISTS idx_npc_search_index_storebank ON npc_search_index (tenant_id, storebank) WHERE storebank = true",
		},
	})
}
