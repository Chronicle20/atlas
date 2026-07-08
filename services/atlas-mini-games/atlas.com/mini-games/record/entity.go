package record

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migration applies the game_records schema.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity is the GORM row for a character's win/tie/loss record for one
// mini-game type. A surrogate uuid PK plus a (tenant_id, character_id,
// game_type) unique index avoids the slug-only-PK tenant collision bug
// (see bug_tenant_table_slug_only_pk_collides.md).
type Entity struct {
	TenantId    uuid.UUID `gorm:"not null;uniqueIndex:idx_record_tenant_char_game"`
	Id          uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	CharacterId uint32    `gorm:"not null;uniqueIndex:idx_record_tenant_char_game"`
	GameType    string    `gorm:"not null;uniqueIndex:idx_record_tenant_char_game"`
	Wins        uint32    `gorm:"not null;default:0"`
	Ties        uint32    `gorm:"not null;default:0"`
	Losses      uint32    `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (e Entity) TableName() string {
	return "game_records"
}

// Make converts a persisted Entity into an immutable Model.
func Make(e Entity) (Model, error) {
	return Model{
		tenantId:    e.TenantId,
		id:          e.Id,
		characterId: e.CharacterId,
		gameType:    GameType(e.GameType),
		wins:        e.Wins,
		ties:        e.Ties,
		losses:      e.Losses,
	}, nil
}
