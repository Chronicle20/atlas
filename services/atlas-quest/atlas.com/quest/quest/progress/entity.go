package progress

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId      uuid.UUID `gorm:"not null;index"`
	ID            uint32    `gorm:"primaryKey;autoIncrement;not null"`
	QuestStatusId uint32    `gorm:"not null;index"`
	InfoNumber    uint32    `gorm:"not null"`
	Progress      string    `gorm:"not null;default:''"`
}

func (e Entity) TableName() string {
	return "quest_progress"
}

func Make(e Entity) (Model, error) {
	return Model{
		tenantId:   e.TenantId,
		id:         e.ID,
		infoNumber: e.InfoNumber,
		progress:   e.Progress,
	}, nil
}
