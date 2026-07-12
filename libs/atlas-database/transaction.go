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

// isTransaction reports whether the handle is already inside a transaction.
// Inside a real transaction Statement.ConnPool is a *sql.Tx (implements
// gorm.TxCommitter); on the root pool it is *sql.DB, which does not. This is
// GORM's own idiom for the same check in finisher_api.go.
func isTransaction(db *gorm.DB) bool {
	if db.Statement == nil || db.Statement.ConnPool == nil {
		return false
	}
	committer, ok := db.Statement.ConnPool.(gorm.TxCommitter)
	return ok && committer != nil
}
