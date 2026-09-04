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

type ConversationIdHandler func(conversationId uuid.UUID) http.HandlerFunc

func ParseConversationId(l logrus.FieldLogger, next ConversationIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conversationIdStr := mux.Vars(r)["conversationId"]
		conversationId, err := uuid.Parse(conversationIdStr)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse conversationId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(conversationId)(w, r)
	}
}

type NpcIdHandler func(npcId uint32) http.HandlerFunc

func ParseNpcId(l logrus.FieldLogger, next NpcIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		npcIdStr := mux.Vars(r)["npcId"]
		var npcId uint32
		_, err := fmt.Sscanf(npcIdStr, "%d", &npcId)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse npcId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(npcId)(w, r)
	}
}

type QuestIdHandler func(questId uint32) http.HandlerFunc

func ParseQuestId(l logrus.FieldLogger, next QuestIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questIdStr := mux.Vars(r)["questId"]
		var questId uint32
		_, err := fmt.Sscanf(questIdStr, "%d", &questId)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse questId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(questId)(w, r)
	}
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
