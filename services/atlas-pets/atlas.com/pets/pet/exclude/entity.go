package exclude

import (
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id     uint32 `gorm:"primary_key;auto_increment"`
	PetId  uint32 `gorm:"not null"`
	ItemId uint32 `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "excludes"
}

func Make(e Entity) (Model, error) {
	return NewModelBuilder(e.ItemId).
		SetId(e.Id).
		Build()
}
