package main

import (
	"atlas-drops-information/continent"
	drop2 "atlas-drops-information/continent/drop"
	"atlas-drops-information/database"
	"atlas-drops-information/logger"
	"atlas-drops-information/monster/drop"
	"atlas-drops-information/seed"
	"atlas-drops-information/service"
	"atlas-drops-information/tracing"
	"github.com/Chronicle20/atlas-rest/server"
)

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

	db := database.Connect(l, database.SetMigrations(drop.Migration, drop2.Migration))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), drop.InitResource(GetServer())(db), continent.InitResource(GetServer())(db), seed.InitResource(GetServer())(db))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
