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

func ParseGachaponId(l logrus.FieldLogger, next func(gachaponId string) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gachaponId, ok := mux.Vars(r)["gachaponId"]
		if !ok || gachaponId == "" {
			l.Errorf("Unable to properly parse gachaponId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(gachaponId)(w, r)
	}
}

func ParseItemId(l logrus.FieldLogger, next func(itemId uint32) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemIdStr, ok := mux.Vars(r)["itemId"]
		if !ok {
			l.Errorf("Unable to properly parse itemId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		itemId, err := strconv.Atoi(itemIdStr)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse itemId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(itemId))(w, r)
	}
}
