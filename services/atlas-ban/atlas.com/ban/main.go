package main

import (
	"context"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	"atlas-ban/ban"
	"atlas-ban/history"
	account2 "atlas-ban/kafka/consumer/account"
	ban2 "atlas-ban/kafka/consumer/ban"
	"atlas-ban/logger"
	"atlas-ban/tasks"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-ban"

var consumerGroupId = consumergroup.Resolve("Ban Service")

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

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	ban2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := ban2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	account2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := account2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(ban.InitResource(GetServer())(db)).
		AddRouteInitializer(history.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		Run()

	routine.Go(l, tdm.Context(), func(_ context.Context) {
		tasks.Register(l, tdm.Context())(ban.NewExpiredBanCleanup(l, tdm.Context(), db, time.Minute*time.Duration(5)))
	})
	routine.Go(l, tdm.Context(), func(_ context.Context) {
		tasks.Register(l, tdm.Context())(history.NewHistoryPurge(l, tdm.Context(), db, time.Hour*time.Duration(24)))
	})

	tdm.TeardownFunc(database.Teardown(l, db))
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()

	l.Infoln("Service shutdown.")
}
