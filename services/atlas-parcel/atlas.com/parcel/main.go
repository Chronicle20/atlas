package main

import (
	custodyConsumer "atlas-parcel/kafka/consumer/custody"
	"atlas-parcel/parcel"
	"context"
	"os"
	"strconv"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	env "github.com/Chronicle20/atlas/libs/atlas-env"
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
	// db is also threaded through the kafka consumer and both periodic
	// tasks registered below (expiry sweep, notification sweep).

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	custodyConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := custodyConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatalf("Unable to register parcel custody command handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// DB-driven expiry sweep: expires unclaimed pending parcels and inserts
	// a return-to-sender leg for each (design §7.4 / §8).
	expiryTask := parcel.NewExpiryTask(l, rt.Context(), db, getExpirationInterval())
	expiryTask.Start()
	rt.TeardownFunc(expiryTask.Stop)

	// envContext originates this pod's own environment identity onto a
	// background task's per-tenant context before it reaches a Kafka emit
	// (mirrors services/atlas-merchant/atlas.com/merchant/main.go's own
	// envContext, task-232).
	envContext := func(ctx context.Context) context.Context {
		return env.WithContext(ctx, env.Self())
	}

	// DB-driven notification sweep: notifies recipients of newly-receivable
	// parcels via a PARCEL_ARRIVED status event and stamps LastNotified so
	// it fires at most once per parcel (design §7.1 / §8).
	notificationTask := parcel.NewNotificationTask(l, rt.Context(), db, getNotificationInterval(), envContext)
	notificationTask.Start()
	rt.TeardownFunc(notificationTask.Stop)

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

// getExpirationInterval reads the expiry sweep cadence from
// PARCEL_EXPIRY_INTERVAL_SECONDS, falling back to parcel.DefaultExpiryInterval
// when unset or invalid.
func getExpirationInterval() time.Duration {
	intervalStr := os.Getenv("PARCEL_EXPIRY_INTERVAL_SECONDS")
	if intervalStr == "" {
		return parcel.DefaultExpiryInterval
	}
	seconds, err := strconv.Atoi(intervalStr)
	if err != nil || seconds <= 0 {
		return parcel.DefaultExpiryInterval
	}
	return time.Duration(seconds) * time.Second
}

// getNotificationInterval reads the notification sweep cadence from
// PARCEL_NOTIFICATION_INTERVAL_SECONDS, falling back to
// parcel.DefaultNotificationInterval when unset or invalid.
func getNotificationInterval() time.Duration {
	intervalStr := os.Getenv("PARCEL_NOTIFICATION_INTERVAL_SECONDS")
	if intervalStr == "" {
		return parcel.DefaultNotificationInterval
	}
	seconds, err := strconv.Atoi(intervalStr)
	if err != nil || seconds <= 0 {
		return parcel.DefaultNotificationInterval
	}
	return time.Duration(seconds) * time.Second
}
