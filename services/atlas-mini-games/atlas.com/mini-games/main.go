package main

import (
	"atlas-mini-games/game"
	characterconsumer "atlas-mini-games/kafka/consumer/character"
	minigameconsumer "atlas-mini-games/kafka/consumer/minigame"
	sessionconsumer "atlas-mini-games/kafka/consumer/session"
	"atlas-mini-games/record"
	"context"
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/Chronicle20/atlas/libs/atlas-service"
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
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	// Transient DB-connection errors surface as 503 + Retry-After (DOM-27,
	// task-168) instead of 500, so callers retry rather than treating a
	// pool blip as a hard failure.
	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	// No Redis for this service — the miniroom/game state lives in the
	// process-wide in-memory game.Registry, not Redis. The outbox migration
	// backs the transactional emission of the endGame record-carrying events.
	db := database.Connect(l, database.SetMigrations(record.Migration, outboxlib.Migration))

	// Boot the outbox drainer: publishes the transactional outbox to Kafka.
	// Leadership is gated by a postgres advisory lock — replicas are safe.
	publisher := outboxlib.NewTopicWriterPool()
	drainer := outboxlib.NewDrainer(l, db, publisher, outboxlib.WithDSN(database.DSN()))
	routine.Go(l, rt.Context(), func(_ context.Context) {
		drainer.Run(rt.Context())
	})
	rt.TeardownFunc(func() {
		drainer.Stop()
		publisher.Close()
	})

	// Mini-game lifecycle + gameplay command consumer (create/visit/leave/chat/
	// expel plus ready/start/move/flip/tie/retreat/skip/exit-after). record has
	// no Kafka consumers — it's a pure REST + persistence domain called directly
	// by the game domain.
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	minigameconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := minigameconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Teardown consumers: release a character's mini-game room membership on
	// session destroy (disconnect/kick) or map-leave/logout, same forfeit-then-
	// leave path as an explicit LEAVE command (game.Processor.TeardownCharacter).
	sessionconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := sessionconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	characterconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := characterconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(record.InitResource(GetServer())(db)).
		AddRouteInitializer(game.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
