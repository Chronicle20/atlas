package main

import (
	"atlas-doors/character"
	"atlas-doors/door"
	"atlas-doors/world"
	"context"
	"os"
	"time"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	character2 "atlas-doors/kafka/consumer/character"
	door2 "atlas-doors/kafka/consumer/door"
	party2 "atlas-doors/kafka/consumer/party"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	lock "github.com/Chronicle20/atlas/libs/atlas-lock"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-doors"

var consumerGroupId = consumergroup.Resolve("Door Registry Service")

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
	door.InitIdAllocator(rc)
	door.InitRegistry(rc)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	door2.InitConsumers(l)(cmf)(consumerGroupId)
	character2.InitConsumers(l)(cmf)(consumerGroupId)
	party2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := door2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := character2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := party2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(door.InitResource(GetServer())).
		AddRouteInitializer(world.InitResource(GetServer())).
		AddRouteInitializer(character.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	registerSweepTasks := func(l logrus.FieldLogger, ctx context.Context) {
		// door sits outside env-domain-guard's permitted atlas-env import
		// list (main.go, kafka/, rest/, socket/), so this pod's environment
		// identity is threaded in as a plain function value rather than the
		// package importing atlas-env itself. Without it, ExpiryTask's
		// per-tenant REMOVED Kafka events would carry an empty environment
		// header and fail decide() open per FR-1.8.
		routine.Register(l, ctx, rt.WaitGroup())(door.NewExpiryTask(l, time.Second, func(ctx context.Context) context.Context {
			return env.WithContext(ctx, env.Self())
		}))
	}

	if leaderEnabled(l) {
		ttl := leaderTTL(l)
		le, err := lock.New(rc, "doors-sweep",
			lock.WithTTL(ttl),
			lock.WithRefreshInterval(leaderRefresh(l, ttl)),
			lock.WithBackoff(leaderBackoff(l)),
			lock.WithLogger(l),
		)
		if err != nil {
			l.WithError(err).Fatal("Unable to construct LeaderElection.")
		}
		// Held open for the lifetime of the leader-election goroutine below.
		// registerSweepTasks (and therefore its routine.Register call) only
		// runs once le.Run elects this pod as leader, which happens
		// asynchronously inside the routine.Go goroutine. Registering there
		// calls wg.Add(1) on a detached goroutine, which would race
		// Manager.Wait() if the counter were allowed to hit zero in between.
		// Adding 1 here — synchronously, before routine.Go starts — keeps the
		// counter above zero for the goroutine's whole life, so that later
		// Add can never race a Wait-induced zero crossing.
		rt.WaitGroup().Add(1)
		routine.Go(l, rt.Context(), func(_ context.Context) {
			defer rt.WaitGroup().Done()
			err := le.Run(rt.Context(), func(leaderCtx context.Context) {
				registerSweepTasks(l, leaderCtx)
				<-leaderCtx.Done()
			})
			if err != nil {
				l.WithError(err).Errorf("LeaderElection.Run exited with error.")
			}
		})
	} else {
		l.Warnf("DOOR_LEADER_ELECTION_ENABLED=false — sweep tasks run unconditionally on this pod.")
		registerSweepTasks(l, rt.Context())
	}

	rt.Wait()
}
