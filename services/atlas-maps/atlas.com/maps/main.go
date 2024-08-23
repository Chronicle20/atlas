package main

import (
	"atlas-maps/character"
	"atlas-maps/logger"
	_map "atlas-maps/map"
	"atlas-maps/service"
	"atlas-maps/tasks"
	"atlas-maps/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-maps"
const consumerGroupId = "Map Service"

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
		prefix:  "/api/mas/",
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

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(character.StatusEventConsumer(l)(consumerGroupId))
	_, _ = cm.RegisterHandler(character.StatusEventLoginRegister(l))
	_, _ = cm.RegisterHandler(character.StatusEventLogoutRegister(l))
	_, _ = cm.RegisterHandler(character.StatusEventMapChangedRegister(l))

	go tasks.Register(tasks.NewRespawn(l, 10000))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), _map.InitResource(GetServer()))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
