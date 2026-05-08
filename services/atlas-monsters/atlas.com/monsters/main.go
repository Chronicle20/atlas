package main

import (
	data2 "atlas-monsters/kafka/consumer/data"
	_map "atlas-monsters/kafka/consumer/map"
	monster2 "atlas-monsters/kafka/consumer/monster"
	"atlas-monsters/logger"
	"atlas-monsters/monster"
	"atlas-monsters/monster/information"
	"atlas-monsters/tasks"
	"atlas-monsters/world"
	"context"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	lock "github.com/Chronicle20/atlas/libs/atlas-lock"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

const serviceName = "atlas-monsters"
const consumerGroupId = "Monster Registry Service"
const dataEventsConsumerGroupId = "Monster Data Cache Invalidator"

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

	rc := atlas.Connect(l)
	monster.InitIdAllocator(rc)
	monster.InitCooldownRegistry(rc)
	monster.InitAttackCooldownRegistry(rc)
	monster.InitMonsterRegistry(rc)
	monster.InitDropTimerRegistry(rc)
	information.InitDataCache(rc)

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
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

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(monster.InitResource(GetServer())).
		AddRouteInitializer(world.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/metrics", promhttp.Handler())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
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
		go func() {
			err := le.Run(tdm.Context(), func(leaderCtx context.Context) {
				registerSweepTasks(l, leaderCtx)
				<-leaderCtx.Done()
			})
			if err != nil {
				l.WithError(err).Errorf("LeaderElection.Run exited with error.")
			}
		}()
	} else {
		l.Warnf("MONSTER_LEADER_ELECTION_ENABLED=false — sweep tasks run unconditionally on this pod.")
		registerSweepTasks(l, tdm.Context())
	}

	tdm.TeardownFunc(monster.Teardown(l))
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()

	l.Infoln("Service shutdown.")
}
