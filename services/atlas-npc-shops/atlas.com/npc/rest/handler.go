// Package rest provides HTTP handler registration utilities for the NPC shops service.
//
// HandlerDependency, HandlerContext, GetHandler, InputHandler, RegisterHandler, and
// RegisterInputHandler are thin aliases over the shared libs/atlas-rest/server
// scaffolding, which wraps server.RetrieveSpan and server.ParseTenant to provide
// consistent tracing and multi-tenancy support. Handlers that need database access
// take a *gorm.DB parameter and close over it; resource registrars pass db per
// call site rather than currying it through the registration functions.
package rest

import (
	"net/http"

	"github.com/google/uuid"
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

func ParseNpcId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "npcId", next)
}

func ParseCommodityId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "commodityId", next)
}
