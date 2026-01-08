package database

import "gorm.io/gorm"

func ExecuteTransaction(db *gorm.DB, f func(tx *gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return f(tx)
	})
}
