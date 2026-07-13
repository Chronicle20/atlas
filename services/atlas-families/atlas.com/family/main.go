package main

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"atlas-family/family"
	"atlas-family/kafka/consumer/character"
	family2 "atlas-family/kafka/consumer/family"
	"atlas-family/scheduler"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-family"

var consumerGroupId = consumergroup.Resolve("Family Service")

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
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	// Initialize database connection
	db := database.Connect(l, database.SetMigrations(family.Migration))
	if db == nil {
		l.Fatal("Failed to connect to database")
	}

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	// Initialize and start Kafka consumers
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	family2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := family2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	character.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// Initialize and start reputation reset scheduler
	reputationResetJob := scheduler.NewReputationResetJob(l, db)
	if err := reputationResetJob.Start(rt.Context()); err != nil {
		l.WithError(err).Fatal("Failed to start reputation reset job")
	}

	// Setup graceful shutdown for scheduler
	rt.TeardownFunc(func() {
		reputationResetJob.Stop()
	})

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(family.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
