module github.com/Chronicle20/atlas/libs/atlas-packet

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-constants v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-socket v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0
	github.com/google/uuid v1.6.0
	github.com/sirupsen/logrus v1.9.4
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-constants => ../atlas-constants

replace github.com/Chronicle20/atlas/libs/atlas-socket => ../atlas-socket

replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../atlas-tenant

replace github.com/Chronicle20/atlas/libs/atlas-model => ../atlas-model
