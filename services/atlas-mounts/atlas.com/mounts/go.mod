module atlas-mounts

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-constants v0.0.0-00010101000000-000000000000
	github.com/Chronicle20/atlas/libs/atlas-kafka v0.0.0-00010101000000-000000000000
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0
	github.com/google/uuid v1.6.0
	github.com/segmentio/kafka-go v0.4.51
	github.com/sirupsen/logrus v1.9.4
	go.elastic.co/ecslogrus v1.0.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/Chronicle20/atlas/libs/atlas-retry v0.0.0 // indirect
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.23.0 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-constants => ../../../../libs/atlas-constants

replace github.com/Chronicle20/atlas/libs/atlas-kafka => ../../../../libs/atlas-kafka

replace github.com/Chronicle20/atlas/libs/atlas-model => ../../../../libs/atlas-model

replace github.com/Chronicle20/atlas/libs/atlas-rest => ../../../../libs/atlas-rest

replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../../../../libs/atlas-tenant

replace github.com/Chronicle20/atlas/libs/atlas-database => ../../../../libs/atlas-database

replace github.com/Chronicle20/atlas/libs/atlas-redis => ../../../../libs/atlas-redis

replace github.com/Chronicle20/atlas/libs/atlas-service => ../../../../libs/atlas-service

replace github.com/Chronicle20/atlas/libs/atlas-opcodes => ../../../../libs/atlas-opcodes

replace github.com/Chronicle20/atlas/libs/atlas-packet => ../../../../libs/atlas-packet

replace github.com/Chronicle20/atlas/libs/atlas-retry => ../../../../libs/atlas-retry

replace github.com/Chronicle20/atlas/libs/atlas-saga => ../../../../libs/atlas-saga

replace github.com/Chronicle20/atlas/libs/atlas-script-core => ../../../../libs/atlas-script-core

replace github.com/Chronicle20/atlas/libs/atlas-socket => ../../../../libs/atlas-socket

replace github.com/Chronicle20/atlas/libs/atlas-tracing => ../../../../libs/atlas-tracing
