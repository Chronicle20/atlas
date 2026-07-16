package main

import (
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	"atlas-drops/configuration"
	"atlas-drops/drop"
	drop2 "atlas-drops/kafka/consumer/drop"
	_map "atlas-drops/map"
	"atlas-drops/tasks"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

const serviceName = "atlas-drops"

var consumerGroupId = consumergroup.Resolve("Drops Service")

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
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	rc := atlas.Connect(l)
	drop.InitRegistry(rc)

	configuration.Init(l)(rt.Context())(uuid.MustParse(os.Getenv("SERVICE_ID")))
	config, err := configuration.GetServiceConfig()
	if err != nil {
		l.WithError(err).Fatal("Unable to successfully load configuration.")
	}

	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	drop2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := drop2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	// CreateRoute and run server
	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		AddRouteInitializer(drop.InitResource(GetServer())).
		AddRouteInitializer(_map.InitResource(GetServer())).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	tt, err := config.FindTask(drop.ExpirationTaskName)
	if err != nil {
		l.WithError(err).Fatalf("Unable to find task [%s].", drop.ExpirationTaskName)
	}
	routine.Go(l, rt.Context(), func(_ context.Context) {
		tasks.Register(l, rt.Context())(drop.NewExpirationTask(l, time.Millisecond*time.Duration(tt.Interval)))
	})

	rt.TeardownFunc(func() {
		sctx, span := otel.GetTracerProvider().Tracer("atlas-drops").Start(context.Background(), "teardown")
		_ = model.ForEachSlice(drop.AllProvider, func(m drop.Model) error {
			tctx := tenant.WithContext(sctx, m.Tenant())
			p := drop.NewProcessor(l, tctx)
			return p.ExpireAndEmit(m)
		})
		span.End()
	})
	rt.Wait()
}
