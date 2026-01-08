package storage

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Entity struct {
	TenantId  uuid.UUID `gorm:"not null;uniqueIndex:idx_tenant_world_account"`
	Id        uuid.UUID `gorm:"primaryKey;type:uuid"`
	WorldId   byte      `gorm:"not null;uniqueIndex:idx_tenant_world_account"`
	AccountId uint32    `gorm:"not null;uniqueIndex:idx_tenant_world_account"`
	Capacity  uint32    `gorm:"not null;default:4"`
	Mesos     uint32    `gorm:"not null;default:0"`
}

func (e Entity) TableName() string {
	return "storages"
}

func (e *Entity) BeforeCreate(tx *gorm.DB) error {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return nil
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Make converts an Entity to a Model
func Make(e Entity) Model {
	return NewModelBuilder().
		SetId(e.Id).
		SetWorldId(e.WorldId).
		SetAccountId(e.AccountId).
		SetCapacity(e.Capacity).
		SetMesos(e.Mesos).
		Build()
}
