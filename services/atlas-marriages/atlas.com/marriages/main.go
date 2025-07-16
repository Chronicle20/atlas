package main

import (
	"atlas-marriages/database"
	"atlas-marriages/kafka/consumer/marriage"
	"atlas-marriages/logger"
	marriageService "atlas-marriages/marriage"
	"atlas-marriages/scheduler"
	"atlas-marriages/service"
	"atlas-marriages/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-marriages"

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

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(marriageService.Migration))
	
	// Initialize proposal expiry scheduler
	proposalExpiryScheduler := scheduler.NewProposalExpiryScheduler(l, tdm.Context(), db)
	proposalExpiryScheduler.Start()
	
	// Register scheduler teardown
	tdm.TeardownFunc(func() {
		proposalExpiryScheduler.Stop()
	})
	
	// Initialize Kafka consumers
	consumerManager := consumer.GetManager()
	marriage.InitConsumers(l, tdm.Context(), db)(
		consumerManager.AddConsumer(l, tdm.Context(), tdm.WaitGroup()),
	)("marriage-service")

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
