package database

import (
	"gorm.io/gorm"
)

// ExecuteTransaction runs the given function within a transaction.
// If the provided *gorm.DB is already in a transaction, it will just run the function without starting a new one.
func ExecuteTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if isTransaction(db) {
		return fn(db)
	}
	return db.Transaction(fn)
}

func isTransaction(db *gorm.DB) bool {
	return db.Statement != nil && db.Statement.ConnPool != nil
}
