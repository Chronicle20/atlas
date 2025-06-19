package main

import (
	"atlas-transports/logger"
	"atlas-transports/service"
	"atlas-transports/tracing"
	"atlas-transports/transport"
	"github.com/Chronicle20/atlas-rest/server"
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

	// Load sample routes
	routes, sharedVessels := transport.LoadSampleRoutes()
	l.Infof("Loaded %d routes and %d shared vessels", len(routes), len(sharedVessels))

	// Create processor
	processor := transport.NewProcessor(l, routes, sharedVessels)

	// Start a background goroutine to periodically update route states
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-tdm.Context().Done():
				return
			case <-ticker.C:
				processor.UpdateStates()
			}
		}
	}()

	// Create and run server
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(transport.InitResource(GetServer())(processor)).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
