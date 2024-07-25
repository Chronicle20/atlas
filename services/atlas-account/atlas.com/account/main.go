package main

import (
	"atlas-account/account"
	"atlas-account/account/session"
	"atlas-account/database"
	"atlas-account/logger"
	"atlas-account/tasks"
	"atlas-account/tracing"
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

const serviceName = "atlas-account"
const consumerGroupId = "Account Service"

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
		prefix:  "/api/aos/",
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

	db := database.Connect(l, database.SetMigrations(account.Migration))

	cm := consumer.GetManager()
	cm.AddConsumer(l, ctx, wg)(account.CreateAccountCommandConsumer(l)(consumerGroupId))
	cm.AddConsumer(l, ctx, wg)(session.CreateAccountSessionCommandConsumer(l)(consumerGroupId))
	_, _ = cm.RegisterHandler(account.CreateAccountRegister(l, db))
	_, _ = cm.RegisterHandler(session.CreateAccountSessionRegister(l))

	server.CreateService(l, ctx, wg, GetServer().GetPrefix(), session.InitResource(GetServer())(db), account.InitResource(GetServer())(db))

	go tasks.Register(l, ctx)(account.NewTransitionTimeout(l, db, time.Second*time.Duration(5)))

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
