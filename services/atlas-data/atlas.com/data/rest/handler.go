package rest

import (
	"atlas-data/tenant"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
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

type Handler func(d *HandlerDependency, c *HandlerContext) http.HandlerFunc

func RegisterHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler Handler) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler Handler) http.HandlerFunc {
		return func(handlerName string, handler Handler) http.HandlerFunc {
			return RetrieveSpan(handlerName, func(span opentracing.Span) http.HandlerFunc {
				fl := l.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseTenant(fl, func(tenant tenant.Model) http.HandlerFunc {
					return handler(&HandlerDependency{l: fl, span: span}, &HandlerContext{si: si, t: tenant})
				})
			})
		}
	}
}

type EquipmentIdHandler func(equipmentId uint32) http.HandlerFunc

func ParseEquipmentId(l logrus.FieldLogger, next EquipmentIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		value, err := strconv.Atoi(vars["equipmentId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing characterId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(value))(w, r)
	}
}
