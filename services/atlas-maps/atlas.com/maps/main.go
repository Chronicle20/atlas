package main

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	characterClient "atlas-maps/character"
	"atlas-maps/character/location"
	"atlas-maps/kafka/consumer/cashshop"
	"atlas-maps/kafka/consumer/character"
	mapConsumer "atlas-maps/kafka/consumer/map"
	mistConsumer "atlas-maps/kafka/consumer/mist"
	"atlas-maps/kafka/consumer/monster"
	sessionConsumer "atlas-maps/kafka/consumer/session"
	"atlas-maps/logger"
	_map "atlas-maps/map"
	spawnMonster "atlas-maps/map/monster"
	"atlas-maps/map/weather"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"atlas-maps/tasks"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"atlas-maps/visit"
	"context"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
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

	rc := atlas.Connect(l)
	spawnMonster.InitRegistry(rc)

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
	mistConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	sessionConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character.InitHandlers(l, db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := cashshop.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := monster.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := mapConsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := mistConsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register mist kafka handlers.")
	}
	if err := sessionConsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register session-status kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// posLookup resolves a character's world coordinates via the
	// atlas-character REST client. The closure is recreated per-call so
	// each lookup runs against the caller's tenant-scoped context.
	posLookup := func(ctx context.Context, characterId uint32) (int16, int16, error) {
		return characterClient.NewProcessor(l, ctx).Position(characterId)
	}

	go tasks.Register(tasks.NewRespawn(l, 10000))
	go tasks.Register(tasks.NewWeather(l, time.Second))
	go tasks.Register(tasks.NewMistTick(l, 1000, posLookup))

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(_map.InitResource(GetServer())).
		AddRouteInitializer(weather.InitResource(GetServer())).
		AddRouteInitializer(visit.InitResource(GetServer())(db)).
		AddRouteInitializer(location.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
