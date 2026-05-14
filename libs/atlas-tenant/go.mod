module github.com/Chronicle20/atlas/libs/atlas-tenant

go 1.25.0

require (
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0
	github.com/google/uuid v1.6.0
)

require golang.org/x/sync v0.20.0 // indirect

replace github.com/Chronicle20/atlas/libs/atlas-model => ../atlas-model
