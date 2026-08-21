package main

import (
	"atlas-configurations/environmentcol"
	"atlas-configurations/environments"
	"atlas-configurations/seeder"
	"atlas-configurations/services"
	"atlas-configurations/servicesuniq"
	"atlas-configurations/templates"
	"atlas-configurations/tenants"
	"context"
	"os"
	"path/filepath"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
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
	rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
	l := rt.Logger()

	db := database.Connect(l, database.SetMigrations(
		templates.Migration, tenants.Migration, services.Migration, outboxlib.Migration,
		environments.Migration,
		environmentcol.Migration, // backfills the columns the three above create
		servicesuniq.Migration,   // must run last: it depends on environmentcol's backfilled column
	))

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	// Boot the outbox drainer: publishes the transactional outbox to Kafka.
	// Uses pq.Listener (via WithDSN) for sub-100ms wake-up on Enqueue, with
	// the poll interval as the fallback. Leadership is gated by a postgres
	// advisory lock — multiple atlas-configurations replicas can run safely;
	// only the lock holder publishes.
	publisher := outboxlib.NewTopicWriterPool()
	drainer := outboxlib.NewDrainer(l, db, publisher, outboxlib.WithDSN(database.DSN()))
	routine.Go(l, rt.Context(), func(_ context.Context) {
		drainer.Run(rt.Context())
	})
	rt.TeardownFunc(func() {
		drainer.Stop()
		publisher.Close()
	})

	environments.StartHeartbeat(l, rt.Context(), environments.NewProcessor(l, rt.Context(), db))

	// Run seed import
	seedConfig := seeder.DefaultConfig()
	l.WithFields(map[string]interface{}{
		"seedPath":    seedConfig.SeedPath,
		"seedEnabled": seedConfig.Enabled,
	}).Info("Seed configuration loaded")

	// The shipped-template catalog is loaded UNCONDITIONALLY, before the
	// seeder and before route registration. SEED_ENABLED gates whether
	// templates are imported, not whether the service knows what ships - an
	// operator who has disabled seeding still needs drift detection and the
	// reset button.
	// InitShippedCatalog already logs the outcome itself (directory + entry
	// count on success, or a WARN when the catalog is empty) - a second log
	// line here would either duplicate it or, worse, contradict the WARN with
	// a trailing INFO "loaded". See templates/shipped.go.
	catalog := templates.InitShippedCatalog(l, filepath.Join(seedConfig.SeedPath, "templates"))

	s := seeder.NewSeeder(l, rt.Context(), db, seedConfig, catalog)
	if err := s.Run(); err != nil {
		l.WithError(err).Error("Seed import failed")
	}

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(templates.InitResource(GetServer())(db)).
		AddRouteInitializer(tenants.InitResource(GetServer())(db)).
		AddRouteInitializer(services.InitResource(GetServer())(db)).
		AddRouteInitializer(environments.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
