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

func ParseScriptId(l logrus.FieldLogger, next func(scriptId uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "scriptId", next)
}

func ParseReactorId(l logrus.FieldLogger, next func(reactorId string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "reactorId", next)
}
