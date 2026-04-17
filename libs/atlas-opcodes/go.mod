module github.com/Chronicle20/atlas/libs/atlas-opcodes

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-socket v0.0.0
	github.com/sirupsen/logrus v1.9.4
)

require (
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-socket => ../atlas-socket
