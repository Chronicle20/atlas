package record

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// defaultDB holds the *gorm.DB assigned during Migration so the package-level
// REST route initializer (InitResource, whose signature is fixed by plan
// task 10 to take only jsonapi.ServerInformation) can query the game_records
// table without a db handle being curried through route wiring. Every
// persistence function in this package (GetOrZero/GetByCharacter/ApplyResult)
// still takes an explicit *gorm.DB parameter; this var exists solely to give
// the REST handler in resource.go something to call them with.
var defaultDB *gorm.DB

// Migration applies the game_records schema and records the *gorm.DB for use
// by InitResource's handlers.
func Migration(db *gorm.DB) error {
	defaultDB = db
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
