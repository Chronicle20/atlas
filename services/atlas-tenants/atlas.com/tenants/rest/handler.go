package rest

import (
	"atlas-tenants/scope"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

func ParseInput[M any](d *HandlerDependency, c *HandlerContext, next InputHandler[M]) http.HandlerFunc {
	return server.ParseInput[M](d, c, next)
}

func RegisterHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return server.RegisterSimpleHandler(l)
}

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterSimpleInputHandler[M](l)
}

// WriteErrorResponse wraps server.WriteErrorResponse to surface
// scope.ErrCrossEnvironmentWrite as 403 Forbidden (task-232 FR-7.8) instead
// of the generic 500 server.WriteErrorResponse would otherwise produce.
// Every other error falls through unchanged.
func WriteErrorResponse(l logrus.FieldLogger) func(w http.ResponseWriter) func(err error) {
	return func(w http.ResponseWriter) func(err error) {
		return func(err error) {
			if errors.Is(err, scope.ErrCrossEnvironmentWrite) {
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"errors": []map[string]any{{
						"status": strconv.Itoa(http.StatusForbidden),
						"title":  "cross-environment write",
						"detail": err.Error(),
					}},
				})
				return
			}
			server.WriteErrorResponse(l)(w)(err)
		}
	}
}

func ParseTenantId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "tenantId", next)
}

func ParseRouteId(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "routeId", next)
}

func ParseVesselId(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "vesselId", next)
}

func ParseInstanceRouteId(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "instanceRouteId", next)
}

func ParseRpsRewardId(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "rpsRewardId", next)
}

func ParseMtsConfigId(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "mtsConfigId", next)
}

func ParseTradeConfigId(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "tradeConfigId", next)
}

func ParseImprintConfigId(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "imprintConfigId", next)
}
