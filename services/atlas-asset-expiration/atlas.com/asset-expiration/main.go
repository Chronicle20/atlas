package main

import (
	session2 "atlas-asset-expiration/kafka/consumer/session"
	"atlas-asset-expiration/logger"
	"github.com/Chronicle20/atlas-service"
	"atlas-asset-expiration/task"
	"atlas-asset-expiration/tracing"
	"os"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-kafka/consumer"
)

const serviceName = "atlas-asset-expiration"
const consumerGroupId = "Asset Expiration Service"

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

	// Start periodic expiration check task
	interval := getExpirationInterval()
	periodicTask := task.NewPeriodicTask(l, interval)
	periodicTask.Start()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))
	tdm.TeardownFunc(periodicTask.Stop)

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
