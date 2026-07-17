package main

import (
	"atlas-reward-pools/gachapon"
	"atlas-reward-pools/global"
	"atlas-reward-pools/item"
	"atlas-reward-pools/reward"
	"atlas-reward-pools/seed"
	"os"

	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-reward-pools"

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
		gachapon.Migration,
		item.Migration,
		global.Migration,
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
		AddRouteInitializer(gachapon.InitResource(GetServer())(db)).
		AddRouteInitializer(item.InitResource(GetServer())(db)).
		AddRouteInitializer(global.InitResource(GetServer())(db)).
		AddRouteInitializer(reward.InitResource(GetServer())(db)).
		AddRouteInitializer(seed.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
