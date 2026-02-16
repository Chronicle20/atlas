package main

import (
	"atlas-messages/command"
	"atlas-messages/command/buff"
	"atlas-messages/command/character"
	"atlas-messages/command/character/inventory"
	"atlas-messages/command/character/skill"
	"atlas-messages/command/consumable"
	"atlas-messages/command/disease"
	"atlas-messages/command/help"
	"atlas-messages/command/map"
	"atlas-messages/command/monster"
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

	command.Registry().Add(help.HelpCommandProducer)
	command.Registry().Add(_map.WarpCommandProducer)
	command.Registry().Add(_map.WhereAmICommandProducer)
	command.Registry().Add(_map.RatesCommandProducer)
	command.Registry().Add(inventory.AwardItemCommandProducer)
	command.Registry().Add(character.AwardExperienceCommandProducer)
	command.Registry().Add(character.AwardLevelCommandProducer)
	command.Registry().Add(character.AwardMesoCommandProducer)
	command.Registry().Add(character.AwardCurrencyCommandProducer)
	command.Registry().Add(character.ChangeJobCommandProducer)
	command.Registry().Add(skill.MaxSkillCommandProducer)
	command.Registry().Add(skill.ResetSkillCommandProducer)
	command.Registry().Add(buff.BuffCommandProducer)
	command.Registry().Add(consumable.ConsumeCommandProducer)
	command.Registry().Add(monster.MobStatusCommandProducer)
	command.Registry().Add(monster.MobClearCommandProducer)
	command.Registry().Add(disease.DiseaseCommandProducer)

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	message2.InitConsumers(l)(cmf)(consumerGroupId)
	message2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
