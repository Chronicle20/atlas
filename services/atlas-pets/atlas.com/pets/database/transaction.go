package database

import (
	"fmt"
	"gorm.io/gorm"
)

// ExecuteTransaction runs the given function within a transaction.
// If the provided *gorm.DB is already in a transaction, it will just run the function without starting a new one.
func ExecuteTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if isTransaction(db) {
		// Already in a transaction, execute directly
		fmt.Println("Already in transaction, executing directly.")
		return fn(db)
	}

	// Not in a transaction, start a new one
	fmt.Println("Starting new transaction.")
	return db.Transaction(fn)
}

// isTransaction checks if the *gorm.DB is already in a transaction
func isTransaction(db *gorm.DB) bool {
	return db.Statement != nil && db.Statement.ConnPool != nil
}
