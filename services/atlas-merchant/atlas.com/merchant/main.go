package main

import (
	"atlas-merchant/blacklist"
	"atlas-merchant/frederick"
	character "atlas-merchant/kafka/consumer/character"
	compartment2 "atlas-merchant/kafka/consumer/compartment"
	merchant2 "atlas-merchant/kafka/consumer/merchant"
	"atlas-merchant/listing"
	"atlas-merchant/message"
	"atlas-merchant/searchcount"
	"atlas-merchant/shop"
	"atlas-merchant/tasks"
	"atlas-merchant/visit"
	"atlas-merchant/visitor"
	"context"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/Chronicle20/atlas/libs/atlas-service"
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
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	rc := atlas.Connect(l)
	shop.InitRegistry(rc)
	visitor.InitRegistry(rc)

	db := database.Connect(l, database.SetMigrations(shop.Migration, listing.Migration, message.Migration, frederick.Migration, searchcount.Migration, blacklist.Migration, visit.Migration, outboxlib.Migration))

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
	merchant2.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	compartment2.InitConsumers(l)(cmf)(consumerGroupId)
	merchant2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	if err := character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register character status handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	compartment2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	// Start background tasks.
	tasks.Register(l, rt.Context())(shop.NewExpirationTask(l, rt.Context(), db, shop.DefaultExpirationInterval))
	tasks.Register(l, rt.Context())(frederick.NewCleanupTask(l, rt.Context(), db, frederick.DefaultCleanupInterval))
	tasks.Register(l, rt.Context())(frederick.NewNotificationTask(l, rt.Context(), db, frederick.DefaultNotificationInterval))

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(shop.InitializeRoutes(GetServer())(db)).
		AddRouteInitializer(frederick.InitializeRoutes(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
