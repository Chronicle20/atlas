package main

import (
	"atlas-storage/asset"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	account2 "atlas-storage/kafka/consumer/account"
	"atlas-storage/kafka/consumer/character"
	"atlas-storage/kafka/consumer/compartment"
	storage2 "atlas-storage/kafka/consumer/storage"
	"atlas-storage/projection"
	"atlas-storage/service"
	"atlas-storage/storage"
	lifecycle "github.com/Chronicle20/atlas/libs/atlas-service"
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"gorm.io/gorm"
)

const serviceName = "atlas-storage"

var consumerGroupId = consumergroup.Resolve("Storage Service")

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
	return asset.Migration(db)
}

func main() {
	rt := lifecycle.Bootstrap(serviceName)
	l := rt.Logger()

	rc := atlas.Connect(l)
	storage.InitNpcContextCache(rc)
	projection.InitManager(rc)

	db := database.Connect(l, database.SetMigrations(Migrations))

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	// Initialize Kafka consumers for command handling
	if service.GetMode() == service.Mixed {
		cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
		account2.InitConsumers(l)(cmf)(consumerGroupId)
		storage2.InitConsumers(l)(cmf)(consumerGroupId)
		compartment.InitConsumers(l)(cmf)(consumerGroupId)
		character.InitConsumers(l)(cmf)(consumerGroupId)
		if err := account2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}
		if err := storage2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}
		if err := compartment.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}
		if err := character.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	}

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(storage.InitResource(GetServer())(db)).
		AddRouteInitializer(asset.InitResource(GetServer())(db)).
		AddRouteInitializer(projection.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
