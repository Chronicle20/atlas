package outbox

import "gorm.io/gorm"

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
