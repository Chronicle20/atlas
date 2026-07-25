package gorm

type DB struct{}

func (d *DB) Transaction(fn func(tx *DB) error) error { return fn(d) }
