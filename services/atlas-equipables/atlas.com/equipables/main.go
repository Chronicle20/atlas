package main

import (
	"atlas-equipables/database"
	"atlas-equipables/equipable"
	"atlas-equipables/kafka/consumer/asset"
	equipable2 "atlas-equipables/kafka/consumer/equipable"
	"atlas-equipables/logger"
	"atlas-equipables/service"
	"atlas-equipables/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"os"
)

const serviceName = "atlas-equipables"
const consumerGroupId = "Equipable Service"

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

	db := database.Connect(l, database.SetMigrations(equipable.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	equipable2.InitConsumers(l)(cmf)(consumerGroupId)
	asset.InitConsumers(l)(cmf)(consumerGroupId)
	equipable2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	asset.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(equipable.InitResource(GetServer(), db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
