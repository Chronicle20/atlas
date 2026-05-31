module github.com/Chronicle20/atlas/libs/atlas-redis

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.20.0
	github.com/sirupsen/logrus v1.9.4
)

require (
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/klauspost/cpuid/v2 v2.2.11 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../atlas-tenant

replace github.com/Chronicle20/atlas/libs/atlas-model => ../atlas-model
