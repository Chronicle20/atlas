package main

import (
	"atlas-drops-information/continent"
	drop2 "atlas-drops-information/continent/drop"
	"atlas-drops-information/logger"
	"atlas-drops-information/monster/drop"
	"atlas-drops-information/reactor"
	drop3 "atlas-drops-information/reactor/drop"
	"atlas-drops-information/seed"
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"gorm.io/gorm"
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

	db := database.Connect(l, database.SetMigrations(
		drop.Migration,
		drop2.Migration,
		drop3.Migration,
		func(db *gorm.DB) error { return db.AutoMigrate(&seeder.SeedState{}) },
	))

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
