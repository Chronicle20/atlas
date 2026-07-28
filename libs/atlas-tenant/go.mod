module github.com/Chronicle20/atlas/libs/atlas-tenant

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0
	github.com/google/uuid v1.6.0
)

require (
	github.com/Chronicle20/atlas/libs/atlas-routine v0.0.0-00010101000000-000000000000 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-model => ../atlas-model

replace github.com/Chronicle20/atlas/libs/atlas-routine => ../atlas-routine
