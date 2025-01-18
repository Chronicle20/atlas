package main

import (
	"atlas-expressions/expression"
	expression2 "atlas-expressions/kafka/consumer/expression"
	_map "atlas-expressions/kafka/consumer/map"
	"atlas-expressions/logger"
	"atlas-expressions/service"
	"atlas-expressions/tasks"
	"atlas-expressions/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"time"
)

const serviceName = "atlas-expressions"
const consumerGroupId = "Expression Service"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(expression2.CommandConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(expression2.ChangeCommandRegister(l))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(_map.StatusEventConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(_map.StatusEventCharacterExitRegister(l))

	go tasks.Register(l, tdm.Context())(expression.NewRevertTask(l, time.Millisecond*50))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
