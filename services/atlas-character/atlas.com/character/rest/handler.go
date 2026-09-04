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

type InventoryTypeHandler func(inventoryType int8) http.HandlerFunc

func ParseInventoryType(l logrus.FieldLogger, next InventoryTypeHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inventoryType, err := strconv.Atoi(mux.Vars(r)["inventoryType"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse inventoryType from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(int8(inventoryType))(w, r)
	}
}
