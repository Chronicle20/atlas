package wish

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migration creates the wish_entries table. It is a brand-new table (no legacy
// primary-key rewrite), so AutoMigrate alone produces the correct surrogate-key
// shape and the composite index declared on the entity tags.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

// entity is the GORM row for a wish-list entry.
//
// The primary key is a surrogate UUID (Id); business identity is never the key,
// and a (tenant_id, id) unique index keeps the row tenant-scoped — never a
// unique index on tenant_id alone, which would cap a tenant at one wish entry.
//
// One composite index backs the design's hot query:
//   - (tenant_id, character_id) — a character's wish list
type entity struct {
	Id          uuid.UUID `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:idx_wish_entries_tenant_id,priority:2"`
	TenantId    uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_wish_entries_tenant_id,priority:1;index:idx_wish_entries_character,priority:1"`
	CharacterId uint32    `gorm:"column:character_id;not null;index:idx_wish_entries_character,priority:2"`
	ItemId      uint32    `gorm:"column:item_id;not null"`

	CreatedAt time.Time `gorm:"column:created_at"`
}

func (e entity) TableName() string {
	return "wish_entries"
}
