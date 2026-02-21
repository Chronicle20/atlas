package main

import (
	"atlas-monster-death/kafka/consumer/monster"
	"atlas-monster-death/logger"
	"github.com/Chronicle20/atlas-service"
	"atlas-monster-death/tracing"

	"github.com/Chronicle20/atlas-kafka/consumer"
)

const serviceName = "atlas-monster-death"
const consumerGroupId = "Monster Death Service"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	monster.InitConsumers(l)(cmf)(consumerGroupId)
	if err := monster.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
