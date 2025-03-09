package main

import (
	"atlas-pets/database"
	"atlas-pets/logger"
	"atlas-pets/pet"
	"atlas-pets/service"
	"atlas-pets/tracing"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-pets"

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

	_ = database.Connect(l, database.SetMigrations(pet.Migration))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix())

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
