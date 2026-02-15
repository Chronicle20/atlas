package main

import (
	"atlas-party-quests/database"
	"atlas-party-quests/definition"
	"atlas-party-quests/instance"
	pqConsumer "atlas-party-quests/kafka/consumer/party_quest"
	"atlas-party-quests/logger"
	"atlas-party-quests/service"
	tenant2 "atlas-party-quests/tenant"
	"atlas-party-quests/tracing"
	"os"
	"time"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas-tenant"
)

const serviceName = "atlas-party-quests"
const consumerGroupId = "Party Quest Service"

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

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(definition.MigrateTable))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	pqConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	pqConsumer.InitHandlers(l, db)(consumer.GetManager().RegisterHandler)

	tenants, err := tenant2.NewProcessor(l, tdm.Context()).GetAll()
	if err != nil {
		l.WithError(err).Fatal("Unable to load tenants.")
	}

	// Start background ticker for PQ timers
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-tdm.Context().Done():
				return
			case <-ticker.C:
				for _, t := range tenants {
					ctx := tenant.WithContext(tdm.Context(), t)
					ip := instance.NewProcessor(l, ctx, db)
					_ = ip.TickGlobalTimerAndEmit()
					_ = ip.TickStageTimerAndEmit()
					_ = ip.TickBonusTimerAndEmit()
					_ = ip.TickRegistrationTimerAndEmit()
				}
			}
		}
	}()

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(definition.InitResource(GetServer())(db)).
		AddRouteInitializer(instance.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.TeardownFunc(func() {
		l.Infoln("Graceful shutdown: handling active PQ instances.")
		for _, t := range tenants {
			ctx := tenant.WithContext(tdm.Context(), t)
			_ = instance.NewProcessor(l, ctx, db).GracefulShutdownAndEmit()
		}
	})

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
