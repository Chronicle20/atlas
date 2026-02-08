package database

import (
	"gorm.io/gorm"
)

func ExecuteTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if isTransaction(db) {
		return fn(db)
	}
	return db.Transaction(fn)
}

func isTransaction(db *gorm.DB) bool {
	return db.Statement != nil && db.Statement.ConnPool != nil
}
