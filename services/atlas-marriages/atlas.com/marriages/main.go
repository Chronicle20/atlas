package main

import (
	database "github.com/Chronicle20/atlas-database"
	"atlas-marriages/kafka/consumer/character"
	"atlas-marriages/kafka/consumer/marriage"
	"atlas-marriages/logger"
	marriageService "atlas-marriages/marriage"
	"atlas-marriages/scheduler"
	"github.com/Chronicle20/atlas-service"
	"atlas-marriages/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-marriages"
const consumerGroupId = "Marriage Service"

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

	db := database.Connect(l, database.SetMigrations(marriageService.Migration))

	// Initialize proposal expiry scheduler
	proposalExpiryScheduler := scheduler.NewProposalExpiryScheduler(l, tdm.Context(), db)
	proposalExpiryScheduler.Start()

	// Initialize ceremony timeout scheduler
	ceremonyTimeoutScheduler := scheduler.NewCeremonyTimeoutScheduler(l, tdm.Context(), db)
	ceremonyTimeoutScheduler.Start()

	// Register scheduler teardowns
	tdm.TeardownFunc(func() {
		proposalExpiryScheduler.Stop()
		ceremonyTimeoutScheduler.Stop()
	})

	// Initialize Kafka consumers
	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	marriage.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	if err := marriage.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		AddRouteInitializer(marriageService.InitializeRoutes(db)(GetServer())).
		SetPort(os.Getenv("REST_PORT")).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
