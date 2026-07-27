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
	_map "atlas-messages/command/map"
	"atlas-messages/command/monster"
	party_quest "atlas-messages/command/party_quest"
	commandpet "atlas-messages/command/pet"
	message2 "atlas-messages/kafka/consumer/message"
	"os"

	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-messages"

var consumerGroupId = consumergroup.Resolve("Messages Service")

func main() {
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

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
	command.Registry().Add(monster.MobKillAllCommandProducer)
	command.Registry().Add(monster.MobStatusCommandProducer)
	command.Registry().Add(monster.MobClearCommandProducer)
	command.Registry().Add(monster.MobSpawnCommandProducer)
	command.Registry().Add(commandpet.AwardTamenessCommandProducer)
	command.Registry().Add(disease.DiseaseCommandProducer)
	command.Registry().Add(party_quest.PQRegisterCommandProducer)
	command.Registry().Add(party_quest.PQStageCommandProducer)
	command.Registry().Add(_map.WeatherCommandProducer)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	message2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := message2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
