// Package rest provides HTTP handler registration utilities for the NPC shops service.
//
// This service uses a DB-parameterized variant of RegisterHandler and RegisterInputHandler
// because handlers require database access through processors. This pattern follows the
// atlas-rest convention for services with database dependencies, using curried function
// composition to inject the database connection.
//
// The handler registration wraps server.RetrieveSpan and server.ParseTenant from atlas-rest
// to provide consistent tracing and multi-tenancy support.
//
// For services without database requirements, see atlas-parties/rest/handler.go as a reference
// for the simpler variant without the db parameter.
package rest

import (
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

type NpcIdHandler func(npcId uint32) http.HandlerFunc

func ParseNpcId(l logrus.FieldLogger, next NpcIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		npcId, err := strconv.Atoi(vars["npcId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing npcId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(npcId))(w, r)
	}
}

type CommodityIdHandler func(commodityId uuid.UUID) http.HandlerFunc

func ParseCommodityId(l logrus.FieldLogger, next CommodityIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		commodityId, err := uuid.Parse(vars["commodityId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing commodityId as uuid")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(commodityId)(w, r)
	}
}
