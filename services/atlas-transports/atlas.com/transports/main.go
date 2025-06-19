package main

import (
	"atlas-transports/logger"
	"atlas-transports/service"
	"atlas-transports/tracing"
	"atlas-transports/transport"
	"github.com/Chronicle20/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"os"
	"time"
)

const serviceName = "atlas-transports"

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

	ten1, _ := tenant.Create(uuid.MustParse("083839c6-c47c-42a6-9585-76492795d123"), "GMS", 83, 1)
	tenants := []tenant.Model{ten1}

	// TODO load this from a configuration source
	for _, t := range tenants {
		routes, sharedVessels := transport.LoadSampleRoutes()
		ctx := tenant.WithContext(tdm.Context(), t)
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
					transport.NewProcessor(l, tenant.WithContext(tdm.Context(), t)).UpdateStates()
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
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
