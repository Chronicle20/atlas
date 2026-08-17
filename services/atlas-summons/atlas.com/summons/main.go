package main

import (
	"atlas-summons/summon"
	"atlas-summons/tasks"
	"atlas-summons/world"
	"context"
	"os"
	"time"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	characterevt "atlas-summons/kafka/consumer/character"
	summoncmd "atlas-summons/kafka/consumer/summon"

	"github.com/sirupsen/logrus"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	lock "github.com/Chronicle20/atlas/libs/atlas-lock"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-summons"

// withSelfEnvironment attaches this pod's own environment identity
// (env.Self()) to ctx, with no tenant pairing. summon.NewExpiryTask and
// summon.NewBeholderTask are periodic per-tenant sweeps: each rebuilds a
// per-tenant context from the summon registry's grouping, which carries no
// environment of its own (there is no inbound request to inherit ENVIRONMENT
// from). Without this, their DESTROYED/CHANGE_HP/APPLY/skill-status emits
// would carry an empty ENVIRONMENT header and fail decide() open per FR-1.8,
// actioning the sweep on every live deployment instead of just this pod's.
// summon/ is outside env-domain-guard's permitted atlas-env import list
// (main.go, kafka/, rest/), so this is threaded in as a plain function value
// rather than the package importing atlas-env itself (matches
// socket.WithSelfEnvironment, 99c0e598d).
func withSelfEnvironment(ctx context.Context) context.Context {
	return env.WithContext(ctx, env.Self())
}

var consumerGroupId = consumergroup.Resolve("Summon Registry Service")

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

	rc := atlas.Connect(l)
	summon.InitIdAllocator(rc)
	summon.InitRegistry(rc)

	// Kafka consumers: the COMMAND_TOPIC_SUMMON command consumer (SPAWN) and the
	// character-status despawn cascade (logout / channel-change / map-change).
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	summoncmd.InitConsumers(l)(cmf)(consumerGroupId)
	characterevt.InitConsumers(l)(cmf)(consumerGroupId)
	if err := summoncmd.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register summon command handlers.")
	}
	if err := characterevt.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register character status handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(summon.InitResource(GetServer())).
		AddRouteInitializer(world.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	// registerSweepTasks runs only on the leader-elected pod. It registers the
	// duration-expiry sweep that despawns summons whose lifetime has elapsed, and
	// the Beholder aura sweep that heals/buffs owners of deployed Beholders.
	registerSweepTasks := func(l logrus.FieldLogger, ctx context.Context) {
		tasks.Register(l, ctx)(summon.NewExpiryTask(l, ctx, time.Second, withSelfEnvironment))
		tasks.Register(l, ctx)(summon.NewBeholderTask(l, ctx, time.Second, withSelfEnvironment))
	}

	if leaderEnabled(l) {
		ttl := leaderTTL(l)
		le, err := lock.New(rc, "summons-sweep",
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
				registerSweepTasks(l, leaderCtx)
				<-leaderCtx.Done()
			})
			if err != nil {
				l.WithError(err).Errorf("LeaderElection.Run exited with error.")
			}
		})
	} else {
		l.Warnf("SUMMON_LEADER_ELECTION_ENABLED=false — sweep tasks run unconditionally on this pod.")
		registerSweepTasks(l, rt.Context())
	}

	rt.Wait()
}
