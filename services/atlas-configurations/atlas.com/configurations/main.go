package main

import (
	"atlas-configurations/configuration"
	"atlas-configurations/database"
	"atlas-configurations/logger"
	"atlas-configurations/service"
	"atlas-configurations/templates"
	"atlas-configurations/tenants"
	"atlas-configurations/tracing"
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

	db := database.Connect(l, database.SetMigrations(configuration.Migration, templates.Migration, tenants.Migration))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), templates.InitResource(GetServer())(db), tenants.InitResource(GetServer())(db))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
