package main

import (
	"fmt"
	"net/http"
	"os"

	"atlas-renders/character"
	"atlas-renders/mapr"
	"atlas-renders/storage"

	"github.com/gorilla/mux"
)

func main() {
	l := newLogger()
	s, err := storage.New(storage.ConfigFromEnv())
	if err != nil {
		l.WithError(err).Warn("storage init failed; render handlers will 503")
		s = nil
	}
	r := mux.NewRouter()
	r.HandleFunc("/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png", character.Handler(l, s)).Methods(http.MethodGet)
	r.HandleFunc("/api/wz/map/render/{tenant}/{region}/{version}/{mapId}/{kind}.png", mapr.Handler(l, s)).Methods(http.MethodGet)
	r.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})
	port := os.Getenv("REST_PORT")
	if port == "" {
		port = "8080"
	}
	l.Infof("atlas-renders listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		l.WithError(err).Fatal("server exited")
	}
}
