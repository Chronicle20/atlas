package main

import (
	"os"

	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-events"

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string { return s.baseUrl }
func (s Server) GetPrefix() string  { return s.prefix }
func GetServer() Server             { return Server{baseUrl: "", prefix: "/api/"} }

func main() {
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	db := database.Connect(l, database.SetMigrations(func(db *gorm.DB) error {
		return db.AutoMigrate(&seeder.SeedState{})
	}))
	_ = db

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	_ = consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
