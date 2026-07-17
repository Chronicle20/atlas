package main

import (
	"atlas-monster-death/kafka/consumer/monster"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-monster-death"

var consumerGroupId = consumergroup.Resolve("Monster Death Service")

func main() {
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	monster.InitConsumers(l)(cmf)(consumerGroupId)
	if err := monster.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
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
