package wallet

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id        uuid.UUID `gorm:"primaryKey;type:uuid"`
	TenantId  uuid.UUID `gorm:"not null"`
	AccountId uint32    `gorm:"not null"`
	Credit    uint32    `gorm:"not null;default=0"`
	Points    uint32    `gorm:"not null;default=0"`
	Prepaid   uint32    `gorm:"not null;default=0"`
}

func (e Entity) TableName() string {
	return "accounts"
}

func (e *Entity) BeforeCreate(_ *gorm.DB) (err error) {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return
}

func Make(e Entity) (Model, error) {
	return Model{
		id:        e.Id,
		accountId: e.AccountId,
		credit:    e.Credit,
		points:    e.Points,
		prepaid:   e.Prepaid,
	}, nil
}
