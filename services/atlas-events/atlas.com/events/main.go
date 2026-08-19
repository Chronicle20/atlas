package main

import (
	"atlas-events/event/definition"
	"atlas-events/event/occurrence"
	"atlas-events/event/orchestration"
	"atlas-events/event/registry"
	"atlas-events/event/scheduling"
	"atlas-events/event/transition"
	"atlas-events/events/anniversary"
	"atlas-events/events/crimsonbalrog"
	characterStatusConsumer "atlas-events/kafka/consumer/characterstatus"
	monsterStatusConsumer "atlas-events/kafka/consumer/monsterstatus"
	transportConsumer "atlas-events/kafka/consumer/transport"
	"context"
	"os"

	"github.com/google/uuid"

	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-events"

var consumerGroupId = consumergroup.Resolve("Events Service")

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string { return s.baseUrl }
func (s Server) GetPrefix() string  { return s.prefix }
func GetServer() Server             { return Server{baseUrl: "", prefix: "/api/"} }

// init wires the FR-A2 seam — event/definition's PATCH handler resolves
// enabled-toggles through event/orchestration.SetEnabled, so a false->true
// transition also schedules the TRIGGER_EVALUATION row (task-231 R33-3).
// event/definition cannot wire this itself without importing
// event/scheduling, which would cycle back through event/scheduling's own
// import of event/definition (event/definition/resource.go's
// EnabledOrchestrator doc comment).
//
// This lives in init(), not func main(), specifically so main_test.go can
// pin it: go test runs package init unconditionally, so a test asserting
// definition.EnabledOrchestrator != nil fails if this assignment is ever
// removed, without needing to invoke — or be able to invoke — main() itself
// (task-231 fix-round-1 finding 2: deleting this line used to compile and
// pass `go test ./...` clean while silently dropping FR-A2 in production).
func init() {
	definition.EnabledOrchestrator = orchestration.SetEnabled
}

func main() {
	rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
	l := rt.Logger()

	// registry.Register makes each event type's handler resolvable by
	// definition type (event/definition/processor.go, event/scheduling/
	// processor.go's dispatch). Until this runs, CRIMSON_BALROG's handler is
	// never constructed at all and the entire feature is unreachable at
	// runtime — a definition of this type would fail every dispatch with
	// "no handler for type CRIMSON_BALROG".
	registry.Register(crimsonbalrog.NewHandler())

	db := database.Connect(l, database.SetMigrations(
		definition.MigrateTable,
		occurrence.MigrateTable,
		transition.MigrateTable,
		scheduling.MigrateTable,
		func(db *gorm.DB) error { return db.AutoMigrate(&seeder.SeedState{}) },
	))

	// registry.Register(anniversary.NewHandler(db)) needs db, so it runs here
	// rather than beside crimsonbalrog's above — see that call's doc comment
	// for why unregistered leaves the definition type entirely unreachable.
	registry.Register(anniversary.NewHandler(db))

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	transportConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := transportConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatalf("Unable to register transport status event handlers.")
	}
	monsterStatusConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := monsterStatusConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatalf("Unable to register monster status event handlers.")
	}
	characterStatusConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := characterStatusConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatalf("Unable to register character status event handlers.")
	}
	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// The poller is the only in-memory component, and it is stateless: every
	// correctness-critical fact is a row (FR-N1). Interval, lease, batch size
	// and max attempts are configuration (FR-N16).
	pollerCfg := scheduling.ConfigFromEnv()
	routine.Go(l, rt.Context(), func(ctx context.Context) {
		p := scheduling.NewPoller(l, ctx, db, pollerCfg)
		p.SetOwnership(ownsTenant(serviceName))
		p.Run(ctx)
	})

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(definition.InitResource(GetServer())(db)).
		AddRouteInitializer(definition.InitSeedResource(GetServer())(db)).
		AddRouteInitializer(occurrence.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}

// ownsTenant answers, for one scheduled-work row, whether THIS deployment
// serves the environment that owns the row's tenant (task-232 FR-4.6).
//
// The scheduling poller is deliberately cross-tenant (design §4.2), so it is
// the one place in this service where the GORM tenant filter cannot do the
// scoping for us. Once a PR environment shares atlas-events' database with
// the baseline deployment, "every tenant's due work" would otherwise include
// the other deployment's work.
//
// It lives in main.go, not in event/scheduling, because libs/atlas-env may
// not be imported from a domain package (task-232 NG5/FR-4.5, enforced by
// tools/envguard) — the same reason atlas-saga-orchestrator resolves
// environment ownership in its own main.go rather than inside saga/.
//
// The registry is resolved on EVERY call rather than captured once: an
// environment provisioned after this pod started must be honoured without a
// restart, since a baseline pod cannot be redeployed to serve an ephemeral
// environment (FR-6.4, G7/NG6 — the same rule libs/atlas-service's
// eachOwned follows).
func ownsTenant(service string) scheduling.TenantOwnership {
	return func(tenantId uuid.UUID) bool {
		reg := env.CurrentRegistry()
		tr, ok := reg.(env.TenantResolver)
		if !ok {
			// A registry that does not project tenants is the legacy,
			// single-deployment case: everything visible is ours (FR-1.8).
			return true
		}
		e, known := tr.EnvironmentOfTenant(tenantId.String())
		if !known {
			// Unknown tenants pass through, mirroring the unknown-tenant rule
			// in libs/atlas-service's eachOwned and env.Reconcile. Treating
			// them as unowned would silently strand work for any tenant the
			// registry has not projected yet.
			return true
		}
		return reg.IsOwner(e, service)
	}
}
