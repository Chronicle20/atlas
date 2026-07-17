package main

import (
	"atlas-rates/character"
	asset2 "atlas-rates/kafka/consumer/asset"
	buff2 "atlas-rates/kafka/consumer/buff"
	character2 "atlas-rates/kafka/consumer/character"
	rate2 "atlas-rates/kafka/consumer/rate"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-rates"

var consumerGroupId = consumergroup.Resolve("Rate Service")

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
	character.InitItemTracker(rc)
	character.InitInitializedRegistry(rc)

	// Initialize Kafka consumers
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())

	// Buff status events consumer
	buff2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := buff2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// World/Channel rate events consumer
	rate2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := rate2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Asset status events consumer (for item rate tracking)
	asset2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := asset2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Character status events consumer (for rate initialization on map enter)
	character2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Start REST server
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
