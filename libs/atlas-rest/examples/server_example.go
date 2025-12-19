package main

import (
	"context"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.SetLevel(logrus.InfoLevel)

	ctx, cancel := context.WithCancel(context.Background())

	wg := &sync.WaitGroup{}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGHUP)

	server.New(l).
		WithContext(ctx).
		WithWaitGroup(wg).
		SetPort("4000").
		AddRouteInitializer(func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/first").Subrouter()
			r.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
				l.Infof("Received request for path: %s", r.URL.Path)
				w.WriteHeader(http.StatusOK)
			}).Methods(http.MethodGet)
		}).
		AddRouteInitializer(func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/other").Subrouter()
			r.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
				l.Infof("Received request for path: %s", r.URL.Path)
				w.WriteHeader(http.StatusOK)
			}).Methods(http.MethodGet)
		}).
		Run()

	<-sigChan
	cancel()
	wg.Wait()

}
