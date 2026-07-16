package main

import (
	"atlas-rps/game"
	rpsConsumer "atlas-rps/kafka/consumer/rps"
	"atlas-rps/rest"
	"atlas-rps/tasks"
	"context"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"github.com/sirupsen/logrus"
)

// newRpsProcessor is the REST bootstrap's game.ProcessorFactory: it wires a
// real (non-shell) game.Processor per request, backed by the same
// configuration-derived LadderProvider and saga-orchestrator-backed
// SagaSubmitter the kafka command consumer uses (see
// rpsConsumer.LadderProviderFor / rpsConsumer.SagaSubmitterFor).
// game/resource.go cannot build this itself because "atlas-rps/configuration"
// imports "atlas-rps/game", so main.go - which is free to import both -
// supplies it. The REST factory never calls Collect (POST only starts a
// session), but still needs a non-nil SagaSubmitter to satisfy the
// constructor; the real one is harmless to supply here.
func newRpsProcessor(l logrus.FieldLogger, ctx context.Context) game.Processor {
	return game.NewProcessorWithLadder(l, ctx, game.DefaultThrowSource, rpsConsumer.LadderProviderFor(l, ctx), rpsConsumer.SagaSubmitterFor(l, ctx))
}

const serviceName = "atlas-rps"

var consumerGroupId = consumergroup.Resolve("RPS Service")

func main() {
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	rc := atlas.Connect(l)
	game.InitRegistry(rc)

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	rpsConsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := rpsConsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(game.NewSweepTask(l, time.Millisecond*50))
	})

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(game.InitResource(rest.GetServer(), newRpsProcessor)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
