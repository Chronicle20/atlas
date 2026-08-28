module github.com/Chronicle20/atlas/libs/atlas-opcodes

go 1.27.0

require (
	github.com/Chronicle20/atlas/libs/atlas-socket v0.0.0
	github.com/sirupsen/logrus v1.10.2
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/Chronicle20/atlas/libs/atlas-routine v0.0.0-00010101000000-000000000000 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-socket => ../atlas-socket

replace github.com/Chronicle20/atlas/libs/atlas-routine => ../atlas-routine
