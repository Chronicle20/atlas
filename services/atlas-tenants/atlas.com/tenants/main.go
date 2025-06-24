package main

import (
	"atlas-tenants/database"
	"atlas-tenants/logger"
	"atlas-tenants/service"
	"atlas-tenants/tenant"
	"atlas-tenants/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"os"
)

const serviceName = "atlas-tenants"
const consumerGroupId = "Tenant Service"

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

	db := database.Connect(l, database.SetMigrations(tenant.MigrateEntities))

	_ = consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())

	// Create and run server
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		AddRouteInitializer(tenant.RegisterRoutes(db)(GetServer())).
		SetPort(os.Getenv("REST_PORT")).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
