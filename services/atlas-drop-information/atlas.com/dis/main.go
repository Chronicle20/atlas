package main

import (
	"atlas-drops-information/configuration"
	"atlas-drops-information/continent"
	drop2 "atlas-drops-information/continent/drop"
	"atlas-drops-information/database"
	"atlas-drops-information/logger"
	"atlas-drops-information/monster/drop"
	"atlas-drops-information/service"
	"atlas-drops-information/tracing"
	"context"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)
import "gorm.io/gorm"

const serviceName = "atlas-drops-information"

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

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	//configuration.Init(l)(tdm.Context())(uuid.MustParse(os.Getenv("SERVICE_ID")), os.Getenv("SERVICE_TYPE"))
	//c, err := configuration.Get()
	//if err != nil {
	//	l.WithError(err).Fatal("Unable to successfully load configuration.")
	//}

	db := database.Connect(l, database.SetMigrations(drop.Migration, drop2.Migration))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), drop.InitResource(GetServer())(db), continent.InitResource(GetServer())(db))

	//initializeMonsterDrops(l)(*c)(db)
	//initializeContinentDrops(l)(*c)(db)

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}

func initializeMonsterDrops(l logrus.FieldLogger) func(c configuration.RestModel) func(db *gorm.DB) {
	return func(c configuration.RestModel) func(db *gorm.DB) {
		return func(db *gorm.DB) {
			for _, s := range c.Servers {
				t, err := tenant.Create(s.TenantId, "", 0, 0)
				if err != nil {
					continue
				}
				tctx := tenant.WithContext(context.Background(), t)

				ds, _ := drop.GetAll(l)(tctx)(db)
				if len(ds) > 0 {
					continue
				}

				var drops []drop.Model
				for _, mon := range s.Monsters {
					for _, d := range mon.Items {
						md := drop.NewMonsterDropBuilder(t.Id(), 0).
							SetMonsterId(uint32(mon.Id)).
							SetItemId(d.ItemId).
							SetMinimumQuantity(d.MinimumQuantity).
							SetMaximumQuantity(d.MaximumQuantity).
							SetQuestId(d.QuestId).
							SetChance(d.Chance).
							Build()
						drops = append(drops, md)
					}
				}
				err = drop.BulkCreateMonsterDrop(db, drops)
				if err != nil {
					l.Fatalf(err.Error())
				}
			}
		}
	}
}

func initializeContinentDrops(l logrus.FieldLogger) func(c configuration.RestModel) func(db *gorm.DB) {
	return func(c configuration.RestModel) func(db *gorm.DB) {
		return func(db *gorm.DB) {
			for _, s := range c.Servers {
				t, err := tenant.Create(s.TenantId, "", 0, 0)
				if err != nil {
					continue
				}
				tctx := tenant.WithContext(context.Background(), t)

				ds, _ := drop2.GetAll(l)(tctx)(db)()
				if len(ds) > 0 {
					continue
				}

				var drops []drop2.Model

				for _, con := range s.Continents {
					for _, d := range con.Items {
						md := drop2.NewContinentDropBuilder(t.Id(), 0).
							SetContinentId(con.Id).
							SetItemId(d.ItemId).
							SetMinimumQuantity(d.MinimumQuantity).
							SetMaximumQuantity(d.MaximumQuantity).
							SetQuestId(d.QuestId).
							SetChance(d.Chance).
							Build()
						drops = append(drops, md)
					}
				}

				err = drop2.BulkCreateContinentDrop(db, drops)
				if err != nil {
					l.Fatalf(err.Error())
				}
			}
		}
	}
}
