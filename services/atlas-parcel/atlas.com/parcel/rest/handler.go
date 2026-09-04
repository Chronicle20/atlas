package rest

import (
	"net/http"

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

// ParseCharacterId parses the {characterId} path var into a uint32.
func ParseCharacterId(l logrus.FieldLogger, next func(characterId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

// ParseParcelId parses the {parcelId} path var (a UUID string).
func ParseParcelId(l logrus.FieldLogger, next func(parcelId string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "parcelId", next)
}
