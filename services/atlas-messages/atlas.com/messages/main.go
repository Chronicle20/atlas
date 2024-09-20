package main

import (
	"atlas-messages/character/inventory"
	"atlas-messages/command"
	"atlas-messages/logger"
	_map "atlas-messages/map"
	"atlas-messages/message"
	"atlas-messages/service"
	"atlas-messages/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
)

const serviceName = "atlas-messages"
const consumerGroupId = "Messages Service"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	command.Registry().Add(_map.WarpMapCommandProducer)
	command.Registry().Add(inventory.AwardItemCommandProducer)

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(message.GeneralChatCommandConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(message.GeneralChatCommandRegister(l))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
