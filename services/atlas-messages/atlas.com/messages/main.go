package main

import (
	"atlas-messages/logger"
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

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(message.GeneralChatCommandConsumer(l)(consumerGroupId))
	_, _ = cm.RegisterHandler(message.GeneralChatCommandRegister(l))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
