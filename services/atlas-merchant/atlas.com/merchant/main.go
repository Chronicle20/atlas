package main

import (
	"atlas-merchant/frederick"
	character "atlas-merchant/kafka/consumer/character"
	compartment2 "atlas-merchant/kafka/consumer/compartment"
	merchant2 "atlas-merchant/kafka/consumer/merchant"
	"atlas-merchant/listing"
	"atlas-merchant/logger"
	"atlas-merchant/message"
	"atlas-merchant/service"
	"atlas-merchant/shop"
	"atlas-merchant/tasks"
	"atlas-merchant/tracing"
	"atlas-merchant/visitor"
	"os"

	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-kafka/consumer"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-rest/server"
)

const serviceName = "atlas-merchant"
const consumerGroupId = "Merchant Service"

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

	rc := atlas.Connect(l)
	shop.InitRegistry(rc)
	visitor.InitRegistry(rc)

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(shop.Migration, listing.Migration, message.Migration, frederick.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	merchant2.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	compartment2.InitConsumers(l)(cmf)(consumerGroupId)
	merchant2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)
	if err := character.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register character status handlers.")
	}
	compartment2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	// Start background tasks.
	tasks.Register(l, tdm.Context())(shop.NewExpirationTask(l, tdm.Context(), db, shop.DefaultExpirationInterval))
	tasks.Register(l, tdm.Context())(frederick.NewCleanupTask(l, tdm.Context(), db, frederick.DefaultCleanupInterval))
	tasks.Register(l, tdm.Context())(frederick.NewNotificationTask(l, tdm.Context(), db, frederick.DefaultNotificationInterval))

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(shop.InitializeRoutes(GetServer())(db)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
