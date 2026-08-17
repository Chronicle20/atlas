package main

import (
	"atlas-saga-orchestrator/kafka/consumer/asset"
	"atlas-saga-orchestrator/kafka/consumer/buddylist"
	"atlas-saga-orchestrator/kafka/consumer/cashshop"
	"atlas-saga-orchestrator/kafka/consumer/character"
	"atlas-saga-orchestrator/kafka/consumer/compartment"
	"atlas-saga-orchestrator/kafka/consumer/consumable"
	"atlas-saga-orchestrator/kafka/consumer/guild"
	"atlas-saga-orchestrator/kafka/consumer/pet"
	"atlas-saga-orchestrator/kafka/consumer/quest"
	"atlas-saga-orchestrator/kafka/consumer/skill"
	"atlas-saga-orchestrator/kafka/consumer/storage"
	"atlas-saga-orchestrator/saga"
	"context"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	cashshopCompartment "atlas-saga-orchestrator/kafka/consumer/cashshop/compartment"

	inventoryConsumer "atlas-saga-orchestrator/kafka/consumer/inventory"
	mtsCustody "atlas-saga-orchestrator/kafka/consumer/mts/custody"
	noteConsumer "atlas-saga-orchestrator/kafka/consumer/note"
	npcconversationconsumer "atlas-saga-orchestrator/kafka/consumer/npcconversation"
	npcshopconsumer "atlas-saga-orchestrator/kafka/consumer/npcshop"
	tradeCustody "atlas-saga-orchestrator/kafka/consumer/trade/custody"

	saga2 "atlas-saga-orchestrator/kafka/consumer/saga"

	storageCompartment "atlas-saga-orchestrator/kafka/consumer/storage/compartment"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const serviceName = "atlas-saga-orchestrator"

// withSelfEnvironment attaches this pod's own environment identity
// (env.Self()) to ctx, with no tenant pairing. recoverSagas and
// reapTimedOutSagas reconstruct a per-row tenant context from Entity
// columns that carry no environment of their own (saga/entity.go has no
// environment column); without this, every recovered/reaped saga's REST
// calls and Kafka emits would resolve to the baseline URL/topic regardless
// of which environment originally drove the saga (FR-1.8, FR-3.1/FR-3.2).
// saga/ is outside env-domain-guard's permitted atlas-env import list, so
// this is threaded in as a plain function value rather than the package
// importing atlas-env itself (matches socket.WithSelfEnvironment,
// 99c0e598d).
func withSelfEnvironment(ctx context.Context) context.Context {
	return env.WithContext(ctx, env.Self())
}

var consumerGroupId = consumergroup.Resolve("Saga Orchestrator Service")

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

	// Initialize database connection
	db := database.Connect(l, database.SetMigrations(saga.Migration))
	l.Infoln("Database connected and migrated.")

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	// Initialize PostgreSQL-backed saga store
	store := saga.NewPostgresStore(db, l)
	saga.SetCache(store)
	l.Infoln("PostgreSQL saga store initialized.")

	// Configure saga timeout
	defaultTimeout := 5 * time.Minute
	if v, ok := os.LookupEnv("SAGA_DEFAULT_TIMEOUT"); ok {
		if parsed, err := time.ParseDuration(v); err == nil {
			defaultTimeout = parsed
		}
	}
	saga.SetDefaultTimeout(defaultTimeout)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	asset.InitConsumers(l)(cmf)(consumerGroupId)
	buddylist.InitConsumers(l)(cmf)(consumerGroupId)
	cashshop.InitConsumers(l)(cmf)(consumerGroupId)
	cashshopCompartment.InitConsumers(l)(cmf)(consumerGroupId)
	mtsCustody.InitConsumers(l)(cmf)(consumerGroupId)
	tradeCustody.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	compartment.InitConsumers(l)(cmf)(consumerGroupId)
	consumable.InitConsumers(l)(cmf)(consumerGroupId)
	guild.InitConsumers(l)(cmf)(consumerGroupId)
	inventoryConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	noteConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	pet.InitConsumers(l)(cmf)(consumerGroupId)
	quest.InitConsumers(l)(cmf)(consumerGroupId)
	saga2.InitConsumers(l)(cmf)(consumerGroupId)
	skill.InitConsumers(l)(cmf)(consumerGroupId)
	storage.InitConsumers(l)(cmf)(consumerGroupId)
	npcshopconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	npcconversationconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	storageCompartment.InitConsumers(l)(cmf)(consumerGroupId)
	if err := asset.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := buddylist.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := cashshop.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := tradeCustody.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatalf("Unable to register trade custody status handlers.")
	}
	if err := mtsCustody.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Failed to register MTS custody status handlers.")
	}
	if err := cashshopCompartment.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := character.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := compartment.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := consumable.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := guild.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := inventoryConsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := noteConsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := pet.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := quest.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := saga2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := skill.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := storage.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := npcshopconsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatalf("Unable to register npc shop status handlers.")
	}
	if err := npcconversationconsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatalf("Unable to register npc conversation status handlers.")
	}
	if err := storageCompartment.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// Arm the saga timeout backstop's timer-fire environment origination
	// (saga/timer.go). The timer re-wraps a fresh context.Background() when
	// it fires, so it has no request context to inherit ENVIRONMENT from --
	// withSelfEnvironment supplies this pod's own environment identity
	// instead (FR-1.8), matching the recipe's envContext DI pattern
	// (99c0e598d) since saga/ is outside env-domain-guard's permitted
	// atlas-env import list.
	saga.SagaTimers().SetEnvContext(withSelfEnvironment)

	// Recover active sagas from database
	recoverSagas(l, store, rt.TeardownManager())

	// Start the stale saga reaper
	startReaper(l, store, rt.TeardownManager())

	// Create the service with the router
	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(saga.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}

// recoverSagas loads all active sagas from the database and re-drives them.
// This is a persisted-work path (design §8.3): each saga row carries its own
// tenant (region/version included, unlike the mts listing table), so
// service.ForEachOwnedEnvironment reconstructs the environment from that
// tenant rather than from env.Self() — a recovered saga belonging to a
// tenant of an environment this deployment does not own is simply never
// visited.
func recoverSagas(l logrus.FieldLogger, store *saga.PostgresStore, tdm *service.Manager) {
	enabled := true
	if v, ok := os.LookupEnv("SAGA_RECOVERY_ENABLED"); ok {
		if parsed, err := strconv.ParseBool(v); err == nil {
			enabled = parsed
		}
	}

	if !enabled {
		l.Infoln("Saga recovery disabled via SAGA_RECOVERY_ENABLED=false")
		return
	}

	entities := store.GetAllActive(tdm.Context())
	if len(entities) == 0 {
		l.Infoln("No active sagas to recover.")
		return
	}

	l.Infof("Recovering %d active sagas from database.", len(entities))
	service.ForEachOwnedEnvironment(l, tdm.Context(), serviceName, sagaEntityTenants(entities),
		func(ctx context.Context) {
			tm := tenant.MustFromContext(ctx)
			processor := saga.NewProcessor(l, ctx)
			for _, e := range entities {
				if e.TenantId != tm.Id() {
					continue
				}

				l.Infof("Recovering saga [%s] type [%s] for tenant [%s]",
					e.TransactionId.String(), e.SagaType, e.TenantId.String())

				err := processor.Step(e.TransactionId)
				if err != nil {
					l.WithError(err).Errorf("Failed to recover saga [%s]", e.TransactionId.String())
				}
			}
		})
	l.Infoln("Saga recovery complete.")
}

// sagaEntityTenants returns a TenantLister that reconstructs the distinct
// tenants referenced by entities, deduplicated by tenant id. Each saga.Entity
// carries its own region/major/minor (unlike e.g. atlas-mts's listing rows),
// so the reconstructed tenant.Model is a real one, not a placeholder.
func sagaEntityTenants(entities []saga.Entity) service.TenantLister {
	return func(_ context.Context) ([]tenant.Model, error) {
		seen := make(map[uuid.UUID]bool, len(entities))
		ts := make([]tenant.Model, 0, len(entities))
		for _, e := range entities {
			if seen[e.TenantId] {
				continue
			}
			tm, err := tenant.Create(e.TenantId, e.TenantRegion, e.TenantMajor, e.TenantMinor)
			if err != nil {
				continue
			}
			seen[e.TenantId] = true
			ts = append(ts, tm)
		}
		return ts, nil
	}
}

// startReaper starts a background goroutine that compensates timed-out sagas
func startReaper(l logrus.FieldLogger, store *saga.PostgresStore, tdm *service.Manager) {
	interval := 30 * time.Second
	if v, ok := os.LookupEnv("SAGA_REAPER_INTERVAL"); ok {
		if parsed, err := time.ParseDuration(v); err == nil {
			interval = parsed
		}
	}

	tdm.WaitGroup().Add(1)
	routine.Go(l, tdm.Context(), func(_ context.Context) {
		defer tdm.WaitGroup().Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		l.Infof("Saga reaper started (interval=%s)", interval)

		for {
			select {
			case <-tdm.Context().Done():
				l.Infoln("Saga reaper shutting down.")
				return
			case <-ticker.C:
				reapTimedOutSagas(l, store, tdm)
			}
		}
	})
}

// reapTimedOutSagas is the same persisted-work path as recoverSagas: the
// environment is reconstructed per-row from the timed-out saga's own tenant
// via service.ForEachOwnedEnvironment, not from env.Self().
func reapTimedOutSagas(l logrus.FieldLogger, store *saga.PostgresStore, tdm *service.Manager) {
	entities := store.GetTimedOut(tdm.Context())
	if len(entities) == 0 {
		return
	}

	l.Infof("Reaping %d timed-out sagas.", len(entities))
	service.ForEachOwnedEnvironment(l, tdm.Context(), serviceName, sagaEntityTenants(entities),
		func(ctx context.Context) {
			tm := tenant.MustFromContext(ctx)
			processor := saga.NewProcessor(l, ctx)
			for _, e := range entities {
				if e.TenantId != tm.Id() {
					continue
				}

				l.Warnf("Saga [%s] type [%s] timed out, triggering compensation.",
					e.TransactionId.String(), e.SagaType)

				// Mark the earliest pending step as failed to trigger compensation
				err := processor.MarkEarliestPendingStep(e.TransactionId, saga.Failed)
				if err != nil {
					l.WithError(err).Errorf("Failed to mark timed-out saga [%s] step as failed", e.TransactionId.String())
					continue
				}

				err = processor.Step(e.TransactionId)
				if err != nil {
					l.WithError(err).Errorf("Failed to compensate timed-out saga [%s]", e.TransactionId.String())
				}
			}
		})
}
