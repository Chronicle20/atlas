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
	"atlas-data/document"
	"atlas-data/equipment"
	"atlas-data/etc"
	"atlas-data/item"
	"atlas-data/job"
	data2 "atlas-data/kafka/consumer/data"
	_map "atlas-data/map"
	"atlas-data/mobskill"
	"atlas-data/monster"
	"atlas-data/npc"
	"atlas-data/pet"
	"atlas-data/quest"
	"atlas-data/reactor"
	"atlas-data/runtime/ingest"
	restruntime "atlas-data/runtime/rest"
	"atlas-data/setup"
	"atlas-data/skill"
	minio "atlas-data/storage/minio"
	"atlas-data/tenantpurge"
	"atlas-data/wzinput"
	"context"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
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
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	switch os.Getenv("MODE") {
	case "ingest":
		if err := ingest.Run(rt.Context(), l); err != nil {
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
			if active, rerr := restruntime.RecoverActiveJobs(rt.Context(), jc.K8s, jc.Namespace); rerr != nil {
				l.WithError(rerr).Warn("restart recovery failed")
			} else if len(active) > 0 {
				l.Infof("restart recovery: %d active ingest job(s): %v", len(active), active)
			}
			// TimeoutSecs is the maximum heartbeat staleness the Watchdog
			// tolerates before deleting a Job. The ingest pod now refreshes
			// its heartbeat every 30s (runtime/ingest/heartbeat.go), so any
			// timeout > ~60s would suffice in the happy path. Pick 7200 (2 h)
			// as a generous belt-and-braces margin for a wedged heartbeat
			// goroutine or a transient Redis blip on the writer side, and to
			// absorb future archive growth without a code change. The legacy
			// value (1800) was a self-inflicted half-hour cap: with no in-pod
			// heartbeat, every Job's heartbeat went stale at creation+timeout
			// regardless of actual progress (PR-544: Map worker killed at
			// 30:28 mid-loop, ~80 maps left without layout.json/minimap.png).
			routine.Go(l, rt.Context(), func(_ context.Context) {
				restruntime.Watchdog{L: l, JobCreator: jc, TimeoutSecs: 7200}.Run(rt.Context())
			})
		}
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

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	// Heal any baseline restore that was interrupted (pod killed / cancelled
	// mid-COPY) before this pod started: such a tenant carries a durable
	// StatusRestoring marker and otherwise stays half-restored until an operator
	// notices (the atlas-pr-933 empty item-search bug). Non-blocking.
	baseline.Reconcile(rt.Context(), l, db, mc)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	data2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := data2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// task-071: PATCH /api/data/wz streams the canonical WZ zip — production
	// atlas.zip is ~1.6 GB. atlas-rest's default 5-second ReadTimeout cuts
	// uploads off mid-stream (observed on PR-544: 550 MB / 1.67 GB before
	// `read tcp ... i/o timeout`). atlas-ingress already allows 3600s on
	// the matching route; align the server with that so the upload has time
	// to complete. Other atlas-data endpoints don't keep connections open
	// long; raising the global timeout is safe.
	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
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
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
