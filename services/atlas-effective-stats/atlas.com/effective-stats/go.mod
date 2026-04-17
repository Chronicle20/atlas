module atlas-effective-stats

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-constants v0.0.0
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
	go.elastic.co/ecslogrus v1.0.0
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
)

require (
	github.com/Chronicle20/atlas/libs/atlas-retry v0.0.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gedex/inflector v0.0.0-20170307190818-16278e9db813 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/grpc v1.80.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-constants => ../../../../libs/atlas-constants

replace github.com/Chronicle20/atlas/libs/atlas-kafka => ../../../../libs/atlas-kafka

replace github.com/Chronicle20/atlas/libs/atlas-model => ../../../../libs/atlas-model

replace github.com/Chronicle20/atlas/libs/atlas-rest => ../../../../libs/atlas-rest

replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../../../../libs/atlas-tenant

replace github.com/Chronicle20/atlas/libs/atlas-redis => ../../../../libs/atlas-redis

replace github.com/Chronicle20/atlas/libs/atlas-service => ../../../../libs/atlas-service

replace github.com/Chronicle20/atlas/libs/atlas-database => ../../../../libs/atlas-database

replace github.com/Chronicle20/atlas/libs/atlas-opcodes => ../../../../libs/atlas-opcodes

replace github.com/Chronicle20/atlas/libs/atlas-packet => ../../../../libs/atlas-packet

replace github.com/Chronicle20/atlas/libs/atlas-retry => ../../../../libs/atlas-retry

replace github.com/Chronicle20/atlas/libs/atlas-saga => ../../../../libs/atlas-saga

replace github.com/Chronicle20/atlas/libs/atlas-script-core => ../../../../libs/atlas-script-core

replace github.com/Chronicle20/atlas/libs/atlas-socket => ../../../../libs/atlas-socket
