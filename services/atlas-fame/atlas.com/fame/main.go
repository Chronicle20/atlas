package main

import (
	"atlas-fame/fame"
	"atlas-fame/kafka/consumer/character"
	fame2 "atlas-fame/kafka/consumer/fame"
	"atlas-fame/logger"
	"github.com/Chronicle20/atlas-service"
	"atlas-fame/tracing"

	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-kafka/consumer"
)

const serviceName = "atlas-fame"
const consumerGroupId = "Fame Service"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(fame.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	fame2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := fame2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	character.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
