package main

import (
	"atlas-expressions/expression"
	expression2 "atlas-expressions/kafka/consumer/expression"
	_map "atlas-expressions/kafka/consumer/map"
	"atlas-expressions/logger"
	"github.com/Chronicle20/atlas-service"
	"atlas-expressions/tasks"
	"atlas-expressions/tracing"
	"time"

	"github.com/Chronicle20/atlas-kafka/consumer"
	atlas "github.com/Chronicle20/atlas-redis"
)

const serviceName = "atlas-expressions"
const consumerGroupId = "Expression Service"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	rc := atlas.Connect(l)
	expression.InitRegistry(rc)

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	expression2.InitConsumers(l)(cmf)(consumerGroupId)
	_map.InitConsumers(l)(cmf)(consumerGroupId)
	if err := expression2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := _map.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	go tasks.Register(l, tdm.Context())(expression.NewRevertTask(l, time.Millisecond*50))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
