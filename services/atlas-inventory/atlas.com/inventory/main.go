package main

import (
	"atlas-inventory/asset"
	"atlas-inventory/compartment"
	"atlas-inventory/database"
	"atlas-inventory/inventory"
	"atlas-inventory/logger"
	"atlas-inventory/service"
	"atlas-inventory/tracing"
	"os"

	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-inventory"

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

	db := database.Connect(l, database.SetMigrations(compartment.Migration, asset.Migration))

	// cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(inventory.InitResource(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
