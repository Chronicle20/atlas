package main

import (
	data2 "atlas-monsters/kafka/consumer/data"
	_map "atlas-monsters/kafka/consumer/map"
	monster2 "atlas-monsters/kafka/consumer/monster"
	"atlas-monsters/monster"
	"atlas-monsters/monster/information"
	"atlas-monsters/tasks"
	"atlas-monsters/world"
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	lock "github.com/Chronicle20/atlas/libs/atlas-lock"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-monsters"

var (
	consumerGroupId           = consumergroup.Resolve("Monster Registry Service")
	dataEventsConsumerGroupId = consumergroup.Resolve("Monster Data Cache Invalidator")
)

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
	monster.InitIdAllocator(rc)
	monster.InitCooldownRegistry(rc)
	monster.InitAttackCooldownRegistry(rc)
	monster.InitMonsterRegistry(rc)
	monster.InitDropTimerRegistry(rc)
	monster.InitPuppetRegistry(rc)
	information.InitDataCache(rc)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	monster2.InitConsumers(l)(cmf)(consumerGroupId)
	_map.InitConsumers(l)(cmf)(consumerGroupId)
	data2.InitConsumers(l)(cmf)(dataEventsConsumerGroupId)
	if err := monster2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := _map.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := data2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register data-events kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(monster.InitResource(GetServer())).
		AddRouteInitializer(world.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	registerSweepTasks := func(l logrus.FieldLogger, ctx context.Context) {
		tasks.Register(l, ctx)(monster.NewRegistryAudit(l, time.Second*30))
		tasks.Register(l, ctx)(monster.NewStatusExpirationTask(l, ctx, time.Second))
		tasks.Register(l, ctx)(monster.NewDropTimerTask(l, ctx, time.Second))
		tasks.Register(l, ctx)(monster.NewMonsterAggroDecayTask(l, ctx, monster.AggroSweepInterval))
		tasks.Register(l, ctx)(monster.NewMonsterSkillPickerSweepTask(l, ctx, monster.MonsterSkillPickerSweepInterval))
		tasks.Register(l, ctx)(monster.NewMonsterRecoveryTask(l, ctx, monster.MonsterRecoveryInterval))
	}

	if leaderEnabled(l) {
		ttl := leaderTTL(l)
		le, err := lock.New(rc, "monsters-sweep",
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
		l.Warnf("MONSTER_LEADER_ELECTION_ENABLED=false — sweep tasks run unconditionally on this pod.")
		registerSweepTasks(l, rt.Context())
	}

	rt.TeardownFunc(monster.Teardown(l))

	rt.Wait()
}
