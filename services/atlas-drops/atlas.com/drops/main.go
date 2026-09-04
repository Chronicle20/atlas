package main

import (
	"atlas-drops/configuration"
	"atlas-drops/drop"
	"context"
	"os"
	"time"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"

	drop2 "atlas-drops/kafka/consumer/drop"
	_map "atlas-drops/map"

	service "github.com/Chronicle20/atlas/libs/atlas-service"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
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
	// drop sits outside env-domain-guard's permitted atlas-env import list
	// (main.go, kafka/, rest/, socket/), so this pod's environment identity
	// is threaded in as a plain function value rather than the package
	// importing atlas-env itself. Without it, ExpirationTask's per-tenant
	// expire Kafka events would carry an empty environment header and fail
	// decide() open per FR-1.8.
	envContext := func(ctx context.Context) context.Context {
		return env.WithContext(ctx, env.Self())
	}

	routine.Register(l, rt.Context(), rt.WaitGroup())(drop.NewExpirationTask(l, time.Millisecond*time.Duration(tt.Interval), envContext))

	rt.TeardownFunc(func() {
		sctx, span := otel.GetTracerProvider().Tracer("atlas-drops").Start(context.Background(), "teardown")
		_ = model.ForEachSlice(drop.AllProvider, func(m drop.Model) error {
			// This pod's own environment identity must be originated here
			// too: teardown emits the same real ExpireAndEmit Kafka event as
			// the periodic sweep above, and main.go is one of the sites
			// env-domain-guard permits to import atlas-env directly.
			tctx := envContext(tenant.WithContext(sctx, m.Tenant()))
			p := drop.NewProcessor(l, tctx)
			return p.ExpireAndEmit(m)
		})
		span.End()
	})
	rt.Wait()
}
