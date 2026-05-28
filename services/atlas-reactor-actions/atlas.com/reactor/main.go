package main

import (
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"atlas-reactor-actions/logger"
	"atlas-reactor-actions/script"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"gorm.io/gorm"
)

const serviceName = "atlas-reactor-actions"

var consumerGroupId = consumergroup.Resolve("Reactor Actions Service")

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
	db := database.Connect(l, database.SetMigrations(
		script.MigrateTable,
		func(db *gorm.DB) error { return db.AutoMigrate(&seeder.SeedState{}) },
	))

	// Initialize Kafka consumers
	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	script.InitConsumers(l)(cmf)(consumerGroupId)

	// Initialize Kafka handlers
	if err := script.InitHandlers(l, db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// Initialize REST server
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(script.InitResource(GetServer())(db)).
		AddRouteInitializer(script.InitSeedResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
