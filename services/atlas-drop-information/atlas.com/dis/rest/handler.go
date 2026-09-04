package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

var RegisterHandler = server.RegisterHandler

type MonsterIdHandler func(monsterId uint32) http.HandlerFunc

func ParseMonsterId(l logrus.FieldLogger, next MonsterIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		monsterId, err := strconv.Atoi(mux.Vars(r)["monsterId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse monsterId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(monsterId))(w, r)
	}
}

type ItemIdHandler func(itemId uint32) http.HandlerFunc

func ParseItemId(l logrus.FieldLogger, next ItemIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemId, err := strconv.Atoi(mux.Vars(r)["itemId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse itemId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(itemId))(w, r)
	}
}
