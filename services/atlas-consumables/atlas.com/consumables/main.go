package main

import (
	"atlas-consumables/kafka/consumer/character"
	"atlas-consumables/kafka/consumer/compartment"
	"atlas-consumables/kafka/consumer/consumable"
	"atlas-consumables/logger"
	mapCharacter "atlas-consumables/map/character"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-consumables"
const consumerGroupId = "Consumables Service"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	rc := atlas.Connect(l)
	mapCharacter.InitRegistry(rc)

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	compartment.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	consumable.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := consumable.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		Run()

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
