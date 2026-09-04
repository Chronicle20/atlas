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

func ParseDefinitionId(l logrus.FieldLogger, next func(definitionId uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "definitionId", next)
}

func ParseInstanceId(l logrus.FieldLogger, next func(instanceId uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "instanceId", next)
}

func ParseQuestId(l logrus.FieldLogger, next func(questId string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "questId", next)
}

func ParseCharacterId(l logrus.FieldLogger, next func(characterId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

func ParseFieldInstance(l logrus.FieldLogger, next func(fieldInstance uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "fieldInstance", next)
}

func ParseMapId(l logrus.FieldLogger, next func(mapId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "mapId", next)
}
