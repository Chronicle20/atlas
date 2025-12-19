package reactor

import (
	"atlas-reactors/kafka/producer"
	"atlas-reactors/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/reactors").Subrouter()
		r.HandleFunc("/{reactorId}", registerGet("get_by_id", handleGetById)).Methods(http.MethodGet)

		r = router.PathPrefix("/worlds/{worldId}/channels/{channelId}/maps/{mapId}/reactors").Subrouter()
		r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("create_in_map", handleCreateInMap)).Methods(http.MethodPost)
		r.HandleFunc("", registerGet("get_in_map", handleGetInMap)).Methods(http.MethodGet)
		r.HandleFunc("/{reactorId}", registerGet("get_by_id", handleGetByIdInMap)).Methods(http.MethodGet)
	}
}

func handleGetById(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseReactorId(d.Logger(), func(reactorId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := GetById(d.Logger())(d.Context())(reactorId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(m))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetByIdInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId byte) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
				return rest.ParseReactorId(d.Logger(), func(reactorId uint32) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						m, err := GetById(d.Logger())(d.Context())(reactorId)
						if err != nil || m.WorldId() != worldId || m.ChannelId() != channelId || m.MapId() != mapId {
							w.WriteHeader(http.StatusNotFound)
							return
						}

						res, err := model.Map(Transform)(model.FixedProvider(m))()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
					}
				})
			})
		})
	})
}

func handleCreateInMap(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId byte) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					err := producer.ProviderImpl(d.Logger())(d.Context())(EnvCommandTopic)(createCommandProvider(worldId, channelId, mapId, i.Classification, i.Name, i.State, i.X, i.Y, i.Delay, i.Direction))
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to accept reactor creation request for processing.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusAccepted)
				}
			})
		})
	})
}

func handleGetInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId byte) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					ms, err := GetInMap(d.Logger())(d.Context())(worldId, channelId, mapId)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					res, err := model.SliceMap(Transform)(model.FixedProvider(ms))()()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
				}
			})
		})
	})
}
