package rest

import (
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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

type DropIdHandler func(dropId uint32) http.HandlerFunc

func ParseDropId(l logrus.FieldLogger, next DropIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dropId, err := strconv.Atoi(mux.Vars(r)["dropId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse dropId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(dropId))(w, r)
	}
}

type WorldIdHandler func(worldId world.Id) http.HandlerFunc

func ParseWorldId(l logrus.FieldLogger, next WorldIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		worldId, err := strconv.Atoi(vars["worldId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing worldId as byte")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(world.Id(worldId))(w, r)
	}
}

type ChannelIdHandler func(channelId channel.Id) http.HandlerFunc

func ParseChannelId(l logrus.FieldLogger, next ChannelIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channelId, err := strconv.Atoi(mux.Vars(r)["channelId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse channelId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(channel.Id(channelId))(w, r)
	}
}

type MapIdHandler func(mapId _map.Id) http.HandlerFunc

func ParseMapId(l logrus.FieldLogger, next MapIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		mapId, err := strconv.Atoi(vars["mapId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing mapId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(_map.Id(mapId))(w, r)
	}
}
