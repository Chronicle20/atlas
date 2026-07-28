package main

import (
	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/projection"
	"atlas-character-factory/factory"
	"atlas-character-factory/kafka/consumer/saga"
	"context"
	"os"
	"time"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-character-factory"

var consumerGroupId = consumergroup.Resolve("Character Factory Service")

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
	state := projection.NewState()
	caughtUp := projection.NewCaughtUp()

	rt := service.Bootstrap(serviceName,
		service.WithConfigProjection(consumerGroupId, func(t service.ProjectionTopics) service.Projection {
			sub := &projection.Subscriber{State: state, CaughtUp: caughtUp, TenantTopic: t.TenantStatus}
			return service.ProjectionFuncs{StartFunc: sub.Start, WaitCaughtUpFunc: caughtUp.WaitCaughtUp}
		}),
		service.WithReadinessGate(caughtUp.CaughtUpNow),
	)
	l := rt.Logger()

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	saga.InitConsumers(l)(cmf)(consumerGroupId)
	if err := saga.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	rt.AwaitProjectionCatchUp()
	l.Info("Configuration projection caught up.")

	// Republish projection snapshots into the configuration package vars
	// so GetTenantConfig callers (the seed saga, preset client) see live
	// updates. onChange is nil — the factory has no per-change side effects.
	routine.Go(l, rt.Context(), func(_ context.Context) {
		configuration.RunBridge(rt.Context(), l, state.Snapshot, time.Second, nil)
	})

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(factory.InitResource(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
