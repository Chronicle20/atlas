package main

import (
	"atlas-messages/chat"
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
	commandplayernpc "atlas-messages/command/playernpc"
	message2 "atlas-messages/kafka/consumer/message"
	consumerplayernpc "atlas-messages/kafka/consumer/playernpc"
	"os"

	service "github.com/Chronicle20/atlas/libs/atlas-service"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-messages"

var consumerGroupId = consumergroup.Resolve("Messages Service")

// Server carries the JSON:API base URL/prefix used to marshal resource
// links. /api/chat/history (chat history for atlas-ban report corroboration)
// is routed through nginx/ingress (deploy/shared/routes.conf) and is
// reachable at the ingress host WITHOUT AUTHENTICATION today — it exposes
// captured player chat, including whispers, to anything that can reach that
// host. Accepted on the basis that API authentication is coming; see
// docs/tasks/task-145-player-reports/scope-amendment.md Amendment 2. Do not
// describe this endpoint as server-to-server only.
type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}

func main() {
	rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
	l := rt.Logger()

	rc := atlasredis.Connect(l)
	chat.InitRegistry(rc)

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
	command.Registry().Add(commandplayernpc.DeployCommandProducer)
	command.Registry().Add(commandplayernpc.RemoveCommandProducer)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	message2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := message2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	consumerplayernpc.InitConsumers(l)(cmf)(consumerGroupId)
	if err := consumerplayernpc.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
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
		AddRouteInitializer(chat.InitResource(GetServer())).
		Run()

	rt.Wait()
}
