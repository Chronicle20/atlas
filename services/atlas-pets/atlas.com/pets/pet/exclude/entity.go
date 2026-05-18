package exclude

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	if err := db.AutoMigrate(&Entity{}); err != nil {
		return err
	}
	if !db.Migrator().HasTable("pets") {
		return nil
	}
	return db.Exec(`
		UPDATE excludes
		SET tenant_id = (SELECT tenant_id FROM pets WHERE pets.id = excludes.pet_id)
		WHERE (tenant_id IS NULL OR tenant_id = '00000000-0000-0000-0000-000000000000')
		  AND EXISTS (SELECT 1 FROM pets WHERE pets.id = excludes.pet_id)
	`).Error
}

type Entity struct {
	Id       uint32    `gorm:"primary_key;auto_increment"`
	PetId    uint32    `gorm:"not null"`
	TenantId uuid.UUID `gorm:"index"`
	ItemId   uint32    `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "excludes"
}

func Make(e Entity) (Model, error) {
	return NewModelBuilder(e.ItemId).
		SetId(e.Id).
		Build()
}
