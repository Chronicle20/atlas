package main

import (
	"atlas-family/database"
	"atlas-family/family"
	family2 "atlas-family/kafka/consumer/family"
	"atlas-family/logger"
	"atlas-family/scheduler"
	"atlas-family/service"
	"atlas-family/tracing"
	"os"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-family"
const consumerGroupId = "Family Service"

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

	// Initialize database connection
	db := database.Connect(l, database.SetMigrations(family.Migration))
	if db == nil {
		l.Fatal("Failed to connect to database")
	}

	// Initialize and start Kafka consumers
	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	family2.InitConsumers(l)(cmf)(consumerGroupId)
	family2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	// Initialize and start reputation reset scheduler
	reputationResetJob := scheduler.NewReputationResetJob(l, db)
	if err := reputationResetJob.Start(tdm.Context()); err != nil {
		l.WithError(err).Fatal("Failed to start reputation reset job")
	}

	// Setup graceful shutdown for scheduler
	tdm.TeardownFunc(func() {
		reputationResetJob.Stop()
	})

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(family.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
