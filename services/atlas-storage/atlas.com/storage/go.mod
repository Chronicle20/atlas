module atlas-storage

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-constants v0.0.0-00010101000000-000000000000
	github.com/Chronicle20/atlas/libs/atlas-database v0.0.0-00010101000000-000000000000
	github.com/Chronicle20/atlas/libs/atlas-kafka v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-redis v0.0.0-00010101000000-000000000000
	github.com/Chronicle20/atlas/libs/atlas-rest v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-service v0.0.0-00010101000000-000000000000
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0
	github.com/alicebob/miniredis/v2 v2.37.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/jtumidanski/api2go v1.0.4
	github.com/redis/go-redis/v9 v9.18.0
	github.com/segmentio/kafka-go v0.4.50
	github.com/sirupsen/logrus v1.9.4
	github.com/stretchr/testify v1.11.1
	go.elastic.co/ecslogrus v1.0.0
	go.opentelemetry.io/otel v1.42.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.42.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.42.0
	go.opentelemetry.io/otel/sdk v1.42.0
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/Chronicle20/atlas/libs/atlas-retry v0.0.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gedex/inflector v0.0.0-20170307190818-16278e9db813 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.4 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.24 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260209200024-4cfbd4190f57 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260209200024-4cfbd4190f57 // indirect
	google.golang.org/grpc v1.79.2 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-kafka => ../../../../libs/atlas-kafka

replace github.com/Chronicle20/atlas/libs/atlas-model => ../../../../libs/atlas-model

replace github.com/Chronicle20/atlas/libs/atlas-rest => ../../../../libs/atlas-rest

replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../../../../libs/atlas-tenant

replace github.com/Chronicle20/atlas/libs/atlas-constants => ../../../../libs/atlas-constants

replace github.com/Chronicle20/atlas/libs/atlas-database => ../../../../libs/atlas-database

replace github.com/Chronicle20/atlas/libs/atlas-redis => ../../../../libs/atlas-redis

replace github.com/Chronicle20/atlas/libs/atlas-service => ../../../../libs/atlas-service

replace github.com/Chronicle20/atlas/libs/atlas-opcodes => ../../../../libs/atlas-opcodes

replace github.com/Chronicle20/atlas/libs/atlas-packet => ../../../../libs/atlas-packet

replace github.com/Chronicle20/atlas/libs/atlas-retry => ../../../../libs/atlas-retry

replace github.com/Chronicle20/atlas/libs/atlas-saga => ../../../../libs/atlas-saga

replace github.com/Chronicle20/atlas/libs/atlas-script-core => ../../../../libs/atlas-script-core

replace github.com/Chronicle20/atlas/libs/atlas-socket => ../../../../libs/atlas-socket
