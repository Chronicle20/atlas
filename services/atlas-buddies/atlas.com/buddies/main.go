package main

import (
	"atlas-buddies/buddy"
	"atlas-buddies/database"
	"atlas-buddies/kafka/consumer/cashshop"
	"atlas-buddies/kafka/consumer/character"
	invite2 "atlas-buddies/kafka/consumer/invite"
	list2 "atlas-buddies/kafka/consumer/list"
	"atlas-buddies/list"
	"atlas-buddies/logger"
	"atlas-buddies/service"
	"atlas-buddies/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-buddies"
const consumerGroupId = "Buddy Service"

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

	db := database.Connect(l, database.SetMigrations(list.Migration, buddy.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	character.InitConsumers(l)(cmf)(consumerGroupId)
	list2.InitConsumers(l)(cmf)(consumerGroupId)
	invite2.InitConsumers(l)(cmf)(consumerGroupId)
	cashshop.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	list2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	invite2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	cashshop.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), list.InitResource(GetServer())(db))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
