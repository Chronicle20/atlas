package main

import (
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/projection"
	"atlas-character-factory/factory"
	"atlas-character-factory/kafka/consumer/saga"
	"atlas-character-factory/logger"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"github.com/google/uuid"
)

const serviceName = "atlas-character-factory"

var consumerGroupId = consumergroup.Resolve("Character Factory Service")

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

	// Configuration projection: consume the tenant config-status topic,
	// gate readiness on catch-up, then republish snapshots into the
	// configuration package vars. Replaces the legacy one-shot REST load
	// that crash-looped the pod when a tenant was provisioned after start.
	state := projection.NewState()
	caughtUp := projection.NewCaughtUp()
	tenantTopic := os.Getenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS")
	if tenantTopic == "" {
		l.Warn("projection: EVENT_TOPIC_CONFIGURATION_TENANT_STATUS is not set; tenant config updates will not propagate live")
	}
	sub := &projection.Subscriber{State: state, CaughtUp: caughtUp, TenantTopic: tenantTopic}
	// Per-process group id so each container start replays the full
	// compacted log from FirstOffset (a shared group id would resume from
	// the previous run's committed offset and never refill State).
	projectionGroupId := fmt.Sprintf("%s - projection - %s", consumerGroupId, uuid.New().String())
	if err := sub.Start(tdm.Context(), l, tdm.WaitGroup(), projectionGroupId); err != nil {
		l.WithError(err).Fatal("Unable to start configuration projection subscriber.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	saga.InitConsumers(l)(cmf)(consumerGroupId)
	if err := saga.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// Gate startup on catch-up. A startup catch-up timeout fails loudly
	// (no traffic served yet; k8s restarts) — distinct from the
	// request-time crash this task eliminates.
	ctxCaught, cancelCaught := context.WithTimeout(tdm.Context(), parseProjectionCatchupTimeout())
	if err := caughtUp.WaitCaughtUp(ctxCaught); err != nil {
		cancelCaught()
		l.WithError(err).Fatal("Configuration projection failed to catch up.")
	}
	cancelCaught()
	l.Info("Configuration projection caught up.")

	// Process-level shutting-down flag; flipped on SIGTERM teardown so
	// /readyz reports not-ready before the rest of shutdown.
	var shuttingDown atomic.Bool
	ready := func() bool { return caughtUp.CaughtUpNow() && !shuttingDown.Load() }
	tdm.TeardownFunc(func() {
		shuttingDown.Store(true)
		l.Info("Flipped /readyz to not-ready for graceful shutdown.")
	})

	// Republish projection snapshots into the configuration package vars
	// so GetTenantConfig callers (the seed saga, preset client) see live
	// updates. onChange is nil — the factory has no per-change side effects.
	routine.Go(l, tdm.Context(), func(_ context.Context) {
		configuration.RunBridge(tdm.Context(), l, state.Snapshot, time.Second, nil)
	})

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(factory.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", ready)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}

// parseProjectionCatchupTimeout reads PROJECTION_CATCHUP_TIMEOUT_S from
// env (positive integer seconds) and returns the catch-up window for the
// configuration projection at startup. Default is 5 minutes, which covers
// the fresh-PR-env case where atlas-pr-bootstrap is still writing the
// initial tenant configs when this pod boots.
func parseProjectionCatchupTimeout() time.Duration {
	const def = 5 * time.Minute
	v := os.Getenv("PROJECTION_CATCHUP_TIMEOUT_S")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return time.Duration(n) * time.Second
}
