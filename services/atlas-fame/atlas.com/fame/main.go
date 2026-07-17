package main

import (
	"atlas-fame/fame"
	"atlas-fame/kafka/consumer/character"
	fame2 "atlas-fame/kafka/consumer/fame"
	"context"
	"os"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	service "github.com/Chronicle20/atlas/libs/atlas-service"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

const serviceName = "atlas-fame"

var consumerGroupId = consumergroup.Resolve("Fame Service")

func main() {
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	db := database.Connect(l, database.SetMigrations(fame.Migration, outboxlib.Migration))

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

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	fame2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := fame2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	character.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
