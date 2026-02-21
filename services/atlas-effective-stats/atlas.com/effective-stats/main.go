package main

import (
	"atlas-effective-stats/character"
	asset2 "atlas-effective-stats/kafka/consumer/asset"
	buff2 "atlas-effective-stats/kafka/consumer/buff"
	character2 "atlas-effective-stats/kafka/consumer/character"
	session2 "atlas-effective-stats/kafka/consumer/session"
	"atlas-effective-stats/logger"
	"github.com/Chronicle20/atlas-service"
	"atlas-effective-stats/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-effective-stats"
const consumerGroupId = "Effective Stats Service"

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

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	// Initialize Kafka consumers
	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())

	// Session status events consumer (login/logout)
	session2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := session2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Buff status events consumer (apply/expire)
	buff2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := buff2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Asset status events consumer (equip/unequip)
	asset2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := asset2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Character status events consumer (stat changes)
	character2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

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
