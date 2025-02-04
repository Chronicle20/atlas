package main

import (
	"atlas-chairs/kafka/consumer/chair"
	"atlas-chairs/kafka/consumer/character"
	"atlas-chairs/logger"
	"atlas-chairs/service"
	"atlas-chairs/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
)

const serviceName = "atlas-chairs"
const consumerGroupId = "Chairs Service"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	chair.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	chair.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	character.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
