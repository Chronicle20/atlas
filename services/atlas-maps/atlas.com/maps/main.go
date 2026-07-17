package main

import (
	characterClient "atlas-maps/character"
	"atlas-maps/character/location"
	"atlas-maps/character/warp"
	"atlas-maps/kafka/consumer/cashshop"
	"atlas-maps/kafka/consumer/character"
	data2 "atlas-maps/kafka/consumer/data"
	mapConsumer "atlas-maps/kafka/consumer/map"
	mistConsumer "atlas-maps/kafka/consumer/mist"
	"atlas-maps/kafka/consumer/monster"
	sessionConsumer "atlas-maps/kafka/consumer/session"
	_map "atlas-maps/map"
	spawnMonster "atlas-maps/map/monster"
	"atlas-maps/map/weather"
	"atlas-maps/tasks"
	"atlas-maps/visit"
	"context"
	"os"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-service"

	database "github.com/Chronicle20/atlas/libs/atlas-database"

	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

const serviceName = "atlas-maps"

var (
	consumerGroupId           = consumergroup.Resolve("Map Service")
	dataEventsConsumerGroupId = consumergroup.Resolve("Map Spawn Registry Invalidator")
)

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
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	rc := atlas.Connect(l)
	spawnMonster.InitRegistry(rc)

	db := database.Connect(l, database.SetMigrations(visit.MigrateTable, location.Migration))

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	character.InitConsumers(l)(cmf)(consumerGroupId)
	cashshop.InitConsumers(l)(cmf)(consumerGroupId)
	monster.InitConsumers(l)(cmf)(consumerGroupId)
	mapConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	mistConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	sessionConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	data2.InitConsumers(l)(cmf)(dataEventsConsumerGroupId)
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
	if err := data2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register data-events kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// posLookup resolves a character's world coordinates via the
	// atlas-character REST client. The closure is recreated per-call so
	// each lookup runs against the caller's tenant-scoped context.
	posLookup := func(ctx context.Context, characterId uint32) (int16, int16, error) {
		return characterClient.NewProcessor(l, ctx).Position(characterId)
	}

	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(tasks.NewRespawn(l, 10000))
	})
	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(tasks.NewWeather(l, time.Second))
	})
	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(tasks.NewMistTick(l, 1000, posLookup))
	})

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(_map.InitResource(GetServer())).
		AddRouteInitializer(weather.InitResource(GetServer())).
		AddRouteInitializer(visit.InitResource(GetServer())(db)).
		AddRouteInitializer(location.InitResource(GetServer())(db, func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) location.WarpProcessor {
			return warp.NewProcessor(l, ctx, db)
		})).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
