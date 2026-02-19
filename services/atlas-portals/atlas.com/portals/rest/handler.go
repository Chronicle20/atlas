package rest

import (
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

func RegisterHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return server.RegisterHandler(l)
}

func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		characterIdStr := mux.Vars(r)["characterId"]
		if characterIdStr == "" {
			characterIdStr = r.URL.Query().Get("characterId")
		}

		if characterIdStr == "" {
			l.Errorf("Unable to find characterId in path or query params.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		characterId, err := strconv.ParseUint(characterIdStr, 10, 32)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse characterId.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(characterId))(w, r)
	}
}
