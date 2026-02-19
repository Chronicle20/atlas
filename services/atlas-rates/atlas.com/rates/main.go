package main

import (
	"atlas-rates/character"
	asset2 "atlas-rates/kafka/consumer/asset"
	buff2 "atlas-rates/kafka/consumer/buff"
	character2 "atlas-rates/kafka/consumer/character"
	rate2 "atlas-rates/kafka/consumer/rate"
	"atlas-rates/logger"
	"atlas-rates/service"
	"atlas-rates/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-rates"
const consumerGroupId = "Rate Service"

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
	character.InitRegistry(rc)
	character.InitItemTracker(rc)
	character.InitInitializedRegistry(rc)

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	// Initialize Kafka consumers
	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())

	// Buff status events consumer
	buff2.InitConsumers(l)(cmf)(consumerGroupId)
	buff2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	// World/Channel rate events consumer
	rate2.InitConsumers(l)(cmf)(consumerGroupId)
	rate2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	// Asset status events consumer (for item rate tracking)
	asset2.InitConsumers(l)(cmf)(consumerGroupId)
	asset2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	// Character status events consumer (for rate initialization on map enter)
	character2.InitConsumers(l)(cmf)(consumerGroupId)
	character2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	// Start REST server
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(character.InitResource(GetServer())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
