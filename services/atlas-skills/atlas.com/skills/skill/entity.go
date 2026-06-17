package skill

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity's primary key is the composite (TenantId, CharacterId, Id): a skill row
// is identified by its owning character within a tenant, NOT by the skill id
// alone. A skill id is shared across every character (e.g. every priest learns
// 2000000), so an id-only PK let the first character claim each skill id and
// every later character collided on skills_pkey (SQLSTATE 23505) — breaking
// character creation for the 2nd+ character that shared any skill. AutoMigrate
// builds the composite key on a fresh database; pre-existing tables (which
// AutoMigrate will not re-key) are reshaped with a one-off DDL ALTER.
type Entity struct {
	TenantId    uuid.UUID `gorm:"primaryKey;not null"`
	CharacterId uint32    `gorm:"primaryKey;not null"`
	Id          uint32    `gorm:"primaryKey;not null"`
	Level       byte      `gorm:"not null"`
	MasterLevel byte      `gorm:"not null"`
	Expiration  time.Time `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "skills"
}

func Make(e Entity) (Model, error) {
	return NewModelBuilder().
		SetId(e.Id).
		SetLevel(e.Level).
		SetMasterLevel(e.MasterLevel).
		SetExpiration(e.Expiration).
		Build()
}
