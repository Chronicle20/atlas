package main

import (
	"atlas-world/channel"
	"atlas-world/configuration"
	"atlas-world/logger"
	"atlas-world/tracing"
	"atlas-world/world"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/opentracing/opentracing-go"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

	cm := consumer.GetManager()
	cm.AddConsumer(l, ctx, wg)(channel.EventStatusConsumer(l)(consumerGroupId))
	_, _ = cm.RegisterHandler(channel.EventStatusRegister(l))

	server.CreateService(l, ctx, wg, GetServer().GetPrefix(), channel.InitResource(GetServer()), world.InitResource(GetServer()))

	l.Infof("Service started.")
	config, err := configuration.GetConfiguration()
	if err != nil {
		l.WithError(err).Fatal("Unable to load configuration.")
	}
	span := opentracing.StartSpan("startup")
	defer span.Finish()
	channel.RequestStatus(l, span, config)

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
