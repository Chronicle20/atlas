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
	restmode "atlas-data/runtime/rest"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"atlas-data/setup"
	"atlas-data/skill"
	minio "atlas-data/storage/minio"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"atlas-data/wzinput"
	"os"

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
	case "rest":
		if err := restmode.Run(tdm.Context(), l); err != nil {
			l.WithError(err).Fatal("rest mode failed")
		}
		return
	case "ingest":
		if err := ingest.Run(tdm.Context(), l); err != nil {
			l.WithError(err).Fatal("ingest mode failed")
		}
		return
	}
	// default ("all" or empty) falls through to existing main flow.

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

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(data.InitResource(db)(GetServer())).
		AddRouteInitializer(wzinput.InitResource(mc)(GetServer())).
		AddRouteInitializer(baseline.InitResource(db, mc)(GetServer())).
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
