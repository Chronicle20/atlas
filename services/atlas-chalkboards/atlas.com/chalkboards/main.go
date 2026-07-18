package main

import (
	"atlas-chalkboards/chalkboard"
	character2 "atlas-chalkboards/character"
	chalkboard2 "atlas-chalkboards/kafka/consumer/chalkboard"
	"atlas-chalkboards/kafka/consumer/character"
	"os"

	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-chalkboards"

var consumerGroupId = consumergroup.Resolve("Chalkboard Service")

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
	chalkboard.InitRegistry(rc)
	character2.InitRegistry(rc)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	character.InitConsumers(l)(cmf)(consumerGroupId)
	chalkboard2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := chalkboard2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// CreateRoute and run server
	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		AddRouteInitializer(chalkboard.InitResource(GetServer())).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
