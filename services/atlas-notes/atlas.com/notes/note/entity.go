package note

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity represents a note in the database
type Entity struct {
	ID          uint32 `gorm:"primaryKey;autoIncrement"`
	TenantId    uuid.UUID
	CharacterID uint32
	SenderID    uint32
	Message     string
	Timestamp   time.Time
	Flag        byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the database table name for Entity
func (Entity) TableName() string {
	return "notes"
}

// Make converts an Entity to a Model domain model
func Make(e Entity) (Model, error) {
	return NewBuilder().
		SetId(e.ID).
		SetCharacterId(e.CharacterID).
		SetSenderId(e.SenderID).
		SetMessage(e.Message).
		SetTimestamp(e.Timestamp).
		SetFlag(e.Flag).
		Build()
}

// MakeEntity converts a Model domain model to an Entity
func MakeEntity(tenantId uuid.UUID, n Model) Entity {
	return Entity{
		ID:          n.Id(),
		TenantId:    tenantId,
		CharacterID: n.CharacterId(),
		SenderID:    n.SenderId(),
		Message:     n.Message(),
		Timestamp:   n.Timestamp(),
		Flag:        n.Flag(),
	}
}

// Migration sets up the notes table in the database
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
