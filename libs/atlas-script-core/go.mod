module github.com/Chronicle20/atlas/libs/atlas-script-core

go 1.27.0

require (
	github.com/Chronicle20/atlas/libs/atlas-constants v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-saga v0.0.0
	github.com/google/uuid v1.6.0
)

replace github.com/Chronicle20/atlas/libs/atlas-constants => ../atlas-constants

replace github.com/Chronicle20/atlas/libs/atlas-routine => ../atlas-routine

replace github.com/Chronicle20/atlas/libs/atlas-saga => ../atlas-saga
