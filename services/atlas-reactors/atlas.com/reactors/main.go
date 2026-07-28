package main

import (
	"atlas-reactors/reactor"
	"atlas-reactors/tasks"
	"context"
	"os"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	drop2 "atlas-reactors/kafka/consumer/drop"
	reactor2 "atlas-reactors/kafka/consumer/reactor"

	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-reactors"

var consumerGroupId = consumergroup.Resolve("Reactors Service")

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
	reactor.InitRegistry(rc)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	reactor2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := reactor2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	drop2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := drop2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(tasks.NewCooldownCleanup(l))
	})

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(reactor.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.TeardownFunc(reactor.NewProcessor(l, rt.Context()).Teardown())

	rt.Wait()
}
