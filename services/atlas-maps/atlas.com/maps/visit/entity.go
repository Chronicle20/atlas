package visit

import (
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Entity struct {
	ID             uuid.UUID `gorm:"primaryKey;column:id;type:uuid"`
	TenantId       uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_visits_tenant_char_map,priority:1;index:idx_visits_tenant_char,priority:1"`
	CharacterID    uint32    `gorm:"column:character_id;not null;uniqueIndex:idx_visits_tenant_char_map,priority:2;index:idx_visits_tenant_char,priority:2"`
	MapID          uint32    `gorm:"column:map_id;not null;uniqueIndex:idx_visits_tenant_char_map,priority:3"`
	FirstVisitedAt time.Time `gorm:"column:first_visited_at;not null;default:CURRENT_TIMESTAMP"`
}

func (Entity) TableName() string {
	return "character_map_visits"
}

func Make(e Entity) (Visit, error) {
	return Visit{
		characterId:    e.CharacterID,
		mapId:          _map.Id(e.MapID),
		firstVisitedAt: e.FirstVisitedAt,
	}, nil
}

func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
