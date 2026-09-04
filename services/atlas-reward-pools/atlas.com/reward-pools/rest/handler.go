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

func ParseGachaponId(l logrus.FieldLogger, next func(gachaponId string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "gachaponId", next)
}

func ParseItemId(l logrus.FieldLogger, next func(itemId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "itemId", next)
}
