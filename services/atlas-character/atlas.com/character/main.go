package main

import (
	"context"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	"atlas-character/character"
	account2 "atlas-character/kafka/consumer/account"
	character2 "atlas-character/kafka/consumer/character"
	"atlas-character/kafka/consumer/drop"
	session2 "atlas-character/kafka/consumer/session"
	"atlas-character/saved_location"
	"atlas-character/service"
	"atlas-character/session"
	"atlas-character/session/history"
	"atlas-character/tasks"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	lifecycle "github.com/Chronicle20/atlas/libs/atlas-service"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)
import _ "net/http/pprof"

const serviceName = "atlas-character"

var consumerGroupId = consumergroup.Resolve("Character Service")

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
	rt := lifecycle.Bootstrap(serviceName)
	l := rt.Logger()

	rc := atlas.Connect(l)
	session.InitRegistry(rc)
	character.InitTemporalRegistry(rc)

	db := database.Connect(l, database.SetMigrations(character.Migration, history.Migration, saved_location.Migration, outboxlib.Migration))

	// Boot the outbox drainer: publishes the transactional outbox to Kafka.
	// Leadership is gated by a postgres advisory lock — replicas are safe.
	publisher := outboxlib.NewTopicWriterPool()
	drainer := outboxlib.NewDrainer(l, db, publisher, outboxlib.WithDSN(database.DSN()))
	routine.Go(l, rt.Context(), func(_ context.Context) {
		drainer.Run(rt.Context())
	})
	rt.TeardownFunc(func() {
		drainer.Stop()
		publisher.Close()
	})

	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})

	if service.GetMode() == service.Mixed {
		cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
		account2.InitConsumers(l)(cmf)(consumerGroupId)
		character2.InitConsumers(l)(cmf)(consumerGroupId)
		session2.InitConsumers(l)(cmf)(consumerGroupId)
		drop.InitConsumers(l)(cmf)(consumerGroupId)
		if err := account2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}
		if err := character2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}
		if err := session2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}
		if err := drop.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}

		rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	}

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(character.InitResource(GetServer())(db)).
		AddRouteInitializer(history.InitResource(GetServer())(db)).
		AddRouteInitializer(saved_location.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(session.NewTimeout(l, db, time.Millisecond*time.Duration(5000)))
	})

	rt.Wait()
}
