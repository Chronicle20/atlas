package main

import (
	_map "atlas-monsters/kafka/consumer/map"
	"atlas-monsters/logger"
	"atlas-monsters/monster"
	"atlas-monsters/service"
	"atlas-monsters/tasks"
	"atlas-monsters/tracing"
	"atlas-monsters/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"time"
)

const serviceName = "atlas-monsters"
const consumerGroupId = "Monster Registry Service"

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

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	monster.InitConsumers(l)(cmf)(consumerGroupId)
	_map.InitConsumers(l)(cmf)(consumerGroupId)
	monster.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	_map.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), monster.InitResource(GetServer()), world.InitResource(GetServer()))

	tasks.Register(l, tdm.Context())(monster.NewRegistryAudit(l, time.Second*30))

	tdm.TeardownFunc(monster.Teardown(l))
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()

	l.Infoln("Service shutdown.")
}
