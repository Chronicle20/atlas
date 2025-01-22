package main

import (
	"atlas-npc-conversations/configuration"
	"atlas-npc-conversations/conversation/script/registry"
	"atlas-npc-conversations/kafka/consumer/npc"
	"atlas-npc-conversations/logger"
	"atlas-npc-conversations/service"
	"atlas-npc-conversations/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/google/uuid"
)

const serviceName = "atlas-npc-conversations"
const consumerGroupId = "NPC Conversation Service"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	config, err := configuration.GetConfiguration()
	if err != nil {
		l.WithError(err).Fatal("Unable to successfully load configuration.")
	}

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(npc.CommandConsumer(l)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler(npc.StartConversationCommandRegister(l))
	_, _ = cm.RegisterHandler(npc.ContinueConversationCommandRegister(l))
	_, _ = cm.RegisterHandler(npc.EndConversationCommandRegister(l))

	for _, s := range config.Data.Attributes.Servers {
		for _, sct := range s.Scripts {
			registry.GetRegistry().InitScript(uuid.MustParse(s.Tenant), sct.NPCId, sct.Impl)
		}
	}

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
