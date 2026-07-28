package database

import "gorm"

func ExecuteTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error { return fn(db) }
