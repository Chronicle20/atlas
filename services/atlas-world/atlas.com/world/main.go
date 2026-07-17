package main

import (
	"atlas-world/channel"
	"atlas-world/configuration"
	"atlas-world/configuration/projection"
	channel2 "atlas-world/kafka/consumer/channel"
	"atlas-world/logger"
	"atlas-world/rate"
	"atlas-world/tasks"
	"atlas-world/world"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	service "github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

const serviceName = "atlas-world"

var consumerGroupId = consumergroup.Resolve("World Orchestrator")

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

	rc := atlas.Connect(l)
	channel.InitRegistry(rc)
	rate.InitRegistry(rc)

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	// Configuration projection: consume the tenant config-status topic and
	// gate readiness on catch-up. Created BEFORE the REST server so
	// /readyz can close over caughtUp. Replaces the legacy one-shot REST
	// load that crash-looped the pod when a tenant was provisioned after
	// start.
	state := projection.NewState()
	caughtUp := projection.NewCaughtUp()
	tenantTopic := os.Getenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS")
	if tenantTopic == "" {
		l.Warn("projection: EVENT_TOPIC_CONFIGURATION_TENANT_STATUS is not set; tenant config updates will not propagate live")
	}
	sub := &projection.Subscriber{State: state, CaughtUp: caughtUp, TenantTopic: tenantTopic}
	projectionGroupId := fmt.Sprintf("%s - projection - %s", consumerGroupId, uuid.New().String())
	if err := sub.Start(tdm.Context(), l, tdm.WaitGroup(), projectionGroupId); err != nil {
		l.WithError(err).Fatal("Unable to start configuration projection subscriber.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	channel2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := channel2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// Process-level shutting-down flag; flipped on SIGTERM teardown so
	// /readyz reports not-ready before the rest of shutdown.
	var shuttingDown atomic.Bool
	ready := func() bool { return configuration.SnapshotReady() && !shuttingDown.Load() }
	tdm.TeardownFunc(func() {
		shuttingDown.Store(true)
		l.Info("Flipped /readyz to not-ready for graceful shutdown.")
	})

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(channel.InitResource(GetServer())).
		AddRouteInitializer(world.InitResource(GetServer())).
		AddRouteInitializer(rate.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", ready)).
		Run()

	l.Infof("Service started.")

	// Gate on catch-up. A startup catch-up timeout fails loudly (k8s
	// restarts) — distinct from the request-time crash this task removes.
	ctxCaught, cancelCaught := context.WithTimeout(tdm.Context(), parseProjectionCatchupTimeout())
	if err := caughtUp.WaitCaughtUp(ctxCaught); err != nil {
		cancelCaught()
		l.WithError(err).Fatal("Configuration projection failed to catch up.")
	}
	cancelCaught()
	l.Info("Configuration projection caught up.")

	// Republish projection snapshots into the configuration package vars
	// and re-init world rates on tenant apply/change. The first publish
	// runs synchronously inside RunBridge before its ticker, so
	// GetTenantConfigs below (which blocks on readyCh) sees a populated
	// snapshot.
	routine.Go(l, tdm.Context(), func(_ context.Context) {
		configuration.RunBridge(tdm.Context(), l, state.Snapshot, time.Second, configuration.ReinitChangedRates(l))
	})

	// Boot channel-status sweep. GetTenantConfigs blocks until the bridge's
	// first publish closes readyCh; on error (not ready) log and skip
	// rather than Fatal.
	ctx, span := otel.GetTracerProvider().Tracer(serviceName).Start(context.Background(), "startup")
	if tcs, err := configuration.GetTenantConfigs(); err != nil {
		l.WithError(err).Warn("Skipping boot channel-status sweep; tenant configs not ready.")
	} else {
		_ = model.ForEachMap(model.FixedProvider(tcs), channel.RequestStatus(l)(ctx))
	}
	span.End()

	routine.Go(l, tdm.Context(), func(_ context.Context) {
		tasks.Register(l, tdm.Context())(channel.NewExpiration(l, tdm.Context(), time.Second*10))
	})

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}

// parseProjectionCatchupTimeout reads PROJECTION_CATCHUP_TIMEOUT_S from
// env (positive integer seconds) and returns the catch-up window for the
// configuration projection at startup. Default is 5 minutes, covering the
// fresh-PR-env case where atlas-pr-bootstrap is still writing the initial
// tenant configs when this pod boots.
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
