package main

import (
	"atlas-configurations/database"
	"atlas-configurations/logger"
	"atlas-configurations/seeder"
	"atlas-configurations/service"
	"atlas-configurations/services"
	"atlas-configurations/templates"
	"atlas-configurations/tenants"
	"atlas-configurations/tracing"
	"os"

	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-configurations"

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

	db := database.Connect(l, database.SetMigrations(templates.Migration, tenants.Migration, services.Migration))

	// Run seed import
	seedConfig := seeder.DefaultConfig()
	l.WithFields(map[string]interface{}{
		"seedPath":    seedConfig.SeedPath,
		"seedEnabled": seedConfig.Enabled,
	}).Info("Seed configuration loaded")
	s := seeder.NewSeeder(l, tdm.Context(), db, seedConfig)
	if err := s.Run(); err != nil {
		l.WithError(err).Error("Seed import failed")
	}

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(templates.InitResource(GetServer())(db)).
		AddRouteInitializer(tenants.InitResource(GetServer())(db)).
		AddRouteInitializer(services.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
