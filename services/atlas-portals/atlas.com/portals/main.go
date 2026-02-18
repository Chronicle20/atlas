package main

import (
	"atlas-portals/blocked"
	"atlas-portals/kafka/consumer/character"
	"atlas-portals/logger"
	"atlas-portals/portal"
	"atlas-portals/service"
	"atlas-portals/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-portals"
const consumerGroupId = "Portal Service"

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

	rc := atlas.Connect(l)
	blocked.InitRegistry(rc)

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	portal.InitConsumers(l)(cmf)(consumerGroupId)
	portal.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	// Create the REST server
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(blocked.InitResource(GetServer())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
