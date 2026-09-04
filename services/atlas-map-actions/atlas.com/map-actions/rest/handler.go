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

type ScriptIdHandler func(scriptId uuid.UUID) http.HandlerFunc

func ParseScriptId(l logrus.FieldLogger, next ScriptIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scriptIdStr := mux.Vars(r)["scriptId"]
		scriptId, err := uuid.Parse(scriptIdStr)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse scriptId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(scriptId)(w, r)
	}
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
