package main

import (
	custodyConsumer "atlas-parcel/kafka/consumer/custody"
	"atlas-parcel/parcel"
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-parcel"

var consumerGroupId = consumergroup.Resolve("Parcel Service")

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

	db := database.Connect(l, database.SetMigrations(parcel.Migration))
	// db is also threaded through the kafka consumer and periodic tasks that
	// later tasks in this plan register (kafka consumer, expiry sweep,
	// notification task).

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	// Periodic tasks (expiry sweep, notification) are registered by later
	// tasks in the Duey parcel-delivery plan.
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	custodyConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := custodyConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatalf("Unable to register parcel custody command handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	srv := server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(parcel.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready))

	srv.Run()

	rt.Wait()
}
