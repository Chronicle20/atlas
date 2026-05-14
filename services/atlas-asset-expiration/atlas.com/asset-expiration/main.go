package main

import (
	session2 "atlas-asset-expiration/kafka/consumer/session"
	"atlas-asset-expiration/logger"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"atlas-asset-expiration/task"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
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
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	// Initialize Kafka consumers
	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())

	// Session status events consumer (for login/logout tracking)
	session2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := session2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// Start periodic expiration check task
	interval := getExpirationInterval()
	periodicTask := task.NewPeriodicTask(l, interval)
	periodicTask.Start()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))
	tdm.TeardownFunc(periodicTask.Stop)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		Run()

	tdm.Wait()
	l.Infoln("Service shutdown.")
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
