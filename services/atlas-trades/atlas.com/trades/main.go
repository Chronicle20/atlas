package main

import (
	inviteconsumer "atlas-trades/kafka/consumer/invite"
	tradeconsumer "atlas-trades/kafka/consumer/trade"
	"atlas-trades/ledger"
	"atlas-trades/trade"
	"context"
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-trades"

// consumerGroupId is the Kafka consumer group this service registers under. It
// is also the literal that deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
// reads out of this file to derive KAFKA_CONSUMER_GROUP for ephemeral envs, and
// that tools/service-registration-guard.sh mirrors to decide whether this
// service needs a generated PR consumer-group document. Deleting or renaming it
// would silently drop both.
var consumerGroupId = consumergroup.Resolve("Trade Service")

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

	// Live room state is the process-local in-memory trade.Registry, not the
	// DB — atlas-trades runs replicas: 1 for that reason (design §9). The DB
	// backs only the completed-trade ledger and the transactional outbox.
	db := database.Connect(l, database.SetMigrations(ledger.Migration, outboxlib.Migration))

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

	// Trade room lifecycle commands from atlas-channel (create/invite/decline/
	// enter), plus the answer half of the invites we issue: an accepted TRADE
	// invite seats the visitor, a rejected or expired one tears the room down.
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	tradeconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := tradeconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	inviteconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := inviteconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(trade.InitResource(GetServer())(db)).
		AddRouteInitializer(ledger.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
