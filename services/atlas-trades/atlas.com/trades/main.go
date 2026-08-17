package main

import (
	"atlas-trades/escrow"
	characterconsumer "atlas-trades/kafka/consumer/character"
	custodyconsumer "atlas-trades/kafka/consumer/custody"
	inviteconsumer "atlas-trades/kafka/consumer/invite"
	sagaconsumer "atlas-trades/kafka/consumer/saga"
	sessionconsumer "atlas-trades/kafka/consumer/session"
	tradeconsumer "atlas-trades/kafka/consumer/trade"
	"atlas-trades/ledger"
	"atlas-trades/settlement"
	"atlas-trades/trade"
	"context"
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-trades"

// withSelfEnvironment attaches this pod's own environment identity
// (env.Self()) to ctx, with no tenant pairing. trade.detached() and the boot
// reconciliation passes (trade.Reconcile, trade.ReconcileEscrow) rebuild a
// per-row context with no request in flight to inherit ENVIRONMENT from;
// without this, their REST calls and Kafka emits would resolve to the
// baseline URL/topic regardless of which environment the pod belongs to
// (FR-1.8, FR-3.1/FR-3.2). trade/ is outside env-domain-guard's permitted
// atlas-env import list, so this is threaded in as a plain function value
// rather than the package importing atlas-env itself (matches
// socket.WithSelfEnvironment, 99c0e598d).
func withSelfEnvironment(ctx context.Context) context.Context {
	return env.WithContext(ctx, env.Self())
}

// consumerGroupId is the Kafka consumer group this service registers under. It
// is also the literal that deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
// reads out of this file to derive KAFKA_CONSUMER_GROUP for ephemeral envs, and
// that tools/service-registration-guard.sh mirrors to decide whether this
// service needs a generated PR consumer-group document. Deleting or renaming it
// would silently drop both.
var consumerGroupId = consumergroup.Resolve("Trade Service")

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

	// Arm trade's environment-origination DI (settlement.go's applyEnvContext)
	// before anything that could reach it runs: the attestation deadline
	// registry, the boot reconciliation passes, and every consumer command.
	trade.SetEnvContext(withSelfEnvironment)

	// Transient DB-connection errors surface as 503 + Retry-After (DOM-27,
	// task-168) instead of 500, so callers retry rather than treating a
	// pool blip as a hard failure.
	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	// Live room state is the process-local in-memory trade.Registry, not the
	// DB — atlas-trades runs replicas: 1 for that reason (design §9). The DB
	// backs only the completed-trade ledger and the transactional outbox.
	db := database.Connect(l, database.SetMigrations(ledger.Migration, settlement.Migration, escrow.Migration, outboxlib.Migration))

	// Boot the outbox drainer: publishes the transactional outbox to Kafka.
	// Leadership is gated by a postgres advisory lock — replicas are safe.
	publisher := outboxlib.NewTopicWriterPool()
	drainer := outboxlib.NewDrainer(l, db, publisher, outboxlib.WithDSN(database.DSN()))
	routine.Go(l, rt.Context(), func(_ context.Context) {
		drainer.Run(rt.Context())
	})
	rt.TeardownFunc(func() {
		drainer.Stop()
		publisher.Close()
	})

	// Trade room lifecycle commands from atlas-channel (create/invite/decline/
	// enter), plus the answer half of the invites we issue: an accepted TRADE
	// invite seats the visitor, a rejected or expired one tears the room down.
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	custodyconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := custodyconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatalf("Unable to register trade custody command handlers.")
	}

	tradeconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := tradeconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	inviteconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := inviteconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	// Terminal settlement outcomes. A trade's LEAVE 7 / LEAVE 8 is produced
	// here, not when the saga is submitted (design §6.4).
	sagaconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := sagaconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	// Teardown triggers (design §3.3). A logout, a map change and a channel
	// change each end an open trade, with the leaveReason their trigger implies;
	// a destroyed session covers the client that vanished without a clean
	// logout. All four lose to an in-flight settlement (FR-6.5).
	characterconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := characterconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	sessionconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := sessionconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	// There is no reservation-refresh loop. Under reserve-at-staging one was
	// mandatory: a staged item was held by an atlas-inventory reservation with a
	// TTL, and a trade window left open longer than the TTL would settle onto an
	// expired hold. Escrow has no TTL to outlive — the asset is in atlas-trades'
	// own custody table — so the loop was deleted rather than retuned
	// (design §5A.10).

	// Attestation deadlines are sleeping goroutines; without this, shutdown
	// waits out the longest one.
	// Settlements submitted by a previous process whose terminal status this one
	// never saw. Off the boot path in its own goroutine: the orchestrator is a
	// REST hop away and a reconciliation failure must not wedge startup — every
	// record it cannot resolve is simply left for the next boot or for the live
	// status event.
	routine.Go(l, rt.Context(), func(ctx context.Context) {
		if err := trade.ReconcileAtBoot(l, ctx, db); err != nil {
			l.WithError(err).Error("Unable to reconcile in-flight trade settlements and stranded escrow. Both are durable and are retried at the next boot.")
		}
	})

	rt.TeardownFunc(func() { trade.GetAttestationTimers().StopAll() })
	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(trade.InitResource(GetServer())(db)).
		AddRouteInitializer(ledger.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
