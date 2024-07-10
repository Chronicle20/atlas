package rest

import (
	"atlas-data/tenant"
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

type Handler func(d *HandlerDependency, c *HandlerContext) http.HandlerFunc

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

func RegisterHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler Handler) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler Handler) http.HandlerFunc {
		return func(handlerName string, handler Handler) http.HandlerFunc {
			return RetrieveSpan(l, handlerName, func(sl logrus.FieldLogger, span opentracing.Span) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseTenant(fl, func(tenant tenant.Model) http.HandlerFunc {
					return handler(&HandlerDependency{l: fl, span: span}, &HandlerContext{si: si, t: tenant})
				})
			})
		}
	}
}

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
		return func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
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
