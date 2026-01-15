package main

import (
	"atlas-storage/asset"
	"atlas-storage/database"
	"atlas-storage/kafka/consumer/character"
	"atlas-storage/kafka/consumer/compartment"
	storage2 "atlas-storage/kafka/consumer/storage"
	"atlas-storage/logger"
	"atlas-storage/projection"
	"atlas-storage/service"
	"atlas-storage/stackable"
	"atlas-storage/storage"
	"atlas-storage/tracing"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"gorm.io/gorm"
	"os"
)

const serviceName = "atlas-storage"
const consumerGroupId = "Storage Service"

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

func Migrations(db *gorm.DB) error {
	if err := storage.Migration(db); err != nil {
		return err
	}
	if err := asset.Migration(db); err != nil {
		return err
	}
	return stackable.Migration(db)
}

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(Migrations))

	// Initialize Kafka consumers for command handling
	if service.GetMode() == service.Mixed {
		cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
		storage2.InitConsumers(l)(cmf)(consumerGroupId)
		compartment.InitConsumers(l)(cmf)(consumerGroupId)
		character.InitConsumers(l)(cmf)(consumerGroupId)
		storage2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
		compartment.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
		character.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	}

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(storage.InitResource(GetServer())(db)).
		AddRouteInitializer(asset.InitResource(GetServer())(db)).
		AddRouteInitializer(projection.InitResource(GetServer())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
