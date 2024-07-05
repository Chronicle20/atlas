package main

import (
	"atlas-data/equipment/slots"
	"atlas-data/equipment/statistics"
	"atlas-data/logger"
	"atlas-data/tracing"
	"atlas-data/wz"
	"context"
	"github.com/Chronicle20/atlas-rest/server"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const serviceName = "atlas-data"

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
		prefix:  "/api/gis/",
	}
}

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}
	defer func(tc io.Closer) {
		err := tc.Close()
		if err != nil {
			l.WithError(err).Errorf("Unable to close tracer.")
		}
	}(tc)

	wzDir := os.Getenv("GAME_DATA_ROOT_DIR")

	l.Infoln("Initializing game data cache.")
	wz.GetFileCache().Init(wzDir)
	l.Infoln("Completed initializing game data cache.")

	server.CreateService(l, ctx, wg, GetServer().GetPrefix(), slots.InitResource(GetServer()), statistics.InitResource(GetServer()))

	// trap sigterm or interrupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Block until a signal is received.
	sig := <-c
	l.Infof("Initiating shutdown with signal %s.", sig)
	cancel()
	wg.Wait()
	l.Infoln("Service shutdown.")
}
