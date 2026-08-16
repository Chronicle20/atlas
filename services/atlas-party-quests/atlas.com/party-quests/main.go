package main

import (
	"atlas-party-quests/definition"
	"atlas-party-quests/instance"
	"context"
	"os"
	"time"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	characterConsumer "atlas-party-quests/kafka/consumer/character"
	monsterConsumer "atlas-party-quests/kafka/consumer/monster"
	pqConsumer "atlas-party-quests/kafka/consumer/party_quest"
	tenant2 "atlas-party-quests/tenant"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const serviceName = "atlas-party-quests"

var consumerGroupId = consumergroup.Resolve("Party Quest Service")

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

	db := database.Connect(l, database.SetMigrations(definition.MigrateTable, func(db *gorm.DB) error {
		return db.AutoMigrate(&seeder.SeedState{})
	}))

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	pqConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	characterConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	monsterConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := pqConsumer.InitHandlers(l, db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := characterConsumer.InitHandlers(l, db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := monsterConsumer.InitHandlers(l, db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	listTenants := func(ctx context.Context) ([]tenant.Model, error) {
		return tenant2.NewProcessor(l, ctx).GetAll()
	}

	// Start background ticker for PQ timers. Both the owned-environment set
	// and each environment's tenant set are resolved fresh on every tick via
	// service.ForEachOwnedEnvironment (FR-6.4/design C4) — the pre-existing
	// shape loaded tenants once before the ticker and closed over the slice;
	// a tenant provisioned after this pod started would never be picked up.
	routine.Go(l, rt.Context(), func(_ context.Context) {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-rt.Context().Done():
				return
			case <-ticker.C:
				service.ForEachOwnedEnvironment(l, rt.Context(), serviceName, listTenants,
					func(ctx context.Context) {
						ip := instance.NewProcessor(l, ctx, db)
						if err := ip.TickGlobalTimerAndEmit(); err != nil {
							l.WithError(err).Warn("Error ticking global timer.")
						}
						if err := ip.TickStageTimerAndEmit(); err != nil {
							l.WithError(err).Warn("Error ticking stage timer.")
						}
						if err := ip.TickBonusTimerAndEmit(); err != nil {
							l.WithError(err).Warn("Error ticking bonus timer.")
						}
						if err := ip.TickCompletionTimerAndEmit(); err != nil {
							l.WithError(err).Warn("Error ticking completion timer.")
						}
						if err := ip.TickRegistrationTimerAndEmit(); err != nil {
							l.WithError(err).Warn("Error ticking registration timer.")
						}
					})
			}
		}
	})

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(definition.InitResource(GetServer())(db)).
		AddRouteInitializer(definition.InitSeedResource(GetServer())(db)).
		AddRouteInitializer(instance.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.TeardownFunc(func() {
		l.Infoln("Graceful shutdown: handling active PQ instances.")
		service.ForEachOwnedEnvironment(l, rt.Context(), serviceName, listTenants,
			func(ctx context.Context) {
				_ = instance.NewProcessor(l, ctx, db).GracefulShutdownAndEmit()
			})
	})

	rt.Wait()
}
