package main

import (
	minigameconsumer "atlas-mini-games/kafka/consumer/minigame"
	"atlas-mini-games/logger"
	"atlas-mini-games/record"
	"os"
	"sync/atomic"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
)

const serviceName = "atlas-mini-games"

var consumerGroupId = consumergroup.Resolve("Mini Game Service")

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

	// No Redis / registry for this service — the miniroom/game state lives in
	// an in-memory registry added by a later plan task, not Redis.
	db := database.Connect(l, database.SetMigrations(record.Migration))

	// Mini-game lifecycle command consumer (create/visit/leave/chat/expel).
	// record has no Kafka consumers — it's a pure REST + persistence domain
	// called directly by the game domain. Gameplay command handlers register
	// alongside these once Task 15 lands.
	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	minigameconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := minigameconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// Process-level shutting-down flag; flipped on SIGTERM teardown so
	// /readyz reports not-ready before the rest of shutdown.
	var shuttingDown atomic.Bool
	ready := func() bool { return !shuttingDown.Load() }
	tdm.TeardownFunc(func() {
		shuttingDown.Store(true)
		l.Info("Flipped /readyz to not-ready for graceful shutdown.")
	})

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(record.InitResource(GetServer())(db)).
		// AddRouteInitializer(game.InitResource(GetServer())) is added here
		// once the game domain lands (plan tasks 11+).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", ready)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
