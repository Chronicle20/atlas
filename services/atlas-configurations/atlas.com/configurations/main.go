package main

import (
	"atlas-configurations/logger"
	"atlas-configurations/outbox"
	"atlas-configurations/seeder"
	"atlas-configurations/services"
	"atlas-configurations/templates"
	"atlas-configurations/tenants"
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
)

const serviceName = "atlas-configurations"

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

	db := database.Connect(l, database.SetMigrations(templates.Migration, tenants.Migration, services.Migration, outboxlib.Migration))

	// Boot the outbox drainer: publishes the transactional outbox to Kafka.
	// Uses pq.Listener (via WithDSN) for sub-100ms wake-up on Enqueue, with
	// the poll interval as the fallback. Leadership is gated by a postgres
	// advisory lock — multiple atlas-configurations replicas can run safely;
	// only the lock holder publishes.
	publisher := outbox.NewTopicWriterPool()
	drainer := outboxlib.NewDrainer(l, db, publisher, outboxlib.WithDSN(database.DSN()))
	go drainer.Run(tdm.Context())
	tdm.TeardownFunc(func() {
		drainer.Stop()
		publisher.Close()
	})

	// Run seed import
	seedConfig := seeder.DefaultConfig()
	l.WithFields(map[string]interface{}{
		"seedPath":    seedConfig.SeedPath,
		"seedEnabled": seedConfig.Enabled,
	}).Info("Seed configuration loaded")
	s := seeder.NewSeeder(l, tdm.Context(), db, seedConfig)
	if err := s.Run(); err != nil {
		l.WithError(err).Error("Seed import failed")
	}

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(templates.InitResource(GetServer())(db)).
		AddRouteInitializer(tenants.InitResource(GetServer())(db)).
		AddRouteInitializer(services.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
