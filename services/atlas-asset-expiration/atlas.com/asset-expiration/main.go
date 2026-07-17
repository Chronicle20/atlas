package main

import (
	session2 "atlas-asset-expiration/kafka/consumer/session"
	"atlas-asset-expiration/task"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"os"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const serviceName = "atlas-asset-expiration"

var consumerGroupId = consumergroup.Resolve("Asset Expiration Service")

func main() {
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	// Initialize Kafka consumers
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())

	// Session status events consumer (for login/logout tracking)
	session2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := session2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// Start periodic expiration check task
	interval := getExpirationInterval()
	periodicTask := task.NewPeriodicTask(l, rt.Context(), interval)
	periodicTask.Start()

	rt.TeardownFunc(periodicTask.Stop)

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}

func getExpirationInterval() time.Duration {
	intervalStr := os.Getenv("EXPIRATION_CHECK_INTERVAL_SECONDS")
	if intervalStr == "" {
		return 60 * time.Second
	}
	seconds, err := strconv.Atoi(intervalStr)
	if err != nil || seconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(seconds) * time.Second
}
