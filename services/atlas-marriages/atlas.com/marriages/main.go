package main

import (
	"atlas-marriages/kafka/consumer/character"
	"atlas-marriages/kafka/consumer/marriage"
	marriageService "atlas-marriages/marriage"
	"atlas-marriages/scheduler"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-marriages"

var consumerGroupId = consumergroup.Resolve("Marriage Service")

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

	db := database.Connect(l, database.SetMigrations(marriageService.Migration))

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	// Initialize proposal expiry scheduler
	proposalExpiryScheduler := scheduler.NewProposalExpiryScheduler(l, rt.Context(), db)
	proposalExpiryScheduler.Start()

	// Initialize ceremony timeout scheduler
	ceremonyTimeoutScheduler := scheduler.NewCeremonyTimeoutScheduler(l, rt.Context(), db)
	ceremonyTimeoutScheduler.Start()

	// Register scheduler teardowns
	rt.TeardownFunc(func() {
		proposalExpiryScheduler.Stop()
		ceremonyTimeoutScheduler.Stop()
	})

	// Initialize Kafka consumers
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	marriage.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	if err := marriage.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		AddRouteInitializer(marriageService.InitializeRoutes(db)(GetServer())).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
