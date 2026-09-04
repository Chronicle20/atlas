package rest

import (
	"fmt"
	"net/http"
	"strconv"

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

type DefinitionIdHandler func(definitionId uuid.UUID) http.HandlerFunc

func ParseDefinitionId(l logrus.FieldLogger, next DefinitionIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		definitionId, err := uuid.Parse(mux.Vars(r)["definitionId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse definitionId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(definitionId)(w, r)
	}
}

type InstanceIdHandler func(instanceId uuid.UUID) http.HandlerFunc

func ParseInstanceId(l logrus.FieldLogger, next InstanceIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instanceId, err := uuid.Parse(mux.Vars(r)["instanceId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse instanceId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(instanceId)(w, r)
	}
}

type QuestIdHandler func(questId string) http.HandlerFunc

func ParseQuestId(l logrus.FieldLogger, next QuestIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questId := mux.Vars(r)["questId"]
		if questId == "" {
			l.Errorf("Empty questId in path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(questId)(w, r)
	}
}

type CharacterIdHandler func(characterId uint32) http.HandlerFunc

func ParseCharacterId(l logrus.FieldLogger, next CharacterIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		characterIdStr := mux.Vars(r)["characterId"]
		characterId, err := strconv.Atoi(characterIdStr)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse characterId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(characterId))(w, r)
	}
}

type FieldInstanceHandler func(fieldInstance uuid.UUID) http.HandlerFunc

func ParseFieldInstance(l logrus.FieldLogger, next FieldInstanceHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fieldInstance, err := uuid.Parse(mux.Vars(r)["fieldInstance"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse fieldInstance from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(fieldInstance)(w, r)
	}
}

type MapIdHandler func(mapId uint32) http.HandlerFunc

func ParseMapId(l logrus.FieldLogger, next MapIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mapIdStr := mux.Vars(r)["mapId"]
		var mapId uint32
		_, err := fmt.Sscanf(mapIdStr, "%d", &mapId)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse mapId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(mapId)(w, r)
	}
}
