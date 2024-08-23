package main

import (
	"atlas-world/channel"
	"atlas-world/configuration"
	"atlas-world/logger"
	"atlas-world/service"
	"atlas-world/tracing"
	"atlas-world/world"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
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
		prefix:  "/api/wrg/",
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

	cm := consumer.GetManager()
	cm.AddConsumer(l, tdm.Context(), tdm.WaitGroup())(channel.EventStatusConsumer(l)(consumerGroupId))
	_, _ = cm.RegisterHandler(channel.EventStatusRegister(l))

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), channel.InitResource(GetServer()), world.InitResource(GetServer()))

	l.Infof("Service started.")
	config, err := configuration.GetConfiguration()
	if err != nil {
		l.WithError(err).Fatal("Unable to load configuration.")
	}

	ctx, span := otel.GetTracerProvider().Tracer(serviceName).Start(context.Background(), "startup")
	channel.RequestStatus(l, ctx, config)
	span.End()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
