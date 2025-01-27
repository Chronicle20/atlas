package main

import (
	"atlas-guilds/database"
	"atlas-guilds/guild"
	"atlas-guilds/guild/character"
	"atlas-guilds/guild/member"
	"atlas-guilds/guild/title"
	character2 "atlas-guilds/kafka/consumer/character"
	guild2 "atlas-guilds/kafka/consumer/guild"
	"atlas-guilds/kafka/consumer/invite"
	thread2 "atlas-guilds/kafka/consumer/thread"
	"atlas-guilds/logger"
	"atlas-guilds/service"
	"atlas-guilds/tasks"
	"atlas-guilds/thread"
	"atlas-guilds/thread/reply"
	"atlas-guilds/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"time"
)

const serviceName = "atlas-guilds"
const consumerGroupId = "Guilds Service"

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

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(guild.Migration, title.Migration, member.Migration, character.Migration, thread.Migration, reply.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	guild2.InitConsumers(l)(cmf)(consumerGroupId)
	character2.InitConsumers(l)(cmf)(consumerGroupId)
	invite.InitConsumers(l)(cmf)(consumerGroupId)
	thread2.InitConsumers(l)(cmf)(consumerGroupId)

	guild2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	character2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	invite.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	thread2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), guild.InitResource(GetServer())(db), thread.InitResource(GetServer())(db))

	go tasks.Register(l, tdm.Context())(guild.NewTransitionTimeout(l, db, time.Second*time.Duration(35)))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
