package main

import (
	"atlas-mts/bid"
	"atlas-mts/holding"
	custodyConsumer "atlas-mts/kafka/consumer/custody"
	mtsConsumer "atlas-mts/kafka/consumer/mts"
	"atlas-mts/listing"
	"atlas-mts/logger"
	"atlas-mts/task"
	"atlas-mts/testsupport"
	"atlas-mts/transaction"
	"atlas-mts/wallet"
	"atlas-mts/wish"
	"os"
	"strconv"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
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
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(
		listing.Migration,
		holding.Migration,
		bid.Migration,
		wish.Migration,
		transaction.Migration,
	))

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	custodyConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	mtsConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := custodyConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := mtsConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// DB-driven auction-expiration sweep: each tick moves expired active auctions
	// to their seller's holding (origin=expired) across every tenant.
	expirationTask := task.NewPeriodicTask(l, tdm.Context(), db, getExpirationInterval())
	expirationTask.Start()
	tdm.TeardownFunc(expirationTask.Stop)

	srv := server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(listing.InitResource(GetServer())(db)).
		AddRouteInitializer(holding.InitResource(GetServer())(db)).
		AddRouteInitializer(wish.InitResource(GetServer())(db)).
		AddRouteInitializer(transaction.InitResource(GetServer())(db)).
		AddRouteInitializer(wallet.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler()))

	// E2E test routes (seed/expire/sweep/simulated purchase+bid) — env-gated,
	// never routed through ingress, never enabled in any overlay. Enable ad hoc:
	//   kubectl set env deployment/atlas-mts MTS_TEST_ROUTES_ENABLED=true
	// See docs/tasks/task-102-mts-marketplace/design-e2e-testing.md.
	if os.Getenv("MTS_TEST_ROUTES_ENABLED") == "true" {
		l.Warnln("MTS TEST ROUTES ENABLED — /api/test/* is live. This must never be set in production.")
		srv = srv.AddRouteInitializer(testsupport.InitResource(GetServer())(db))
	}

	srv.Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
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
