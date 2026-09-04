package main

import (
	"atlas-monsters/character/hidden"
	buffconsumer "atlas-monsters/kafka/consumer/buff"
	data2 "atlas-monsters/kafka/consumer/data"
	_map "atlas-monsters/kafka/consumer/map"
	monster2 "atlas-monsters/kafka/consumer/monster"
	"atlas-monsters/monster"
	"atlas-monsters/monster/information"
	"atlas-monsters/world"
	"context"
	"os"
	"sync"
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
	rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
	l := rt.Logger()

	rc := atlas.Connect(l)
	monster.InitIdAllocator(rc)
	monster.InitCooldownRegistry(rc)
	monster.InitAttackCooldownRegistry(rc)
	monster.InitMonsterRegistry(rc)
	monster.InitDropTimerRegistry(rc)
	monster.InitSelfDestructTimerRegistry(rc)
	monster.InitPuppetRegistry(rc)
	hidden.InitRegistry(rc)
	information.InitDataCache(rc)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	monster2.InitConsumers(l)(cmf)(consumerGroupId)
	_map.InitConsumers(l)(cmf)(consumerGroupId)
	buffconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	data2.InitConsumers(l)(cmf)(dataEventsConsumerGroupId)
	if err := monster2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := _map.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := buffconsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
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

	registerSweepTasks := func(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) {
		routine.Register(l, ctx, wg)(monster.NewRegistryAudit(l, time.Second*30))
		routine.Register(l, ctx, wg)(monster.NewStatusExpirationTask(l, time.Second))
		routine.Register(l, ctx, wg)(monster.NewDropTimerTask(l, time.Second))
		routine.Register(l, ctx, wg)(monster.NewSelfDestructTimerTask(l, time.Second))
		routine.Register(l, ctx, wg)(monster.NewMonsterAggroDecayTask(l, ctx, monster.AggroSweepInterval))
		routine.Register(l, ctx, wg)(monster.NewMonsterSkillPickerSweepTask(l, ctx, monster.MonsterSkillPickerSweepInterval))
		routine.Register(l, ctx, wg)(monster.NewMonsterRecoveryTask(l, ctx, monster.MonsterRecoveryInterval))
		routine.Register(l, ctx, wg)(hidden.NewReconciliationTask(l, ctx, hidden.ReconcileInterval))
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
				registerSweepTasks(l, leaderCtx, rt.WaitGroup())
				<-leaderCtx.Done()
			})
			if err != nil {
				l.WithError(err).Errorf("LeaderElection.Run exited with error.")
			}
		})
	} else {
		l.Warnf("MONSTER_LEADER_ELECTION_ENABLED=false — sweep tasks run unconditionally on this pod.")
		registerSweepTasks(l, rt.Context(), rt.WaitGroup())
	}

	rt.TeardownFunc(monster.Teardown(l))

	rt.Wait()
}
