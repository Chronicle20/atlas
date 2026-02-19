package main

import (
	database "github.com/Chronicle20/atlas-database"
	"atlas-maps/kafka/consumer/cashshop"
	"atlas-maps/kafka/consumer/character"
	mapConsumer "atlas-maps/kafka/consumer/map"
	"atlas-maps/kafka/consumer/monster"
	"atlas-maps/logger"
	_map "atlas-maps/map"
	"atlas-maps/map/weather"
	"atlas-maps/service"
	"atlas-maps/tasks"
	"atlas-maps/tracing"
	"atlas-maps/visit"
	"os"
	"time"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-maps"
const consumerGroupId = "Map Service"

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

	db := database.Connect(l, database.SetMigrations(visit.MigrateTable))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	character.InitConsumers(l)(cmf)(consumerGroupId)
	cashshop.InitConsumers(l)(cmf)(consumerGroupId)
	monster.InitConsumers(l)(cmf)(consumerGroupId)
	mapConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitHandlers(l, db)(consumer.GetManager().RegisterHandler)
	cashshop.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	monster.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	mapConsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	go tasks.Register(tasks.NewRespawn(l, 10000))
	go tasks.Register(tasks.NewWeather(l, time.Second))

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(_map.InitResource(GetServer())).
		AddRouteInitializer(weather.InitResource(GetServer())).
		AddRouteInitializer(visit.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
