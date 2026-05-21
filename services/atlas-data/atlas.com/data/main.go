package main

import (
	"atlas-data/baseline"
	"atlas-data/cash"
	"atlas-data/characters/templates"
	"atlas-data/commodity"
	"atlas-data/consumable"
	"atlas-data/cosmetic/face"
	"atlas-data/cosmetic/hair"
	"atlas-data/data"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"atlas-data/document"
	"atlas-data/equipment"
	"atlas-data/etc"
	"atlas-data/item"
	"atlas-data/job"
	data2 "atlas-data/kafka/consumer/data"
	"atlas-data/logger"
	_map "atlas-data/map"
	"atlas-data/mobskill"
	"atlas-data/monster"
	"atlas-data/npc"
	"atlas-data/pet"
	"atlas-data/quest"
	"atlas-data/reactor"
	"atlas-data/runtime/ingest"
	restruntime "atlas-data/runtime/rest"
	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"atlas-data/setup"
	"atlas-data/skill"
	minio "atlas-data/storage/minio"
	"atlas-data/tenantpurge"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"atlas-data/wzinput"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-data"

var consumerGroupId = consumergroup.Resolve("Data Service")

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

	switch os.Getenv("MODE") {
	case "ingest":
		if err := ingest.Run(tdm.Context(), l); err != nil {
			l.WithError(err).Fatal("ingest mode failed")
		}
		return
	}
	// default ("all" or empty) and MODE=rest fall through to the HTTP flow.
	// MODE=rest additionally provisions a JobCreator + Watchdog so the
	// /api/data/process handler can launch ingest Jobs.
	var jc *restruntime.JobCreator
	if os.Getenv("MODE") == "rest" {
		rdb := redis.Connect(l)
		var jcErr error
		jc, jcErr = restruntime.NewJobCreatorInClusterWithRedis(rdb)
		if jcErr != nil {
			l.WithError(jcErr).Warn("k8s in-cluster config unavailable; /api/data/process will return 503")
			jc = nil
		} else {
			if active, rerr := restruntime.RecoverActiveJobs(tdm.Context(), jc.K8s, jc.Namespace); rerr != nil {
				l.WithError(rerr).Warn("restart recovery failed")
			} else if len(active) > 0 {
				l.Infof("restart recovery: %d active ingest job(s): %v", len(active), active)
			}
			go restruntime.Watchdog{L: l, JobCreator: jc, Redis: rdb, TimeoutSecs: 1800}.Run(tdm.Context())
		}
	}

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	// MinIO client (best-effort: nil on failure, /api/data/wz handlers respond 503).
	mc, err := minio.NewClient(minio.FromEnv())
	if err != nil {
		l.WithError(err).Warn("minio client init failed; /api/data/wz will return 503")
		mc = nil
	}

	db := database.Connect(l, database.SetMigrations(
		document.Migration,
		_map.Migration,
		npc.Migration,
		monster.Migration,
		monster.SpawnIndexMigration,
		npc.SpawnIndexMigration,
		reactor.Migration,
		item.StringMigration,
		baseline.Migration,
	))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	data2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := data2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// task-071: PATCH /api/data/wz streams the canonical WZ zip — production
	// atlas.zip is ~1.6 GB. atlas-rest's default 5-second ReadTimeout cuts
	// uploads off mid-stream (observed on PR-544: 550 MB / 1.67 GB before
	// `read tcp ... i/o timeout`). atlas-ingress already allows 3600s on
	// the matching route; align the server with that so the upload has time
	// to complete. Other atlas-data endpoints don't keep connections open
	// long; raising the global timeout is safe.
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		SetReadTimeout(time.Hour).
		SetWriteTimeout(time.Hour).
		AddRouteInitializer(data.InitResource(db)(GetServer())).
		AddRouteInitializer(wzinput.InitResource(mc)(GetServer())).
		AddRouteInitializer(restruntime.InitResource(jc)(GetServer())).
		AddRouteInitializer(baseline.InitResource(db, mc)(GetServer())).
		AddRouteInitializer(tenantpurge.InitResource(db, mc)(GetServer())).
		AddRouteInitializer(_map.InitResource(db)(GetServer())).
		AddRouteInitializer(monster.InitResource(db)(GetServer())).
		AddRouteInitializer(equipment.InitResource(db)(GetServer())).
		AddRouteInitializer(reactor.InitResource(db)(GetServer())).
		AddRouteInitializer(skill.InitResource(db)(GetServer())).
		AddRouteInitializer(job.InitResource(GetServer())).
		AddRouteInitializer(pet.InitResource(db)(GetServer())).
		AddRouteInitializer(consumable.InitResource(db)(GetServer())).
		AddRouteInitializer(cash.InitResource(db)(GetServer())).
		AddRouteInitializer(commodity.InitResource(db)(GetServer())).
		AddRouteInitializer(etc.InitResource(db)(GetServer())).
		AddRouteInitializer(item.InitStringResource(db)(GetServer())).
		AddRouteInitializer(setup.InitResource(db)(GetServer())).
		AddRouteInitializer(templates.InitResource(db)(GetServer())).
		AddRouteInitializer(quest.InitResource(db)(GetServer())).
		AddRouteInitializer(npc.InitResource(db)(GetServer())).
		AddRouteInitializer(face.InitResource(db)(GetServer())).
		AddRouteInitializer(hair.InitResource(db)(GetServer())).
		AddRouteInitializer(mobskill.InitResource(db)(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
