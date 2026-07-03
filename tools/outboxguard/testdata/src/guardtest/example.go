package guardtest

import (
	"gorm"
	"producer"
	"txdb"
)

func bad(db *gorm.DB) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		_ = producer.ProviderImpl(nil) // want "producer.ProviderImpl inside a DB transaction closure"
		return nil
	})
}

func alsoBad(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		_ = producer.ProviderImpl(nil) // want "producer.ProviderImpl inside a DB transaction closure"
		return nil
	})
}

func good(db *gorm.DB) error {
	err := database.ExecuteTransaction(db, func(tx *gorm.DB) error { return nil })
	_ = producer.ProviderImpl(nil)
	return err
}
