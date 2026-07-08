package main

import (
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

	// Domain consumer registrations (game: later tasks) land here via
	// consumer.GetManager().AddConsumer(...) followed by each domain's
	// InitConsumers/InitHandlers, mirroring atlas-buddies/main.go. record has
	// no Kafka consumers — it's a pure REST + persistence domain called
	// directly by the game domain (plan tasks 11+). Nothing to register yet,
	// so only the debug endpoint is mounted below.

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
