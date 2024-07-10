package main

import (
	"atlas-character/character"
	"atlas-character/database"
	"atlas-character/equipable"
	"atlas-character/inventory"
	"atlas-character/inventory/item"
	"atlas-character/logger"
	"atlas-character/tracing"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
)
import _ "net/http/pprof"

const serviceName = "atlas-character"
const consumerGroupId = "Character Service"

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
		prefix:  "/api/cos/",
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

	db := database.Connect(l, database.SetMigrations(character.Migration, inventory.Migration, item.Migration, equipable.Migration))

	cm := consumer.GetManager()
	cm.AddConsumer(l, ctx, wg)(inventory.EquipItemCommandConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, ctx, wg)(inventory.UnequipItemCommandConsumer(l)(consumerGroupId))
	_, _ = cm.RegisterHandler(inventory.EquipItemRegister(l, db))
	_, _ = cm.RegisterHandler(inventory.UnequipItemRegister(l, db))

	server.CreateService(l, ctx, wg, GetServer().GetPrefix(), character.InitResource(GetServer())(db), inventory.InitResource(GetServer())(db))

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
