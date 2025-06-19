package main

import (
	"atlas-messages/command"
	"atlas-messages/command/character"
	"atlas-messages/command/character/inventory"
	"atlas-messages/command/character/skill"
	"atlas-messages/command/map"
	message2 "atlas-messages/kafka/consumer/message"
	"atlas-messages/logger"
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

	command.Registry().Add(_map.WarpCommandProducer)
	command.Registry().Add(_map.WhereAmICommandProducer)
	command.Registry().Add(inventory.AwardItemCommandProducer)
	command.Registry().Add(character.AwardExperienceCommandProducer)
	command.Registry().Add(character.AwardLevelCommandProducer)
	command.Registry().Add(character.AwardMesoCommandProducer)
	command.Registry().Add(character.ChangeJobCommandProducer)
	command.Registry().Add(skill.MaxSkillCommandProducer)
	command.Registry().Add(skill.ResetSkillCommandProducer)

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	message2.InitConsumers(l)(cmf)(consumerGroupId)
	message2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
