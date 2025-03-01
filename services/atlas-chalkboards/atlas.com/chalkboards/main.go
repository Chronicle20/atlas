package main

import (
	"atlas-chalkboards/chalkboard"
	chalkboard2 "atlas-chalkboards/kafka/consumer/chalkboard"
	"atlas-chalkboards/kafka/consumer/character"
	"atlas-chalkboards/logger"
	"atlas-chalkboards/service"
	"atlas-chalkboards/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-chalkboards"
const consumerGroupId = "Chalkboard Service"

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
	character.InitConsumers(l)(cmf)(consumerGroupId)
	chalkboard2.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	chalkboard2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), chalkboard.InitResource(GetServer()))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
