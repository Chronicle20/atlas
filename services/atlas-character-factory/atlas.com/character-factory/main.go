package main

import (
	"atlas-character-factory/character"
	"atlas-character-factory/factory"
	"atlas-character-factory/logger"
	"atlas-character-factory/tracing"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const serviceName = "atlas-character-factory"
const consumerGroupId = "Character Factory Service"

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
		prefix:  "/api/cfs/",
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
	cm.AddConsumer(l, ctx, wg)(character.CreatedConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, ctx, wg)(character.ItemGainedConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, ctx, wg)(character.EquipChangedConsumer(l)(consumerGroupId))

	server.CreateService(l, ctx, wg, GetServer().GetPrefix(), factory.InitResource(GetServer()))

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
