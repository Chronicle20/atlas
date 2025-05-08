package main

import (
	"atlas-data/cash"
	"atlas-data/commodity"
	"atlas-data/consumable"
	"atlas-data/data"
	"atlas-data/database"
	"atlas-data/document"
	"atlas-data/equipment"
	"atlas-data/etc"
	data2 "atlas-data/kafka/consumer/data"
	"atlas-data/logger"
	_map "atlas-data/map"
	"atlas-data/monster"
	"atlas-data/pet"
	"atlas-data/reactor"
	"atlas-data/service"
	"atlas-data/skill"
	"atlas-data/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"os"
)

const serviceName = "atlas-data"
const consumerGroupId = "Data Service"

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

	db := database.Connect(l, database.SetMigrations(document.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	data2.InitConsumers(l)(cmf)(consumerGroupId)
	data2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(data.InitResource(db)(GetServer())).
		AddRouteInitializer(_map.InitResource(db)(GetServer())).
		AddRouteInitializer(monster.InitResource(db)(GetServer())).
		AddRouteInitializer(equipment.InitResource(db)(GetServer())).
		AddRouteInitializer(reactor.InitResource(db)(GetServer())).
		AddRouteInitializer(skill.InitResource(db)(GetServer())).
		AddRouteInitializer(pet.InitResource(db)(GetServer())).
		AddRouteInitializer(consumable.InitResource(db)(GetServer())).
		AddRouteInitializer(cash.InitResource(db)(GetServer())).
		AddRouteInitializer(commodity.InitResource(db)(GetServer())).
		AddRouteInitializer(etc.InitResource(db)(GetServer())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
