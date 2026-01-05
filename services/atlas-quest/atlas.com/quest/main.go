package main

import (
	"atlas-quest/database"
	"atlas-quest/logger"
	"atlas-quest/quest"
	"atlas-quest/quest/progress"
	"atlas-quest/service"
	"atlas-quest/tracing"

	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-quest"
const consumerGroupId = "Quest Service"

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

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(quest.Migration, progress.Migration))

	// TODO: Add Kafka consumers for monster kills, item changes, etc.
	// cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	// monster.InitConsumers(l)(cmf)(consumerGroupId)
	// monster.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), quest.InitResource(GetServer())(db))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
