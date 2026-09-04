package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}

// ParseCharacterId parses the {characterId} path var into a uint32.
func ParseCharacterId(l logrus.FieldLogger, next func(characterId uint32) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		characterIdStr, ok := mux.Vars(r)["characterId"]
		if !ok {
			l.Errorf("Unable to properly parse characterId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		characterId, err := strconv.ParseUint(characterIdStr, 10, 32)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse characterId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(characterId))(w, r)
	}
}

// ParseParcelId parses the {parcelId} path var (a UUID string).
func ParseParcelId(l logrus.FieldLogger, next func(parcelId string) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parcelId, ok := mux.Vars(r)["parcelId"]
		if !ok || parcelId == "" {
			l.Errorf("Unable to properly parse parcelId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(parcelId)(w, r)
	}
}
