package main

import (
	"atlas-invites/character"
	"atlas-invites/invite"
	"atlas-invites/logger"
	"atlas-invites/service"
	"atlas-invites/tasks"
	"atlas-invites/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"time"
)

const serviceName = "atlas-invites"
const consumerGroupId = "Invitation Service"

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
	invite.InitConsumers(l)(cmf)(consumerGroupId)
	invite.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), character.InitResource(GetServer()))

	go tasks.Register(l, tdm.Context())(invite.NewInviteTimeout(l, time.Second*time.Duration(5)))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
