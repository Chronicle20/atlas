module github.com/Chronicle20/atlas/libs/atlas-rest

go 1.25.5

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/jtumidanski/api2go v1.0.4
	github.com/prometheus/client_golang v1.24.0
	github.com/sirupsen/logrus v1.9.4
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.0 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	golang.org/x/sync v0.22.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-retry v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-routine v0.0.0-00010101000000-000000000000
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gedex/inflector v0.0.0-20170307190818-16278e9db813 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/tools v0.44.0 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-model => ../atlas-model

replace github.com/Chronicle20/atlas/libs/atlas-retry => ../atlas-retry

replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../atlas-tenant

replace github.com/Chronicle20/atlas/libs/atlas-routine => ../atlas-routine
