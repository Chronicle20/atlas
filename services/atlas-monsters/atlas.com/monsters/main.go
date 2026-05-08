package main

import (
	_map "atlas-monsters/kafka/consumer/map"
	monster2 "atlas-monsters/kafka/consumer/monster"
	"atlas-monsters/logger"
	"atlas-monsters/monster"
	"atlas-monsters/monster/information"
	"atlas-monsters/tasks"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"atlas-monsters/world"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const serviceName = "atlas-monsters"

var consumerGroupId = consumergroup.Resolve("Monster Registry Service")

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
	if err := monster2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	if err := _map.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
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

	tasks.Register(l, tdm.Context())(monster.NewRegistryAudit(l, time.Second*30))
	tasks.Register(l, tdm.Context())(monster.NewStatusExpirationTask(l, tdm.Context(), time.Second))
	tasks.Register(l, tdm.Context())(monster.NewDropTimerTask(l, tdm.Context(), time.Second))
	tasks.Register(l, tdm.Context())(monster.NewMonsterAggroDecayTask(l, tdm.Context(), monster.AggroSweepInterval))
	tasks.Register(l, tdm.Context())(monster.NewMonsterSkillPickerSweepTask(l, tdm.Context(), monster.MonsterSkillPickerSweepInterval))
	tasks.Register(l, tdm.Context())(monster.NewMonsterRecoveryTask(l, tdm.Context(), monster.MonsterRecoveryInterval))

	tdm.TeardownFunc(monster.Teardown(l))
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()

	l.Infoln("Service shutdown.")
}
