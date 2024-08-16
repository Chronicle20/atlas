package main

import (
	"atlas-character-factory/character"
	"atlas-character-factory/factory"
	"atlas-character-factory/logger"
	"atlas-character-factory/service"
	"atlas-character-factory/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-character-factory"
const consumerGroupId = "Character Factory Service"

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
		prefix:  "/api/cfs/",
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

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(character.CreatedConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(character.ItemGainedConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(character.EquipChangedConsumer(l)(consumerGroupId))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), factory.InitResource(GetServer()))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
