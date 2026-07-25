package main

import (
	"atlas-world/broadcast"
	"atlas-world/channel"
	"atlas-world/configuration"
	"atlas-world/configuration/projection"
	broadcast2 "atlas-world/kafka/consumer/broadcast"
	channel2 "atlas-world/kafka/consumer/channel"
	"atlas-world/rate"
	"atlas-world/tasks"
	"atlas-world/world"
	"context"
	"os"
	"time"

	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"go.opentelemetry.io/otel"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	lock "github.com/Chronicle20/atlas/libs/atlas-lock"
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
	state := projection.NewState()
	caughtUp := projection.NewCaughtUp()

	rt := service.Bootstrap(serviceName,
		service.WithConfigProjection(consumerGroupId, func(t service.ProjectionTopics) service.Projection {
			sub := &projection.Subscriber{State: state, CaughtUp: caughtUp, TenantTopic: t.TenantStatus}
			return service.ProjectionFuncs{StartFunc: sub.Start, WaitCaughtUpFunc: caughtUp.WaitCaughtUp}
		}),
		service.WithReadinessGate(configuration.SnapshotReady),
	)
	l := rt.Logger()

	rc := atlas.Connect(l)
	channel.InitRegistry(rc)
	rate.InitRegistry(rc)
	broadcast.InitRegistry(rc)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	channel2.InitConsumers(l)(cmf)(consumerGroupId)
	broadcast2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := channel2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := broadcast2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register broadcast command handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(channel.InitResource(GetServer())).
		AddRouteInitializer(world.InitResource(GetServer())).
		AddRouteInitializer(rate.InitResource(GetServer())).
		AddRouteInitializer(broadcast.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	l.Infof("Service started.")

	rt.AwaitProjectionCatchUp()
	l.Info("Configuration projection caught up.")

	// Republish projection snapshots into the configuration package vars
	// and re-init world rates on tenant apply/change. The first publish
	// runs synchronously inside RunBridge before its ticker, so
	// GetTenantConfigs below (which blocks on readyCh) sees a populated
	// snapshot.
	routine.Go(l, rt.Context(), func(_ context.Context) {
		configuration.RunBridge(rt.Context(), l, state.Snapshot, time.Second, configuration.ReinitChangedRates(l))
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

	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(channel.NewExpiration(l, rt.Context(), time.Second*10))
	})

	// registerBroadcastSweep runs only on the leader-elected pod. atlas-world
	// runs replicas:2 with no leader election otherwise; without this gate
	// both pods would sweep every second and STARTED/ENDED status events
	// would double-fire continuously.
	registerBroadcastSweep := func(l logrus.FieldLogger, ctx context.Context) {
		tasks.Register(l, ctx)(broadcast.NewSweep(l, ctx, time.Second))
	}

	if leaderEnabled(l) {
		ttl := leaderTTL(l)
		le, err := lock.New(rc, "world-broadcast-sweep",
			lock.WithTTL(ttl),
			lock.WithRefreshInterval(leaderRefresh(l, ttl)),
			lock.WithBackoff(leaderBackoff(l)),
			lock.WithLogger(l),
		)
		if err != nil {
			l.WithError(err).Fatal("Unable to construct LeaderElection.")
		}
		routine.Go(l, rt.Context(), func(_ context.Context) {
			err := le.Run(rt.Context(), func(leaderCtx context.Context) {
				registerBroadcastSweep(l, leaderCtx)
				<-leaderCtx.Done()
			})
			if err != nil {
				l.WithError(err).Errorf("LeaderElection.Run exited with error.")
			}
		})
	} else {
		l.Warnf("WORLD_BROADCAST_LEADER_ELECTION_ENABLED=false — broadcast sweep runs unconditionally on this pod.")
		registerBroadcastSweep(l, rt.Context())
	}

	rt.Wait()
}
