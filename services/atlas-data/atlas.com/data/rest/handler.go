package rest

import (
	"context"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
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
				return server.ParseTenant(fl, sctx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
					return handler(&HandlerDependency{l: tl, ctx: tctx}, &HandlerContext{si: si})
				})
			})
		}
	}
}

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
		return func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
			return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return server.ParseTenant(fl, sctx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
					return ParseInput[M](&HandlerDependency{l: tl, ctx: tctx}, &HandlerContext{si: si}, handler)
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

type PortalIdHandler func(portalId uint32) http.HandlerFunc

func ParsePortalId(l logrus.FieldLogger, next PortalIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		portalId, err := strconv.Atoi(vars["portalId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing portalId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(portalId))(w, r)
	}
}

type NpcHandler func(npcId uint32) http.HandlerFunc

func ParseNPC(l logrus.FieldLogger, next NpcHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		npcId, err := strconv.Atoi(vars["npcId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing npcId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(npcId))(w, r)
	}
}

type MonsterIdHandler func(monsterId uint32) http.HandlerFunc

func ParseMonsterId(l logrus.FieldLogger, next MonsterIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		monsterId, err := strconv.Atoi(vars["monsterId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing monsterId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(monsterId))(w, r)
	}
}

type ReactorIdHandler func(reactorId uint32) http.HandlerFunc

func ParseReactorId(l logrus.FieldLogger, next ReactorIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		reactorId, err := strconv.Atoi(vars["reactorId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing reactorId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(reactorId))(w, r)
	}
}

type SkillIdHandler func(skillId uint32) http.HandlerFunc

func ParseSkillId(l logrus.FieldLogger, next SkillIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		skillId, err := strconv.Atoi(vars["skillId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing skillId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(skillId))(w, r)
	}
}
