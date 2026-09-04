package rest

import (
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

func ParseScriptId(l logrus.FieldLogger, next func(scriptId uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "scriptId", next)
}

type ScriptNameHandler func(scriptName string) http.HandlerFunc

func ParseScriptName(l logrus.FieldLogger, next ScriptNameHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scriptName := mux.Vars(r)["scriptName"]
		if scriptName == "" {
			l.Errorf("Script name is required")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(scriptName)(w, r)
	}
}
