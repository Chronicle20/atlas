package main

import (
	"atlas-mts/bid"
	"atlas-mts/holding"
	custodyConsumer "atlas-mts/kafka/consumer/custody"
	mtsConsumer "atlas-mts/kafka/consumer/mts"
	"atlas-mts/listing"
	"atlas-mts/task"
	"atlas-mts/testsupport"
	"atlas-mts/transaction"
	"atlas-mts/wallet"
	"atlas-mts/wish"
	"context"
	"os"
	"strconv"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-mts"

var consumerGroupId = consumergroup.Resolve("MTS Service")

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

	db := database.Connect(l, database.SetMigrations(
		listing.Migration,
		holding.Migration,
		bid.Migration,
		wish.Migration,
		transaction.Migration,
		outboxlib.Migration,
	))

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

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	custodyConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	mtsConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := custodyConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := mtsConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// DB-driven auction-expiration sweep: each tick moves expired active auctions
	// to their seller's holding (origin=expired) across every tenant.
	expirationTask := task.NewPeriodicTask(l, rt.Context(), db, getExpirationInterval())
	expirationTask.Start()
	rt.TeardownFunc(expirationTask.Stop)

	srv := server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(listing.InitResource(GetServer())(db)).
		AddRouteInitializer(holding.InitResource(GetServer())(db)).
		AddRouteInitializer(wish.InitResource(GetServer())(db)).
		AddRouteInitializer(transaction.InitResource(GetServer())(db)).
		AddRouteInitializer(wallet.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready))

	// E2E test routes (seed/expire/sweep/simulated purchase+bid) — env-gated,
	// never routed through ingress, never enabled in any overlay. Enable ad hoc:
	//   kubectl set env deployment/atlas-mts MTS_TEST_ROUTES_ENABLED=true
	// See docs/tasks/task-102-mts-marketplace/design-e2e-testing.md.
	if os.Getenv("MTS_TEST_ROUTES_ENABLED") == "true" {
		l.Warnln("MTS TEST ROUTES ENABLED — /api/test/* is live. This must never be set in production.")
		srv = srv.AddRouteInitializer(testsupport.InitResource(GetServer())(db))
	}

	srv.Run()

	rt.Wait()
}

// getExpirationInterval reads the sweep cadence from
// EXPIRATION_CHECK_INTERVAL_SECONDS (mirrors the asset-expiration ticker's env
// name), falling back to 60s when unset or invalid.
func getExpirationInterval() time.Duration {
	intervalStr := os.Getenv("EXPIRATION_CHECK_INTERVAL_SECONDS")
	if intervalStr == "" {
		return 60 * time.Second
	}
	seconds, err := strconv.Atoi(intervalStr)
	if err != nil || seconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(seconds) * time.Second
}
