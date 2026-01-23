package main

import (
	reactor2 "atlas-reactors/kafka/consumer/reactor"
	"atlas-reactors/logger"
	"atlas-reactors/reactor"
	"atlas-reactors/service"
	"atlas-reactors/tasks"
	"atlas-reactors/tracing"

	"github.com/Chronicle20/atlas-kafka/consumer"
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

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	reactor2.InitConsumers(l)(cmf)(consumerGroupId)
	reactor2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	go tasks.Register(tasks.NewCooldownCleanup(l))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), reactor.InitResource(GetServer()))

	tdm.TeardownFunc(reactor.Teardown(l))
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
