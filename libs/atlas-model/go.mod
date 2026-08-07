module github.com/Chronicle20/atlas/libs/atlas-model

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-routine v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.4
	golang.org/x/sync v0.22.0
)

require golang.org/x/sys v0.13.0 // indirect

replace github.com/Chronicle20/atlas/libs/atlas-routine => ../atlas-routine
