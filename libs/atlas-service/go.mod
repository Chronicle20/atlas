module github.com/Chronicle20/atlas/libs/atlas-service

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-routine v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.4
	go.elastic.co/ecslogrus v1.0.0
)

require (
	github.com/magefile/mage v1.9.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-routine => ../atlas-routine
