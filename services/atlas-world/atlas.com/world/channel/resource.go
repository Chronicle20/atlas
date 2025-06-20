package channel

import (
	channel2 "atlas-world/kafka/message/channel"
	"atlas-world/kafka/producer"
	channel3 "atlas-world/kafka/producer/channel"
	"atlas-world/rest"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
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
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
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
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			_ = producer.ProviderImpl(d.Logger())(d.Context())(channel2.EnvEventTopicStatus)(channel3.StartedEventProvider(t, worldId, input.ChannelId, input.IpAddress, input.Port, input.CurrentCapacity, input.MaxCapacity))
			w.WriteHeader(http.StatusAccepted)
		}
	})
}

func handleGetChannel(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId byte) http.HandlerFunc {
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
