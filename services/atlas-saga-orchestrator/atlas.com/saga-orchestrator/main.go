package main

import (
	database "github.com/Chronicle20/atlas-database"
	"atlas-saga-orchestrator/kafka/consumer/asset"
	"atlas-saga-orchestrator/kafka/consumer/buddylist"
	"atlas-saga-orchestrator/kafka/consumer/cashshop"
	cashshopCompartment "atlas-saga-orchestrator/kafka/consumer/cashshop/compartment"
	"atlas-saga-orchestrator/kafka/consumer/character"
	"atlas-saga-orchestrator/kafka/consumer/compartment"
	"atlas-saga-orchestrator/kafka/consumer/consumable"
	"atlas-saga-orchestrator/kafka/consumer/guild"
	"atlas-saga-orchestrator/kafka/consumer/pet"
	"atlas-saga-orchestrator/kafka/consumer/quest"
	saga2 "atlas-saga-orchestrator/kafka/consumer/saga"
	"atlas-saga-orchestrator/kafka/consumer/skill"
	"atlas-saga-orchestrator/kafka/consumer/storage"
	storageCompartment "atlas-saga-orchestrator/kafka/consumer/storage/compartment"
	"atlas-saga-orchestrator/logger"
	"atlas-saga-orchestrator/saga"
	"github.com/Chronicle20/atlas-service"
	"atlas-saga-orchestrator/tracing"
	"os"
	"strconv"
	"time"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/sirupsen/logrus"
)

const serviceName = "atlas-saga-orchestrator"
const consumerGroupId = "Saga Orchestrator Service"

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
	db := database.Connect(l, database.SetMigrations(saga.Migration))
	l.Infoln("Database connected and migrated.")

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

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	asset.InitConsumers(l)(cmf)(consumerGroupId)
	buddylist.InitConsumers(l)(cmf)(consumerGroupId)
	cashshop.InitConsumers(l)(cmf)(consumerGroupId)
	cashshopCompartment.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	compartment.InitConsumers(l)(cmf)(consumerGroupId)
	consumable.InitConsumers(l)(cmf)(consumerGroupId)
	guild.InitConsumers(l)(cmf)(consumerGroupId)
	pet.InitConsumers(l)(cmf)(consumerGroupId)
	quest.InitConsumers(l)(cmf)(consumerGroupId)
	saga2.InitConsumers(l)(cmf)(consumerGroupId)
	skill.InitConsumers(l)(cmf)(consumerGroupId)
	storage.InitConsumers(l)(cmf)(consumerGroupId)
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
	if err := storageCompartment.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// Recover active sagas from database
	recoverSagas(l, store, tdm)

	// Start the stale saga reaper
	startReaper(l, store, tdm)

	// Create the service with the router
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(saga.InitResource(GetServer())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}

// recoverSagas loads all active sagas from the database and re-drives them
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
	for _, e := range entities {
		t, _ := tenant.Create(e.TenantId, e.TenantRegion, e.TenantMajor, e.TenantMinor)
		ctx := tenant.WithContext(tdm.Context(), t)
		processor := saga.NewProcessor(l, ctx)

		l.Infof("Recovering saga [%s] type [%s] for tenant [%s]",
			e.TransactionId.String(), e.SagaType, e.TenantId.String())

		err := processor.Step(e.TransactionId)
		if err != nil {
			l.WithError(err).Errorf("Failed to recover saga [%s]", e.TransactionId.String())
		}
	}
	l.Infoln("Saga recovery complete.")
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
	go func() {
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
	}()
}

func reapTimedOutSagas(l logrus.FieldLogger, store *saga.PostgresStore, tdm *service.Manager) {
	entities := store.GetTimedOut(tdm.Context())
	if len(entities) == 0 {
		return
	}

	l.Infof("Reaping %d timed-out sagas.", len(entities))
	for _, e := range entities {
		t, _ := tenant.Create(e.TenantId, e.TenantRegion, e.TenantMajor, e.TenantMinor)
		ctx := tenant.WithContext(tdm.Context(), t)
		processor := saga.NewProcessor(l, ctx)

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
}
