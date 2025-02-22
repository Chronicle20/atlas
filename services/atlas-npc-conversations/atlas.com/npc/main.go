package main

import (
	"atlas-npc-conversations/configuration"
	"atlas-npc-conversations/conversation/script/registry"
	"atlas-npc-conversations/kafka/consumer/character"
	"atlas-npc-conversations/kafka/consumer/npc"
	"atlas-npc-conversations/logger"
	"atlas-npc-conversations/service"
	"atlas-npc-conversations/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/google/uuid"
	"os"
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

	configuration.Init(l)(tdm.Context())(uuid.MustParse(os.Getenv("SERVICE_ID")))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	character.InitConsumers(l)(cmf)(consumerGroupId)
	npc.InitConsumers(l)(cmf)(consumerGroupId)

	character.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	npc.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	for _, s := range configuration.GetTenantConfigs() {
		for _, sct := range s.NPCs {
			registry.GetRegistry().InitScript(uuid.MustParse(s.Id), sct.NPCId, sct.Impl)
		}
	}

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
