package main

import (
	"atlas-account/account"
	"atlas-account/database"
	account2 "atlas-account/kafka/consumer/account"
	"atlas-account/logger"
	"atlas-account/service"
	"atlas-account/tasks"
	"atlas-account/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"time"
)

const serviceName = "atlas-account"
const consumerGroupId = "Account Service"

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

	db := database.Connect(l, database.SetMigrations(account.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	account2.InitConsumers(l)(cmf)(consumerGroupId)
	account2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), account.InitResource(GetServer())(db))

	go tasks.Register(l, tdm.Context())(account.NewTransitionTimeout(l, db, time.Second*time.Duration(5)))

	tdm.TeardownFunc(account.Teardown(l, db))
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()

	l.Infoln("Service shutdown.")
}
