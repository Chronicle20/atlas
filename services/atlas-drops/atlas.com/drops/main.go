package main

import (
	"atlas-drops/configuration"
	"atlas-drops/drop"
	drop2 "atlas-drops/kafka/consumer/drop"
	"atlas-drops/logger"
	_map "atlas-drops/map"
	"atlas-drops/service"
	"atlas-drops/tasks"
	"atlas-drops/tracing"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"os"
	"time"
)

const serviceName = "atlas-drops"
const consumerGroupId = "Drops Service"

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

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	configuration.Init(l)(tdm.Context())(uuid.MustParse(os.Getenv("SERVICE_ID")))
	config, err := configuration.GetServiceConfig()
	if err != nil {
		l.WithError(err).Fatal("Unable to successfully load configuration.")
	}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	drop2.InitConsumers(l)(cmf)(consumerGroupId)
	drop2.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(), drop.InitResource(GetServer()), _map.InitResource(GetServer()))

	tt, err := config.FindTask(drop.ExpirationTaskName)
	if err != nil {
		l.WithError(err).Fatalf("Unable to find task [%s].", drop.ExpirationTaskName)
	}
	go tasks.Register(l, tdm.Context())(drop.NewExpirationTask(l, time.Millisecond*time.Duration(tt.Interval)))

	tdm.TeardownFunc(func() {
		sctx, span := otel.GetTracerProvider().Tracer("atlas-drops").Start(context.Background(), "teardown")
		_ = model.ForEachSlice(drop.AllProvider, func(m drop.Model) error {
			tctx := tenant.WithContext(sctx, m.Tenant())
			return drop.Expire(l)(tctx)(m)
		})
		span.End()
	})
	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
