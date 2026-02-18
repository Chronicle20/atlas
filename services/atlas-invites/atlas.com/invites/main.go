package main

import (
	"atlas-invites/character"
	"atlas-invites/invite"
	character2 "atlas-invites/kafka/consumer/character"
	invite2 "atlas-invites/kafka/consumer/invite"
	"atlas-invites/logger"
	"atlas-invites/service"
	"atlas-invites/tasks"
	"atlas-invites/tracing"
	"os"
	"time"

	"github.com/Chronicle20/atlas-kafka/consumer"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-rest/server"
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

	rc := atlas.Connect(l)
	invite.InitRegistry(rc)

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	invite2.InitConsumers(l)(cmf)(consumerGroupId)
	character2.InitConsumers(l)(cmf)(consumerGroupId)
	invite2.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	character2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	// Create the service with the router
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(character.InitResource(GetServer())).
		Run()

	go tasks.Register(l, tdm.Context())(invite.NewInviteTimeout(l, time.Second*time.Duration(5)))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
