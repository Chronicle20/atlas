package main

import (
	_map "atlas-monsters/kafka/consumer/map"
	"atlas-monsters/logger"
	"atlas-monsters/monster"
	"atlas-monsters/tasks"
	"atlas-monsters/tracing"
	"atlas-monsters/world"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const serviceName = "atlas-monsters"
const consumerGroupId = "Monster Registry Service"

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
		prefix:  "/api/mos/",
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
	cm.AddConsumer(l, ctx, wg)(monster.DamageConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, ctx, wg)(monster.MovementConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, ctx, wg)(_map.StatusEventConsumer(l)(consumerGroupId))
	_, _ = cm.RegisterHandler(monster.DamageCommandRegister(l))
	_, _ = cm.RegisterHandler(monster.MovementCommandRegister(l))
	_, _ = cm.RegisterHandler(_map.StatusEventCharacterEnterRegister(l))
	_, _ = cm.RegisterHandler(_map.StatusEventCharacterExitRegister(l))

	server.CreateService(l, ctx, wg, GetServer().GetPrefix(), monster.InitResource(GetServer()), world.InitResource(GetServer()))

	tasks.Register(l, ctx)(monster.NewRegistryAudit(l, time.Second*30))

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
