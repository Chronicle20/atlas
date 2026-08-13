package main

import (
	"atlas-dragons/dragon"
	"atlas-dragons/world"
	"os"

	characterevt "atlas-dragons/kafka/consumer/character"
	dragoncmd "atlas-dragons/kafka/consumer/dragon"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-dragons"

var consumerGroupId = consumergroup.Resolve("Dragon Registry Service")

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
	dragon.InitRegistry(rc)

	// Kafka consumers: the COMMAND_TOPIC_DRAGON command consumer (CREATE / DESTROY
	// / MOVE) and the character-status lifecycle cascade (login / logout /
	// map-change / channel-change / job-change destroy).
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	dragoncmd.InitConsumers(l)(cmf)(consumerGroupId)
	characterevt.InitConsumers(l)(cmf)(consumerGroupId)
	if err := dragoncmd.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register dragon command handlers.")
	}
	if err := characterevt.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register character status handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		AddRouteInitializer(dragon.InitResource(GetServer())).
		AddRouteInitializer(world.InitResource(GetServer())).
		Run()

	rt.Wait()
}
