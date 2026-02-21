package main

import (
	"atlas-npc-conversations/conversation"
	npcConversation "atlas-npc-conversations/conversation/npc"
	"atlas-npc-conversations/conversation/quest"
	database "github.com/Chronicle20/atlas-database"
	"atlas-npc-conversations/kafka/consumer/character"
	"atlas-npc-conversations/kafka/consumer/npc"
	questConsumer "atlas-npc-conversations/kafka/consumer/quest"
	"atlas-npc-conversations/kafka/consumer/saga"
	"atlas-npc-conversations/logger"
	"github.com/Chronicle20/atlas-service"
	"atlas-npc-conversations/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-npc-conversations"
const consumerGroupId = "NPC Conversation Service"

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
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	rc := atlas.Connect(l)
	conversation.InitRegistry(rc)

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(npcConversation.MigrateTable, quest.MigrateTable))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	character.InitConsumers(l)(cmf)(consumerGroupId)
	npc.InitConsumers(l)(cmf)(consumerGroupId)
	questConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	saga.InitConsumers(l)(cmf)(consumerGroupId)

	if err := character.InitHandlers(l, db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := npc.InitHandlers(l, db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := questConsumer.InitHandlers(l, db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := saga.InitHandlers(l, db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(npcConversation.InitResource(GetServer())(db)).
		AddRouteInitializer(quest.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
