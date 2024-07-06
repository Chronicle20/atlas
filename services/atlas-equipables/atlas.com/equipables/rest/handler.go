package rest

import (
	"atlas-equipables/tenant"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strconv"
)

type HandlerDependency struct {
	l    logrus.FieldLogger
	span opentracing.Span
	db   *gorm.DB
}

func (h HandlerDependency) Logger() logrus.FieldLogger {
	return h.l
}

func (h HandlerDependency) Span() opentracing.Span {
	return h.span
}

func (h HandlerDependency) DB() *gorm.DB {
	return h.db
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

func RegisterHandler(l logrus.FieldLogger) func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler Handler) http.HandlerFunc {
	return func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler Handler) http.HandlerFunc {
		return func(si jsonapi.ServerInformation) func(handlerName string, handler Handler) http.HandlerFunc {
			return func(handlerName string, handler Handler) http.HandlerFunc {
				return RetrieveSpan(handlerName, func(span opentracing.Span) http.HandlerFunc {
					fl := l.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return ParseTenant(fl, func(tenant tenant.Model) http.HandlerFunc {
						return handler(&HandlerDependency{l: fl, span: span, db: db}, &HandlerContext{si: si, t: tenant})
					})
				})
			}
		}
	}
}

func RegisterCreateHandler[M any](l logrus.FieldLogger) func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler CreateHandler[M]) http.HandlerFunc {
	return func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler CreateHandler[M]) http.HandlerFunc {
		return func(si jsonapi.ServerInformation) func(handlerName string, handler CreateHandler[M]) http.HandlerFunc {
			return func(handlerName string, handler CreateHandler[M]) http.HandlerFunc {
				return RetrieveSpan(handlerName, func(span opentracing.Span) http.HandlerFunc {
					fl := l.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return ParseTenant(fl, func(tenant tenant.Model) http.HandlerFunc {
						d := &HandlerDependency{l: fl, db: db, span: span}
						c := &HandlerContext{si: si, t: tenant}
						return ParseInput[M](d, c, handler)
					})
				})
			}
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
