package rest

import (
	"atlas-account/tenant"
	"context"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strconv"
)

type HandlerDependency struct {
	l   logrus.FieldLogger
	db  *gorm.DB
	ctx context.Context
}

func (h HandlerDependency) Logger() logrus.FieldLogger {
	return h.l
}

func (h HandlerDependency) DB() *gorm.DB {
	return h.db
}

func (h HandlerDependency) Context() context.Context {
	return h.ctx
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

func RegisterHandler(l logrus.FieldLogger) func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
		return func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
			return func(handlerName string, handler GetHandler) http.HandlerFunc {
				return RetrieveSpan(l, handlerName, func(sl logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
					fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return ParseTenant(fl, func(tl logrus.FieldLogger, tenant tenant.Model) http.HandlerFunc {
						return handler(&HandlerDependency{l: tl, db: db, ctx: ctx}, &HandlerContext{si: si, t: tenant})
					})
				})
			}
		}
	}
}

func RegisterInputHandler[M any](l logrus.FieldLogger) func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
		return func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
			return func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
				return RetrieveSpan(l, handlerName, func(sl logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
					fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return ParseTenant(fl, func(tl logrus.FieldLogger, tenant tenant.Model) http.HandlerFunc {
						d := &HandlerDependency{l: tl, db: db, ctx: ctx}
						c := &HandlerContext{si: si, t: tenant}
						return ParseInput[M](d, c, handler)
					})
				})
			}
		}
	}
}

type AccountIdHandler func(id uint32) http.HandlerFunc

func ParseAccountId(l logrus.FieldLogger, next AccountIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		value, err := strconv.Atoi(vars["accountId"])
		if err != nil {
			l.WithError(err).Errorln("Error parsing id as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(value))(w, r)
	}
}
