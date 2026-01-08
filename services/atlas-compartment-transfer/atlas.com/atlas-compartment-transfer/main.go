package main

import (
	csCompartment "atlas-compartment-transfer/kafka/consumer/cashshop/compartment"
	cCompartment "atlas-compartment-transfer/kafka/consumer/character/compartment"
	"atlas-compartment-transfer/kafka/consumer/compartment"
	sCompartment "atlas-compartment-transfer/kafka/consumer/storage/compartment"
	"atlas-compartment-transfer/logger"
	"atlas-compartment-transfer/service"
	"atlas-compartment-transfer/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
)

const serviceName = "atlas-compartment-transfer"
const consumerGroupId = "Compartment Transfer Service"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	compartment.InitConsumers(l)(cmf)(consumerGroupId)
	csCompartment.InitConsumers(l)(cmf)(consumerGroupId)
	cCompartment.InitConsumers(l)(cmf)(consumerGroupId)
	sCompartment.InitConsumers(l)(cmf)(consumerGroupId)
	compartment.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	csCompartment.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	cCompartment.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	sCompartment.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
