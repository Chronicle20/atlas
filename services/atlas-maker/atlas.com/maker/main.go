package main

import (
	"os"

	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-maker"

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
	rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
	l := rt.Logger()

	db := database.Connect(l, database.SetMigrations(
		func(db *gorm.DB) error { return db.AutoMigrate(&seeder.SeedState{}) },
	))
	// db is unused until Tasks 17-18 add the reagent/crystal-band route
	// initializers that need it; this line goes away when the first one
	// lands (atlas-events' task-231 scaffold, aedbda59d, is the precedent).
	_ = db

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
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
