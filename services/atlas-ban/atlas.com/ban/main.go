package main

import (
	"atlas-ban/ban"
	database "github.com/Chronicle20/atlas-database"
	"atlas-ban/history"
	account2 "atlas-ban/kafka/consumer/account"
	ban2 "atlas-ban/kafka/consumer/ban"
	"atlas-ban/logger"
	"atlas-ban/service"
	"atlas-ban/tasks"
	"atlas-ban/tracing"
	"os"
	"time"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-ban"
const consumerGroupId = "Ban Service"

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

	db := database.Connect(l, database.SetMigrations(ban.Migration, history.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	ban2.InitConsumers(l)(cmf)(consumerGroupId)
	ban2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	account2.InitConsumers(l)(cmf)(consumerGroupId)
	account2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(ban.InitResource(GetServer())(db)).
		AddRouteInitializer(history.InitResource(GetServer())(db)).
		Run()

	go tasks.Register(l, tdm.Context())(ban.NewExpiredBanCleanup(l, db, time.Minute*time.Duration(5)))
	go tasks.Register(l, tdm.Context())(history.NewHistoryPurge(l, db, time.Hour*time.Duration(24)))

	tdm.TeardownFunc(database.Teardown(l, db))
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()

	l.Infoln("Service shutdown.")
}
