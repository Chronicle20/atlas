package main

import (
	"atlas-saga-orchestrator/kafka/consumer/asset"
	"atlas-saga-orchestrator/kafka/consumer/buddylist"
	"atlas-saga-orchestrator/kafka/consumer/cashshop"
	cashshopCompartment "atlas-saga-orchestrator/kafka/consumer/cashshop/compartment"
	"atlas-saga-orchestrator/kafka/consumer/character"
	"atlas-saga-orchestrator/kafka/consumer/compartment"
	"atlas-saga-orchestrator/kafka/consumer/consumable"
	"atlas-saga-orchestrator/kafka/consumer/guild"
	"atlas-saga-orchestrator/kafka/consumer/pet"
	saga2 "atlas-saga-orchestrator/kafka/consumer/saga"
	"atlas-saga-orchestrator/kafka/consumer/skill"
	"atlas-saga-orchestrator/kafka/consumer/storage"
	storageCompartment "atlas-saga-orchestrator/kafka/consumer/storage/compartment"
	"atlas-saga-orchestrator/logger"
	"atlas-saga-orchestrator/saga"
	"atlas-saga-orchestrator/service"
	"atlas-saga-orchestrator/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"os"
)

const serviceName = "atlas-saga-orchestrator"
const consumerGroupId = "Saga Orchestrator Service"

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

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	asset.InitConsumers(l)(cmf)(consumerGroupId)
	buddylist.InitConsumers(l)(cmf)(consumerGroupId)
	cashshop.InitConsumers(l)(cmf)(consumerGroupId)
	cashshopCompartment.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	compartment.InitConsumers(l)(cmf)(consumerGroupId)
	consumable.InitConsumers(l)(cmf)(consumerGroupId)
	guild.InitConsumers(l)(cmf)(consumerGroupId)
	pet.InitConsumers(l)(cmf)(consumerGroupId)
	saga2.InitConsumers(l)(cmf)(consumerGroupId)
	skill.InitConsumers(l)(cmf)(consumerGroupId)
	storage.InitConsumers(l)(cmf)(consumerGroupId)
	storageCompartment.InitConsumers(l)(cmf)(consumerGroupId)
	asset.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	buddylist.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	cashshop.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	cashshopCompartment.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	character.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	compartment.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	consumable.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	guild.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	pet.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	saga2.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	skill.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	storage.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	storageCompartment.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	// Create the service with the router
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(saga.InitResource(GetServer())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
