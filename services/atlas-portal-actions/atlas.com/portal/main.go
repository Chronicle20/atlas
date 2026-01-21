package main

import (
	"os"

	"atlas-portal-actions/database"
	"atlas-portal-actions/logger"
	"atlas-portal-actions/script"
	"atlas-portal-actions/service"
	"atlas-portal-actions/tracing"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-portal-actions"
const consumerGroupId = "Portal Actions Service"

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

	// Initialize database connection
	db := database.Connect(l, database.SetMigrations(script.MigrateTable))

	// Initialize Kafka consumers
	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	script.InitConsumers(l)(cmf)(consumerGroupId)

	// Initialize Kafka handlers
	script.InitHandlers(l, db)(consumer.GetManager().RegisterHandler)

	// Initialize REST server
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(script.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
