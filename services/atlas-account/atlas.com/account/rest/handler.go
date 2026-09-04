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

func ParseAccountId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "accountId", next)
}

// AccountIdAndWorldIdHandler is the world-scoped counterpart of
// ParseAccountId, for the character-slots sub-resource
// (accounts/{accountId}/worlds/{worldId}/character-slots, task-246
// bug-b-type-must-add-a-slot.md).
type AccountIdAndWorldIdHandler func(accountId uint32, worldId byte) http.HandlerFunc

func ParseAccountIdAndWorldId(l logrus.FieldLogger, next AccountIdAndWorldIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		accountId, err := strconv.Atoi(vars["accountId"])
		if err != nil {
			l.WithError(err).Errorln("Error parsing accountId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		worldId, err := strconv.Atoi(vars["worldId"])
		if err != nil {
			l.WithError(err).Errorln("Error parsing worldId as byte")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(accountId), byte(worldId))(w, r)
	}
}
