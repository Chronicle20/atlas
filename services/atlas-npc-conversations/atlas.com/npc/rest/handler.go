package rest

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
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

func ParseConversationId(l logrus.FieldLogger, next func(conversationId uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "conversationId", next)
}

func ParseNpcId(l logrus.FieldLogger, next func(npcId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "npcId", next)
}

func ParseQuestId(l logrus.FieldLogger, next func(questId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "questId", next)
}

type ItemIdHandler func(itemId uint32) http.HandlerFunc

func ParseItemId(l logrus.FieldLogger, next ItemIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemIdStr := mux.Vars(r)["itemId"]
		var itemId uint32
		_, err := fmt.Sscanf(itemIdStr, "%d", &itemId)
		if err != nil || itemId == 0 {
			l.WithError(err).Errorf("Unable to properly parse itemId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(itemId)(w, r)
	}
}
