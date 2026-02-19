package main

import (
	"atlas-inventory/asset"
	"atlas-inventory/compartment"
	database "github.com/Chronicle20/atlas-database"
	"atlas-inventory/inventory"
	"atlas-inventory/kafka/consumer/character"
	compartment2 "atlas-inventory/kafka/consumer/compartment"
	"atlas-inventory/kafka/consumer/drop"
	"atlas-inventory/logger"
	"atlas-inventory/service"
	"atlas-inventory/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-inventory"
const consumerGroupId = "Inventory Service"

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	rc := atlas.Connect(l)
	compartment.InitReservationRegistry(rc)
	compartment.InitLockRegistry(rc)

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(compartment.Migration, asset.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	character.InitConsumers(l)(cmf)(consumerGroupId)
	compartment2.InitConsumers(l)(cmf)(consumerGroupId)
	drop.InitConsumers(l)(cmf)(consumerGroupId)

	character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	compartment2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	drop.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(inventory.InitResource(GetServer())(db)).
		AddRouteInitializer(compartment.InitResource(GetServer())(db)).
		AddRouteInitializer(asset.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
