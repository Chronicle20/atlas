package rest

import (
	"context"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type HandlerDependency struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func (h HandlerDependency) Logger() logrus.FieldLogger {
	return h.l
}

func (h HandlerDependency) Context() context.Context {
	return h.ctx
}

type HandlerContext struct {
	si jsonapi.ServerInformation
}

func (h HandlerContext) ServerInformation() jsonapi.ServerInformation {
	return h.si
}

type GetHandler func(d *HandlerDependency, c *HandlerContext) http.HandlerFunc

type InputHandler[M any] func(d *HandlerDependency, c *HandlerContext, model M) http.HandlerFunc

func ParseInput[M any](d *HandlerDependency, c *HandlerContext, next InputHandler[M]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var model M

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = jsonapi.Unmarshal(body, &model)
		if err != nil {
			d.l.WithError(err).Errorln("Deserializing input", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(d, c, model)(w, r)
	}
}

func RegisterHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
		return func(handlerName string, handler GetHandler) http.HandlerFunc {
			return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return handler(&HandlerDependency{l: fl, ctx: sctx}, &HandlerContext{si: si})
			})
		}
	}
}

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
		return func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
			return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseInput[M](&HandlerDependency{l: fl, ctx: sctx}, &HandlerContext{si: si}, handler)
			})
		}
	}
}

type TenantIdHandler func(tenantId uuid.UUID) http.HandlerFunc

func ParseTenantId(l logrus.FieldLogger, next TenantIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantId, err := uuid.Parse(mux.Vars(r)["tenantId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse tenantId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(tenantId)(w, r)
	}
}

type RouteIdHandler func(routeId string) http.HandlerFunc

func ParseRouteId(l logrus.FieldLogger, next RouteIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routeId, ok := mux.Vars(r)["routeId"]
		if !ok {
			l.Errorf("Route ID not provided in path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(routeId)(w, r)
	}
}

type VesselIdHandler func(vesselId string) http.HandlerFunc

func ParseVesselId(l logrus.FieldLogger, next VesselIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vesselId, ok := mux.Vars(r)["vesselId"]
		if !ok {
			l.Errorf("Vessel ID not provided in path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(vesselId)(w, r)
	}
}
