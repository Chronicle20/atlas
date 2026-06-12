module atlas-mounts

go 1.25.5

require (
	github.com/google/uuid v1.6.0
	github.com/sirupsen/logrus v1.9.4
	go.elastic.co/ecslogrus v1.0.0
	gorm.io/gorm v1.31.1
)

require github.com/stretchr/testify v1.11.1 // indirect

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.20.0 // indirect
)

replace github.com/Chronicle20/atlas/libs/atlas-constants => ../../../../libs/atlas-constants

replace github.com/Chronicle20/atlas/libs/atlas-kafka => ../../../../libs/atlas-kafka

replace github.com/Chronicle20/atlas/libs/atlas-model => ../../../../libs/atlas-model

replace github.com/Chronicle20/atlas/libs/atlas-rest => ../../../../libs/atlas-rest

replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../../../../libs/atlas-tenant

replace github.com/Chronicle20/atlas/libs/atlas-database => ../../../../libs/atlas-database

replace github.com/Chronicle20/atlas/libs/atlas-redis => ../../../../libs/atlas-redis

replace github.com/Chronicle20/atlas/libs/atlas-service => ../../../../libs/atlas-service

replace github.com/Chronicle20/atlas/libs/atlas-opcodes => ../../../../libs/atlas-opcodes

replace github.com/Chronicle20/atlas/libs/atlas-packet => ../../../../libs/atlas-packet

replace github.com/Chronicle20/atlas/libs/atlas-retry => ../../../../libs/atlas-retry

replace github.com/Chronicle20/atlas/libs/atlas-saga => ../../../../libs/atlas-saga

replace github.com/Chronicle20/atlas/libs/atlas-script-core => ../../../../libs/atlas-script-core

replace github.com/Chronicle20/atlas/libs/atlas-socket => ../../../../libs/atlas-socket

replace github.com/Chronicle20/atlas/libs/atlas-tracing => ../../../../libs/atlas-tracing
