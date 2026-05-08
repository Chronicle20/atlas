module github.com/Chronicle20/atlas/libs/atlas-object-id

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-redis v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0
	github.com/alicebob/miniredis/v2 v2.37.0
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.19.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-constants => ../atlas-constants

replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../atlas-tenant

replace github.com/Chronicle20/atlas/libs/atlas-model => ../atlas-model

replace github.com/Chronicle20/atlas/libs/atlas-redis => ../atlas-redis
