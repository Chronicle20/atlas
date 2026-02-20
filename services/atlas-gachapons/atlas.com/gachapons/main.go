package main

import (
	database "github.com/Chronicle20/atlas-database"
	"atlas-gachapons/gachapon"
	"atlas-gachapons/global"
	"atlas-gachapons/item"
	"atlas-gachapons/logger"
	"atlas-gachapons/reward"
	"atlas-gachapons/seed"
	"github.com/Chronicle20/atlas-service"
	"atlas-gachapons/tracing"
	"os"

	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-gachapons"

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

	db := database.Connect(l, database.SetMigrations(gachapon.Migration, item.Migration, global.Migration))

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(gachapon.InitResource(GetServer())(db)).
		AddRouteInitializer(item.InitResource(GetServer())(db)).
		AddRouteInitializer(global.InitResource(GetServer())(db)).
		AddRouteInitializer(reward.InitResource(GetServer())(db)).
		AddRouteInitializer(seed.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
