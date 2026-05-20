module github.com/Chronicle20/atlas/libs/atlas-seeder

go 1.25.0

replace github.com/Chronicle20/atlas/libs/atlas-model => ../atlas-model

replace github.com/Chronicle20/atlas/libs/atlas-rest => ../atlas-rest

replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../atlas-tenant

require (
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	gorm.io/datatypes v1.2.7
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	gorm.io/driver/mysql v1.5.6 // indirect
)
