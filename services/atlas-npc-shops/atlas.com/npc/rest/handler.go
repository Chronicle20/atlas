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
