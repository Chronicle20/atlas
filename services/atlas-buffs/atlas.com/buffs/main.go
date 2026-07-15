package main

import (
	"atlas-buffs/character"
	character2 "atlas-buffs/kafka/consumer/character"
	"atlas-buffs/tasks"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

const serviceName = "atlas-buffs"

var consumerGroupId = consumergroup.Resolve("Buff Service")

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
	character.InitRegistry(rc)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	character2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(tasks.NewExpiration(l, 10000))
	})
	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(tasks.NewPoisonTick(l, 1000))
	})

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(character.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
