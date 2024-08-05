package rest

import (
	"atlas-maps/tenant"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
)

type HandlerDependency struct {
	l    logrus.FieldLogger
	span opentracing.Span
}

func (h HandlerDependency) Logger() logrus.FieldLogger {
	return h.l
}

func (h HandlerDependency) Span() opentracing.Span {
	return h.span
}

type HandlerContext struct {
	si jsonapi.ServerInformation
	t  tenant.Model
}

func (h HandlerContext) ServerInformation() jsonapi.ServerInformation {
	return h.si
}

func (h HandlerContext) Tenant() tenant.Model {
	return h.t
}

type GetHandler func(d *HandlerDependency, c *HandlerContext) http.HandlerFunc

type CreateHandler[M any] func(d *HandlerDependency, c *HandlerContext, model M) http.HandlerFunc

func ParseInput[M any](d *HandlerDependency, c *HandlerContext, next CreateHandler[M]) http.HandlerFunc {
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
			return RetrieveSpan(l, handlerName, func(sl logrus.FieldLogger, span opentracing.Span) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseTenant(fl, func(tenant tenant.Model) http.HandlerFunc {
					return handler(&HandlerDependency{l: fl, span: span}, &HandlerContext{si: si, t: tenant})
				})
			})
		}
	}
}

func RegisterCreateHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler CreateHandler[M]) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler CreateHandler[M]) http.HandlerFunc {
		return func(handlerName string, handler CreateHandler[M]) http.HandlerFunc {
			return RetrieveSpan(l, handlerName, func(sl logrus.FieldLogger, span opentracing.Span) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseTenant(fl, func(tenant tenant.Model) http.HandlerFunc {
					d := &HandlerDependency{l: fl, span: span}
					c := &HandlerContext{si: si, t: tenant}
					return ParseInput[M](d, c, handler)
				})
			})
		}
	}
}

type ChannelIdHandler func(channelId byte) http.HandlerFunc

func ParseChannelId(l logrus.FieldLogger, next ChannelIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channelId, err := strconv.Atoi(mux.Vars(r)["channelId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse channelId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(byte(channelId))(w, r)
	}
}

type WorldIdHandler func(worldId byte) http.HandlerFunc

func ParseWorldId(l logrus.FieldLogger, next WorldIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		worldId, err := strconv.Atoi(vars["worldId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing worldId as byte")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(byte(worldId))(w, r)
	}
}

type MapIdHandler func(mapId uint32) http.HandlerFunc

func ParseMapId(l logrus.FieldLogger, next MapIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		mapId, err := strconv.Atoi(vars["mapId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing mapId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(mapId))(w, r)
	}
}
