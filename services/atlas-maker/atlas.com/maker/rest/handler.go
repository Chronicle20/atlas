package rest

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
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
}

func (h HandlerContext) ServerInformation() jsonapi.ServerInformation {
	return h.si
}

type GetHandler func(d *HandlerDependency, c *HandlerContext) http.HandlerFunc

func RegisterHandler(l logrus.FieldLogger) func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
		return func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
			return func(handlerName string, handler GetHandler) http.HandlerFunc {
				return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
					fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return server.ParseEnvironment(fl, sctx, func(el logrus.FieldLogger, ectx context.Context) http.HandlerFunc {
						return server.ParseTenant(el, ectx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
							return handler(&HandlerDependency{l: tl, db: db, ctx: tctx}, &HandlerContext{si: si})
						})
					})
				})
			}
		}
	}
}

// ParseItemId extracts the {itemId} path variable as an item id.
func ParseItemId(l logrus.FieldLogger, next func(itemId item.Id) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemIdStr, ok := mux.Vars(r)["itemId"]
		if !ok {
			l.Errorf("Unable to properly parse itemId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		itemId, err := strconv.ParseUint(itemIdStr, 10, 32)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse itemId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(item.Id(itemId))(w, r)
	}
}

// ParseCharacterId extracts the {characterId} path variable.
func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

type InputHandler[M any] func(d *HandlerDependency, c *HandlerContext, model M) http.HandlerFunc

// ParseInput decodes r's JSON:API body into an M and hands it to next.
func ParseInput[M any](d *HandlerDependency, c *HandlerContext, next InputHandler[M]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var model M

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer func() { _ = r.Body.Close() }()

		if err = jsonapi.Unmarshal(body, &model); err != nil {
			d.l.WithError(err).Errorln("Deserializing input", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(d, c, model)(w, r)
	}
}

func RegisterInputHandler[M any](l logrus.FieldLogger) func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
		return func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
			return func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
				return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
					fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return server.ParseEnvironment(fl, sctx, func(el logrus.FieldLogger, ectx context.Context) http.HandlerFunc {
						return server.ParseTenant(el, ectx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
							return ParseInput[M](&HandlerDependency{l: tl, db: db, ctx: tctx}, &HandlerContext{si: si}, handler)
						})
					})
				})
			}
		}
	}
}
