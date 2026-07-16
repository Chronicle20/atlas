package main

import (
	"atlas-drops-information/continent"
	drop2 "atlas-drops-information/continent/drop"
	"atlas-drops-information/monster/drop"
	"atlas-drops-information/reactor"
	drop3 "atlas-drops-information/reactor/drop"
	"atlas-drops-information/seed"
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
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
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	db := database.Connect(l, database.SetMigrations(
		drop.Migration,
		drop2.Migration,
		drop3.Migration,
		func(db *gorm.DB) error { return db.AutoMigrate(&seeder.SeedState{}) },
	))

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(drop.InitResource(GetServer())(db)).
		AddRouteInitializer(continent.InitResource(GetServer())(db)).
		AddRouteInitializer(reactor.InitResource(GetServer())(db)).
		AddRouteInitializer(seed.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
