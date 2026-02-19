package rest

import (
	"net/http"
	"strconv"

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

func ParseAccountId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "accountId", next)
}

func ParseWorldId(l logrus.FieldLogger, next func(world.Id) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		worldIdStr := query.Get("worldId")
		if worldIdStr == "" {
			l.Errorf("Missing required worldId query parameter")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		worldIdInt, err := strconv.Atoi(worldIdStr)
		if err != nil {
			l.WithError(err).Errorf("Error parsing worldId")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(world.Id(worldIdInt))(w, r)
	}
}

func ParseAssetId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "assetId", next)
}
