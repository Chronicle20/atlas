package serial

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// entity is the per-(tenant, world) counter row. NextSerial holds the LAST
// assigned serial; Next advances it (NextSerial = NextSerial + 1) and returns
// the new value, so the first serial assigned in a (tenant, world) is 1.
//
// The composite primary key (tenant_id, world_id) makes the counter row unique
// per world and lets the seed upsert use ON CONFLICT DO NOTHING. tenant_id is a
// real column so the tenant query/create callbacks scope it automatically — but
// Next deliberately drives every counter access through an explicit name-keyed
// WHERE rather than relying on the callback, because world 0 is a valid world
// and a struct condition would elide it.
type entity struct {
	TenantId   uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;primaryKey"`
	WorldId    byte      `gorm:"column:world_id;not null;primaryKey"`
	NextSerial uint32    `gorm:"column:next_serial;not null"`
}

func (entity) TableName() string {
	return "mts_serials"
}

// Migration creates the mts_serials counter table. Additive — a new table, so
// AutoMigrate alone produces the correct shape.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}
