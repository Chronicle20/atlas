module github.com/Chronicle20/atlas/libs/atlas-seeder

go 1.25.0

replace github.com/Chronicle20/atlas/libs/atlas-model => ../atlas-model

replace github.com/Chronicle20/atlas/libs/atlas-rest => ../atlas-rest

replace github.com/Chronicle20/atlas/libs/atlas-retry => ../atlas-retry

replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../atlas-tenant

require (
	github.com/Chronicle20/atlas/libs/atlas-rest v0.0.0-00010101000000-000000000000
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/prometheus/client_golang v1.23.2
	github.com/sirupsen/logrus v1.9.4
	golang.org/x/sync v0.21.0
	gorm.io/datatypes v1.2.7
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/gedex/inflector v0.0.0-20170307190818-16278e9db813 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jtumidanski/api2go v1.0.4 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gorm.io/driver/mysql v1.5.6 // indirect
)
