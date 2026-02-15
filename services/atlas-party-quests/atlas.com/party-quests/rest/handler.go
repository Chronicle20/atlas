package rest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
				return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
					fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return server.ParseTenant(fl, sctx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
						return handler(&HandlerDependency{l: tl, db: db, ctx: tctx}, &HandlerContext{si: si})
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
				return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
					fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return server.ParseTenant(fl, sctx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
						return ParseInput[M](&HandlerDependency{l: tl, db: db, ctx: tctx}, &HandlerContext{si: si}, handler)
					})
				})
			}
		}
	}
}

type DefinitionIdHandler func(definitionId uuid.UUID) http.HandlerFunc

func ParseDefinitionId(l logrus.FieldLogger, next DefinitionIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		definitionId, err := uuid.Parse(mux.Vars(r)["definitionId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse definitionId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(definitionId)(w, r)
	}
}

type InstanceIdHandler func(instanceId uuid.UUID) http.HandlerFunc

func ParseInstanceId(l logrus.FieldLogger, next InstanceIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instanceId, err := uuid.Parse(mux.Vars(r)["instanceId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse instanceId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(instanceId)(w, r)
	}
}

type QuestIdHandler func(questId string) http.HandlerFunc

func ParseQuestId(l logrus.FieldLogger, next QuestIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questId := mux.Vars(r)["questId"]
		if questId == "" {
			l.Errorf("Empty questId in path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(questId)(w, r)
	}
}

type CharacterIdHandler func(characterId uint32) http.HandlerFunc

func ParseCharacterId(l logrus.FieldLogger, next CharacterIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		characterIdStr := mux.Vars(r)["characterId"]
		characterId, err := strconv.Atoi(characterIdStr)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse characterId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(characterId))(w, r)
	}
}

type MapIdHandler func(mapId uint32) http.HandlerFunc

func ParseMapId(l logrus.FieldLogger, next MapIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mapIdStr := mux.Vars(r)["mapId"]
		var mapId uint32
		_, err := fmt.Sscanf(mapIdStr, "%d", &mapId)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse mapId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(mapId)(w, r)
	}
}
