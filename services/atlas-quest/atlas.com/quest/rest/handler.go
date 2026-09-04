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

type CharacterIdHandler func(characterId uint32) http.HandlerFunc

func ParseCharacterId(l logrus.FieldLogger, next CharacterIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		characterId, err := strconv.Atoi(mux.Vars(r)["characterId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse characterId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(characterId))(w, r)
	}
}

type QuestStatusIdHandler func(questStatusId uint32) http.HandlerFunc

func ParseQuestStatusId(l logrus.FieldLogger, next QuestStatusIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questStatusId, err := strconv.Atoi(mux.Vars(r)["questStatusId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse questStatusId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(questStatusId))(w, r)
	}
}

type QuestIdHandler func(questId uint32) http.HandlerFunc

func ParseQuestId(l logrus.FieldLogger, next QuestIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questId, err := strconv.Atoi(mux.Vars(r)["questId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse questId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(questId))(w, r)
	}
}
