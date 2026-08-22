package main

import (
	"atlas-player-npcs/character"
	"atlas-player-npcs/configuration"
	mapdata "atlas-player-npcs/data/map"
	npcdata "atlas-player-npcs/data/npc"
	"atlas-player-npcs/inventory"
	charconsumer "atlas-player-npcs/kafka/consumer/character"
	npcconsumer "atlas-player-npcs/kafka/consumer/playernpc"
	"atlas-player-npcs/playernpc"
	"atlas-player-npcs/ranking"
	"context"
	"os"

	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	service "github.com/Chronicle20/atlas/libs/atlas-service"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-player-npcs"

var consumerGroupId = consumergroup.Resolve("Player NPCs Service")

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
	rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
	l := rt.Logger()

	// Connect to the database
	db := database.Connect(l, database.SetMigrations(outboxlib.Migration, playernpc.Migration))

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

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	// playerNpcProcessorFor builds the real HTTP/DB-backed Processor for a
	// request-scoped (l, ctx) -- the same shape as rest.go's processorFor
	// (Task 16), but with the Kafka-backed EventEmitter (Task 17) instead
	// of REST's nil no-op. Shared by both Kafka consumers below.
	playerNpcProcessorFor := func(hl logrus.FieldLogger, hctx context.Context) playernpc.Processor {
		return playernpc.NewProcessor(hl, hctx, db,
			character.NewProcessor(hl, hctx),
			inventory.NewProcessor(hl, hctx),
			ranking.NewProcessor(hl, hctx),
			configuration.NewProcessor(hl, hctx),
			npcdata.NewProcessor(hl, hctx),
			mapdata.NewProcessor(hl, hctx),
			playernpc.NewEmitter(hl, hctx),
		)
	}

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	npcconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	charconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := npcconsumer.InitHandlers(l)(playerNpcProcessorFor)(npcconsumer.NewOutcomeEmitter(l, rt.Context()))(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	charDeps := charconsumer.Dependencies{
		Character: func(hl logrus.FieldLogger, hctx context.Context) character.Processor {
			return character.NewProcessor(hl, hctx)
		},
		Configuration: func(hl logrus.FieldLogger, hctx context.Context) configuration.Processor {
			return configuration.NewProcessor(hl, hctx)
		},
		PlayerNpc: playerNpcProcessorFor,
	}
	if err := charconsumer.InitHandlers(l)(charDeps)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(playernpc.InitializeRoutes(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
