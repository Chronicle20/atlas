package main

import (
	"atlas-npc/commodities"
	"atlas-npc/database"
	character2 "atlas-npc/kafka/consumer/character"
	shops2 "atlas-npc/kafka/consumer/shops"
	"atlas-npc/logger"
	"atlas-npc/seed"
	"atlas-npc/service"
	"atlas-npc/shops"
	"atlas-npc/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-npc-shops"
const consumerGroupId = "NPC Shops Service"

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

	db := database.Connect(l, database.SetMigrations(commodities.Migration, shops.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	character2.InitConsumers(l)(cmf)(consumerGroupId)
	shops2.InitConsumers(l)(cmf)(consumerGroupId)
	character2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	shops2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(shops.InitResource(GetServer())(db)).
		AddRouteInitializer(seed.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
