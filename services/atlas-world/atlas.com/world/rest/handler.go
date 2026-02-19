package rest

import (
	"net/http"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

func ParseInput[M any](d *HandlerDependency, c *HandlerContext, next InputHandler[M]) http.HandlerFunc {
	return server.ParseInput[M](d, c, next)
}

func RegisterHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return server.RegisterHandler(l)
}

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}

func ParseChannelId(l logrus.FieldLogger, next func(channel.Id) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[channel.Id](l, "channelId", next)
}

func ParseWorldId(l logrus.FieldLogger, next func(world.Id) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[world.Id](l, "worldId", next)
}
