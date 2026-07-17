package main

import (
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	"atlas-doors/character"
	"atlas-doors/door"
	character2 "atlas-doors/kafka/consumer/character"
	door2 "atlas-doors/kafka/consumer/door"
	party2 "atlas-doors/kafka/consumer/party"
	"atlas-doors/tasks"
	"atlas-doors/world"
	"context"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	lock "github.com/Chronicle20/atlas/libs/atlas-lock"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"github.com/sirupsen/logrus"
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
	rt := service.Bootstrap(serviceName)
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
		tasks.Register(l, ctx)(door.NewExpiryTask(l, ctx, time.Second))
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
		l.Warnf("DOOR_LEADER_ELECTION_ENABLED=false — sweep tasks run unconditionally on this pod.")
		registerSweepTasks(l, rt.Context())
	}

	rt.Wait()
}
