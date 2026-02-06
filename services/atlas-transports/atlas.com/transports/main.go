package main

import (
	"atlas-transports/kafka/consumer/channel"
	"atlas-transports/kafka/consumer/character"
	"atlas-transports/logger"
	"atlas-transports/seed"
	"atlas-transports/service"
	tenant2 "atlas-transports/tenant"
	"atlas-transports/tracing"
	"atlas-transports/transport"
	"atlas-transports/transport/config"
	"os"
	"time"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas-tenant"
)

const serviceName = "atlas-transports"
const consumerGroupId = "Transport Service"

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

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	channel.InitConsumers(l)(cmf)(consumerGroupId)
	character.InitConsumers(l)(cmf)(consumerGroupId)
	channel.InitHandlers(l)(consumer.GetManager().RegisterHandler)
	character.InitHandlers(l)(consumer.GetManager().RegisterHandler)

	tenants, err := tenant2.NewProcessor(l, tdm.Context()).GetAll()
	if err != nil {
		l.WithError(err).Fatal("Unable to load tenants.")
	}

	// Load configurations from the configuration service
	configProcessor := config.NewProcessor(l, tdm.Context())
	for _, t := range tenants {
		ctx := tenant.WithContext(tdm.Context(), t)
		routes, sharedVessels, err := configProcessor.LoadConfigurationsForTenant(t)
		if err != nil {
			l.WithError(err).Errorf("Failed to load configurations for tenant [%s], using empty configuration", t.Id())
			routes = []transport.Model{}
			sharedVessels = []transport.SharedVesselModel{}
		}
		_ = transport.NewProcessor(l, ctx).AddTenant(routes, sharedVessels)
	}

	// Start a background goroutine to periodically update route states
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-tdm.Context().Done():
				return
			case <-ticker.C:
				for _, t := range tenants {
					transport.NewProcessor(l, tenant.WithContext(tdm.Context(), t)).UpdateRoutes()
				}
			}
		}
	}()

	// Create and run server
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(transport.InitResource(GetServer())).
		AddRouteInitializer(seed.InitResource(GetServer())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
