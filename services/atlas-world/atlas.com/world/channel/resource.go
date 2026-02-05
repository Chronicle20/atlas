package channel

import (
	"atlas-world/rest"
	"errors"
	"net/http"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/worlds/{worldId}/channels").Subrouter()
		r.HandleFunc("", registerGet("get_channel_servers", handleGetChannelServers)).Methods(http.MethodGet)
		r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("register_channel_server", handleRegisterChannelServer)).Methods(http.MethodPost)
		r.HandleFunc("/{channelId}", registerGet("get_channel", handleGetChannel)).Methods(http.MethodGet)
	}
}

func handleGetChannelServers(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cs, err := NewProcessor(d.Logger(), d.Context()).GetByWorld(worldId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get all channel servers.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			rm, err := model.SliceMap(Transform)(model.FixedProvider(cs))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

func handleRegisterChannelServer(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context()).EmitStartedAndEmit(worldId, input.ChannelId, input.IpAddress, input.Port, input.CurrentCapacity, input.MaxCapacity)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to emit channel started event.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		}
	})
}

func handleGetChannel(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ch, err := NewProcessor(d.Logger(), d.Context()).GetById(worldId, channelId)
				if err != nil {
					if errors.Is(err, ErrChannelNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}

					d.Logger().WithError(err).Errorf("Unable to get channel.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				rm, err := model.Map(Transform)(model.FixedProvider(ch))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	})
}
