package main

import (
	drop2 "atlas-reactors/kafka/consumer/drop"
	reactor2 "atlas-reactors/kafka/consumer/reactor"
	"atlas-reactors/logger"
	"atlas-reactors/reactor"
	"github.com/Chronicle20/atlas-service"
	"atlas-reactors/tasks"
	"atlas-reactors/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-reactors"
const consumerGroupId = "Reactors Service"

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
	reactor.InitRegistry(rc)

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	reactor2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := reactor2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	drop2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := drop2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	go tasks.Register(tasks.NewCooldownCleanup(l))

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(reactor.InitResource(GetServer())).
		Run()

	tdm.TeardownFunc(reactor.Teardown(l))
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
