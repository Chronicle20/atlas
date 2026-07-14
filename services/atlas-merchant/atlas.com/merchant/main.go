package main

import (
	"atlas-merchant/frederick"
	character "atlas-merchant/kafka/consumer/character"
	compartment2 "atlas-merchant/kafka/consumer/compartment"
	merchant2 "atlas-merchant/kafka/consumer/merchant"
	"atlas-merchant/listing"
	"atlas-merchant/logger"
	"atlas-merchant/message"
	"atlas-merchant/searchcount"
	"atlas-merchant/service"
	"atlas-merchant/shop"
	"atlas-merchant/tasks"
	"atlas-merchant/visitor"
	"context"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-merchant"

var consumerGroupId = consumergroup.Resolve("Merchant Service")

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

	rc := atlas.Connect(l)
	shop.InitRegistry(rc)
	visitor.InitRegistry(rc)

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(shop.Migration, listing.Migration, message.Migration, frederick.Migration, searchcount.Migration, outboxlib.Migration))

	// Boot the outbox drainer: publishes the transactional outbox to Kafka.
	// Leadership is gated by a postgres advisory lock — replicas are safe.
	publisher := outboxlib.NewTopicWriterPool()
	drainer := outboxlib.NewDrainer(l, db, publisher, outboxlib.WithDSN(database.DSN()))
	routine.Go(l, tdm.Context(), func(_ context.Context) {
		drainer.Run(tdm.Context())
	})
	tdm.TeardownFunc(func() {
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

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	merchant2.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	compartment2.InitConsumers(l)(cmf)(consumerGroupId)
	merchant2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	if err := character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register character status handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	compartment2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	// Start background tasks.
	tasks.Register(l, tdm.Context())(shop.NewExpirationTask(l, tdm.Context(), db, shop.DefaultExpirationInterval))
	tasks.Register(l, tdm.Context())(frederick.NewCleanupTask(l, tdm.Context(), db, frederick.DefaultCleanupInterval))
	tasks.Register(l, tdm.Context())(frederick.NewNotificationTask(l, tdm.Context(), db, frederick.DefaultNotificationInterval))

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(shop.InitializeRoutes(GetServer())(db)).
		AddRouteInitializer(frederick.InitializeRoutes(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
