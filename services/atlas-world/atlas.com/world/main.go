package main

import (
	"atlas-world/channel"
	"atlas-world/configuration"
	channel2 "atlas-world/kafka/consumer/channel"
	"atlas-world/logger"
	"atlas-world/rate"
	"atlas-world/service"
	"atlas-world/tasks"
	"atlas-world/tracing"
	"atlas-world/world"
	"context"
	"os"
	"time"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

const serviceName = "atlas-world"
const consumerGroupId = "World Orchestrator"

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
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	channel2.InitConsumers(l)(cmf)(consumerGroupId)
	channel2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(channel.InitResource(GetServer())).
		AddRouteInitializer(world.InitResource(GetServer())).
		AddRouteInitializer(rate.InitResource(GetServer())).
		Run()

	l.Infof("Service started.")
	configuration.Init(l)(tdm.Context())(uuid.MustParse(os.Getenv("SERVICE_ID")))

	ctx, span := otel.GetTracerProvider().Tracer(serviceName).Start(context.Background(), "startup")
	_ = model.ForEachMap(model.FixedProvider(configuration.GetTenantConfigs()), channel.RequestStatus(l)(ctx))
	span.End()

	go tasks.Register(l, tdm.Context())(channel.NewExpiration(l, tdm.Context(), time.Second*10))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
