package main

import (
	"atlas-drops-information/continent"
	drop2 "atlas-drops-information/continent/drop"
	"atlas-drops-information/database"
	"atlas-drops-information/logger"
	"atlas-drops-information/monster/drop"
	"atlas-drops-information/reactor"
	drop3 "atlas-drops-information/reactor/drop"
	"atlas-drops-information/seed"
	"atlas-drops-information/service"
	"atlas-drops-information/tracing"
	"os"

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

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(drop.Migration, drop2.Migration, drop3.Migration))

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(drop.InitResource(GetServer())(db)).
		AddRouteInitializer(continent.InitResource(GetServer())(db)).
		AddRouteInitializer(reactor.InitResource(GetServer())(db)).
		AddRouteInitializer(seed.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
