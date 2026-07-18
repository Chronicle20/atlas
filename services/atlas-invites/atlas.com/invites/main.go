package main

import (
	"atlas-invites/character"
	"atlas-invites/invite"
	"atlas-invites/tasks"
	"context"
	"os"
	"time"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	character2 "atlas-invites/kafka/consumer/character"
	invite2 "atlas-invites/kafka/consumer/invite"

	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-invites"

var consumerGroupId = consumergroup.Resolve("Invitation Service")

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
	invite.InitRegistry(rc)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	invite2.InitConsumers(l)(cmf)(consumerGroupId)
	character2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := invite2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := character2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// Create the service with the router
	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(character.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(invite.NewInviteTimeout(l, time.Second*time.Duration(5)))
	})

	rt.Wait()
}
